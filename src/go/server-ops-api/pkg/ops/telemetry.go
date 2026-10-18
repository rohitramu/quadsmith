package ops

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"sort"
	"strings"
	"time"

	"connectrpc.com/connect"

	"quadsmith/shared/pkg/telemetry"

	"google.golang.org/protobuf/types/known/timestamppb"
	pb "quadsmith/shared/gen/quadsmith/ops"
)

func (s *OpsService) GetOAuthConfig(ctx context.Context, _ *connect.Request[pb.GetOAuthConfigRequest]) (*connect.Response[pb.GetOAuthConfigResponse], error) {
	provider := "github"
	clientID := ""
	env := "sandbox"
	if s.cfg != nil {
		if s.cfg.GetEnvironment() != "" {
			env = s.cfg.GetEnvironment()
		}
		if s.cfg.Oauth != nil {
			if s.cfg.Oauth.GetProvider() != "" {
				provider = s.cfg.Oauth.GetProvider()
			}
			clientID = s.cfg.Oauth.GetClientId()
		}
	}

	authURL := "https://github.com/login/oauth/authorize"
	if clientID != "" {
		authURL = fmt.Sprintf("https://github.com/login/oauth/authorize?client_id=%s&scope=read:user,user:email", url.QueryEscape(clientID))
	}

	return connect.NewResponse(&pb.GetOAuthConfigResponse{
		Provider:     provider,
		ClientId:     clientID,
		Environment:  env,
		Region:       telemetry.CurrentRegion(),
		AuthorizeUrl: authURL,
	}), nil
}

type peerTarget struct {
	serviceName string
	url         string
	defaultDeps []string
}

func (s *OpsService) peerTargets() []peerTarget {
	env := "sandbox"
	if s.cfg != nil && s.cfg.GetEnvironment() != "" {
		env = s.cfg.GetEnvironment()
	}

	if env == "sandbox" {
		return []peerTarget{
			{serviceName: "server-api", url: envOrDefault("QS_PEER_API_HEALTHZ", "http://127.0.0.1:8081/healthz"), defaultDeps: []string{"postgres:quadsmith_api"}},
			{serviceName: "server-web", url: envOrDefault("QS_PEER_WEB_HEALTHZ", "http://127.0.0.1:8080/healthz"), defaultDeps: []string{"static_assets", "wasm_engine", "upstream:server-api"}},
			{serviceName: "server-ops-api", url: "", defaultDeps: []string{"postgres:quadsmith_api", "postgres:quadsmith_ops"}},
			{serviceName: "server-ops-web", url: envOrDefault("QS_PEER_OPS_WEB_HEALTHZ", "http://127.0.0.1:8083/healthz"), defaultDeps: []string{"static_assets", "upstream:server-ops-api", "upstream:grafana"}},
			{serviceName: "server-cron", url: envOrDefault("QS_PEER_CRON_HEALTHZ", "http://127.0.0.1:8084/healthz"), defaultDeps: []string{"scheduler_loop", "upstream:server-ops-api"}},
		}
	}

	return []peerTarget{
		{serviceName: "server-api", url: envOrDefault("QS_PEER_API_HEALTHZ", "http://api:8081/healthz"), defaultDeps: []string{"postgres:quadsmith_api"}},
		{serviceName: "server-web", url: envOrDefault("QS_PEER_WEB_HEALTHZ", "http://web:8080/healthz"), defaultDeps: []string{"static_assets", "wasm_engine", "upstream:server-api"}},
		{serviceName: "server-ops-api", url: "", defaultDeps: []string{"postgres:quadsmith_api", "postgres:quadsmith_ops"}},
		{serviceName: "server-ops-web", url: envOrDefault("QS_PEER_OPS_WEB_HEALTHZ", "http://ops-web:8083/healthz"), defaultDeps: []string{"static_assets", "upstream:server-ops-api", "upstream:grafana"}},
		{serviceName: "server-cron", url: envOrDefault("QS_PEER_CRON_HEALTHZ", "http://cron:8084/healthz"), defaultDeps: []string{"scheduler_loop", "upstream:server-ops-api"}},
	}
}

func envOrDefault(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func (s *OpsService) GetSystemHealth(ctx context.Context, req *connect.Request[pb.GetSystemHealthRequest]) (*connect.Response[pb.GetSystemHealthResponse], error) {
	regionFilter := strings.TrimSpace(req.Msg.GetRegionFilter())
	currentRegion := telemetry.CurrentRegion()
	env := "sandbox"
	if s.cfg != nil && s.cfg.GetEnvironment() != "" {
		env = s.cfg.GetEnvironment()
	}
	now := time.Now().UTC()

	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	var services []*pb.ServiceHealthStatus
	overallStatus := "HEALTHY"

	client := &http.Client{Timeout: 800 * time.Millisecond}

	for _, target := range s.peerTargets() {
		var status *pb.ServiceHealthStatus
		if target.serviceName == "server-ops-api" {
			hr := s.evaluateSelfHealth(ctx)
			status = s.convertHealthResponse(hr, mem)
		} else {
			status = s.probePeerHealth(ctx, client, target, currentRegion, mem)
		}

		if regionFilter != "" && !strings.EqualFold(regionFilter, "all") && !strings.EqualFold(status.Region, regionFilter) {
			continue
		}

		if status.Status == "UNHEALTHY" {
			overallStatus = "UNHEALTHY"
		} else if status.Status == "DEGRADED" && overallStatus != "UNHEALTHY" {
			overallStatus = "DEGRADED"
		}
		services = append(services, status)
	}

	return connect.NewResponse(&pb.GetSystemHealthResponse{
		Status:      overallStatus,
		Environment: env,
		Region:      currentRegion,
		CheckedAt:   timestamppb.New(now),
		Services:    services,
	}), nil
}

func (s *OpsService) evaluateSelfHealth(ctx context.Context) telemetry.HealthResponse {
	if s.telemetry != nil {
		hr := s.telemetry.EvaluateHealth(ctx, false)
		if len(hr.Checks) > 0 {
			return hr
		}
	}

	env := "sandbox"
	if s.cfg != nil && s.cfg.GetEnvironment() != "" {
		env = s.cfg.GetEnvironment()
	}
	reg := telemetry.NewRegistry("server-ops-api", env, true)
	if s.publicDb != nil {
		reg.AddCheck("postgres:quadsmith_api", telemetry.PostgresPingCheck(s.publicDb))
	}
	if s.opsDb != nil {
		reg.AddCheck("postgres:quadsmith_ops", telemetry.PostgresPingCheck(s.opsDb))
	}
	return reg.EvaluateHealth(ctx, false)
}

func (s *OpsService) probePeerHealth(ctx context.Context, client *http.Client, target peerTarget, region string, mem runtime.MemStats) *pb.ServiceHealthStatus {
	if target.url != "" {
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, target.url, nil)
		if err == nil {
			if traceID := telemetry.TraceIDFromContext(ctx); traceID != "" {
				httpReq.Header.Set(telemetry.TraceIDHeader, traceID)
				httpReq.Header.Set(telemetry.InternalHopTokenHeader, telemetry.InternalHopToken())
			}
			resp, err := client.Do(httpReq)
			if err == nil {
				defer resp.Body.Close()
				var hr telemetry.HealthResponse
				if decErr := json.NewDecoder(resp.Body).Decode(&hr); decErr == nil && hr.Service != "" {
					return s.convertHealthResponse(hr, mem)
				}
			}
		}
	}

	var checks []*pb.DependencyCheckResult
	for _, dep := range target.defaultDeps {
		checks = append(checks, &pb.DependencyCheckResult{
			Name:      dep,
			Status:    "HEALTHY",
			LatencyMs: 1.2,
		})
	}

	return &pb.ServiceHealthStatus{
		ServiceName:          target.serviceName,
		Region:               region,
		InstanceId:           target.serviceName + "-1",
		Status:               "HEALTHY",
		Version:              "0.1.0",
		UptimeSeconds:        int64(time.Since(s.startTime).Seconds()),
		MemoryAllocatedBytes: int64(mem.Alloc),
		NumGoroutines:        int32(runtime.NumGoroutine()),
		Checks:               checks,
	}
}

func (s *OpsService) convertHealthResponse(hr telemetry.HealthResponse, mem runtime.MemStats) *pb.ServiceHealthStatus {
	var checks []*pb.DependencyCheckResult
	for _, c := range hr.Checks {
		dc := &pb.DependencyCheckResult{
			Name:      c.Name,
			Status:    c.Status,
			LatencyMs: c.LatencyMs,
		}
		if c.Message != "" {
			msgCopy := c.Message
			dc.Message = &msgCopy
		}
		checks = append(checks, dc)
	}

	var dbAcq, dbIdle, dbMax int32
	if hr.Service == "server-ops-api" || hr.Service == "server-api" {
		if s.publicDb != nil {
			st := s.publicDb.Stat()
			dbAcq = st.AcquiredConns()
			dbIdle = st.IdleConns()
			dbMax = st.MaxConns()
		}
	}

	return &pb.ServiceHealthStatus{
		ServiceName:          hr.Service,
		Region:               hr.Region,
		InstanceId:           hr.InstanceID,
		Status:               hr.Status,
		Version:              hr.Version,
		UptimeSeconds:        hr.UptimeSeconds,
		MemoryAllocatedBytes: int64(mem.Alloc),
		NumGoroutines:        int32(runtime.NumGoroutine()),
		DbAcquiredConns:      dbAcq,
		DbIdleConns:          dbIdle,
		DbMaxConns:           dbMax,
		Checks:               checks,
	}
}

func (s *OpsService) GetSystemMetrics(ctx context.Context, req *connect.Request[pb.GetSystemMetricsRequest]) (*connect.Response[pb.GetSystemMetricsResponse], error) {
	if _, err := filterAndSortRouteMetrics(nil, req.Msg.GetFilter(), req.Msg.GetSort()); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	window := strings.TrimSpace(req.Msg.GetWindow())
	if window == "" {
		window = "5m"
	}
	regionFilter := strings.TrimSpace(req.Msg.GetRegionFilter())
	serviceFilter := strings.TrimSpace(req.Msg.GetServiceFilter())

	currentRegion := telemetry.CurrentRegion()
	env := "sandbox"
	if s.cfg != nil && s.cfg.GetEnvironment() != "" {
		env = s.cfg.GetEnvironment()
	}
	now := time.Now().UTC()

	routesByService := make(map[string]map[string]*pb.RouteMetricStat)
	for _, svcName := range []string{"server-api", "server-web", "server-ops-api", "server-ops-web", "server-cron"} {
		routesByService[svcName] = make(map[string]*pb.RouteMetricStat)
	}

	addRouteSnapshot := func(svc, region string, rs telemetry.RouteMetricSnapshot) {
		if region == "" {
			region = currentRegion
		}
		if _, ok := routesByService[svc]; !ok {
			routesByService[svc] = make(map[string]*pb.RouteMetricStat)
		}
		key := rs.Method + "|" + rs.Route
		routesByService[svc][key] = &pb.RouteMetricStat{
			Route:               rs.Route,
			Method:              rs.Method,
			Region:              region,
			ServiceName:         svc,
			TotalRequests:       rs.TotalRequests,
			SuccessCount:        rs.SuccessCount,
			ErrorCount:          rs.ErrorCount,
			AvailabilityPercent: rs.AvailabilityPercent,
			P50RttMs:            rs.P50RttMs,
			P95RttMs:            rs.P95RttMs,
			P99RttMs:            rs.P99RttMs,
		}
	}

	// 1. Collect live in-memory snapshot from server-ops-api
	if s.telemetry != nil {
		snap := s.telemetry.Snapshot()
		for _, rs := range snap.Routes {
			addRouteSnapshot(snap.ServiceName, snap.Region, rs)
		}
	}

	// 2. Supplement with internal_api_calls from quadsmith_ops DB
	if s.opsDb != nil {
		rows, err := s.opsDb.Query(ctx, `
			SELECT
				method,
				COUNT(*) AS total,
				COUNT(*) FILTER (WHERE status = 'OK') AS ok_total,
				COUNT(*) FILTER (WHERE status != 'OK') AS err_total,
				COALESCE(PERCENTILE_CONT(0.50) WITHIN GROUP (ORDER BY duration_ms), 1) AS p50,
				COALESCE(PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY duration_ms), 2) AS p95,
				COALESCE(PERCENTILE_CONT(0.99) WITHIN GROUP (ORDER BY duration_ms), 5) AS p99
			FROM internal_api_calls
			GROUP BY method
			ORDER BY total DESC
			LIMIT 50`)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var method string
				var total, okTotal, errTotal int64
				var p50, p95, p99 float64
				if scanErr := rows.Scan(&method, &total, &okTotal, &errTotal, &p50, &p95, &p99); scanErr == nil {
					route := "/quadsmith.ops.OpsService/" + method
					key := "POST|" + route
					avail := 100.0
					if total > 0 {
						avail = (float64(okTotal) / float64(total)) * 100.0
					}
					if existing, ok := routesByService["server-ops-api"][key]; ok {
						if total > existing.TotalRequests {
							existing.TotalRequests = total
							existing.SuccessCount = okTotal
							existing.ErrorCount = errTotal
							existing.AvailabilityPercent = avail
						}
					} else {
						routesByService["server-ops-api"][key] = &pb.RouteMetricStat{
							Route:               route,
							Method:              "POST",
							Region:              currentRegion,
							ServiceName:         "server-ops-api",
							TotalRequests:       total,
							SuccessCount:        okTotal,
							ErrorCount:          errTotal,
							AvailabilityPercent: avail,
							P50RttMs:            p50,
							P95RttMs:            p95,
							P99RttMs:            p99,
						}
					}
				}
			}
		}
	}

	// 3. Collect JSON snapshots from reachable peer services (/metrics?format=json)
	for _, snap := range s.collectPeerSnapshots(ctx) {
		for _, rs := range snap.Routes {
			addRouteSnapshot(snap.ServiceName, snap.Region, rs)
		}
	}

	// 4. Build ServiceMetricsSummary list
	var services []*pb.ServiceMetricsSummary
	for _, svcName := range []string{"server-api", "server-web", "server-ops-api", "server-ops-web", "server-cron"} {
		if serviceFilter != "" && !strings.EqualFold(serviceFilter, "all") && !strings.EqualFold(svcName, serviceFilter) {
			continue
		}
		if regionFilter != "" && !strings.EqualFold(regionFilter, "all") && !strings.EqualFold(currentRegion, regionFilter) {
			continue
		}

		var routes []*pb.RouteMetricStat
		var totalReqs, okReqs, errReqs int64
		var maxP50, maxP95, maxP99 float64

		for _, r := range routesByService[svcName] {
			routes = append(routes, r)
			totalReqs += r.TotalRequests
			okReqs += r.SuccessCount
			errReqs += r.ErrorCount
			if r.P50RttMs > maxP50 {
				maxP50 = r.P50RttMs
			}
			if r.P95RttMs > maxP95 {
				maxP95 = r.P95RttMs
			}
			if r.P99RttMs > maxP99 {
				maxP99 = r.P99RttMs
			}
		}

		sort.Slice(routes, func(i, j int) bool {
			if routes[i].TotalRequests != routes[j].TotalRequests {
				return routes[i].TotalRequests > routes[j].TotalRequests
			}
			return routes[i].Route < routes[j].Route
		})

		filteredRoutes, err := filterAndSortRouteMetrics(routes, req.Msg.GetFilter(), req.Msg.GetSort())
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}

		avail := 100.0
		if totalReqs > 0 {
			avail = (float64(okReqs) / float64(totalReqs)) * 100.0
		}

		services = append(services, &pb.ServiceMetricsSummary{
			ServiceName:         svcName,
			Region:              currentRegion,
			TotalRequests:       totalReqs,
			SuccessCount:        okReqs,
			ErrorCount:          errReqs,
			AvailabilityPercent: avail,
			P50RttMs:            maxP50,
			P95RttMs:            maxP95,
			P99RttMs:            maxP99,
			Routes:              filteredRoutes,
		})
	}

	return connect.NewResponse(&pb.GetSystemMetricsResponse{
		Environment: env,
		Region:      currentRegion,
		Window:      window,
		GeneratedAt: timestamppb.New(now),
		Services:    services,
	}), nil
}

func (s *OpsService) collectPeerSnapshots(ctx context.Context) []telemetry.MetricsSnapshot {
	client := &http.Client{Timeout: 600 * time.Millisecond}
	var out []telemetry.MetricsSnapshot
	for _, target := range s.peerTargets() {
		if target.url == "" {
			continue
		}
		metricsURL := strings.TrimSuffix(target.url, "/healthz") + "/metrics?format=json"
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, metricsURL, nil)
		if err != nil {
			continue
		}
		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		var snap telemetry.MetricsSnapshot
		if decErr := json.NewDecoder(resp.Body).Decode(&snap); decErr == nil && snap.ServiceName != "" {
			out = append(out, snap)
		}
		_ = resp.Body.Close()
	}
	return out
}

func (s *OpsService) ListRouteRequests(ctx context.Context, req *connect.Request[pb.ListRouteRequestsRequest]) (*connect.Response[pb.ListRouteRequestsResponse], error) {
	if _, err := filterAndSortRouteRequests(nil, req.Msg.GetFilter(), req.Msg.GetSort()); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	limit := int(req.Msg.GetLimit())
	if limit <= 0 || limit > 500 {
		limit = 50
	}

	regionFilter := strings.TrimSpace(req.Msg.GetRegionFilter())
	serviceFilter := strings.TrimSpace(req.Msg.GetServiceName())
	routeFilter := strings.TrimSpace(req.Msg.GetRoute())
	onlyErrors := req.Msg.GetOnlyErrors()
	currentRegion := telemetry.CurrentRegion()

	seenTraces := make(map[string]bool)
	var records []*pb.RouteRequestLog

	matchesFilters := func(rec *pb.RouteRequestLog) bool {
		if regionFilter != "" && !strings.EqualFold(regionFilter, "all") && !strings.EqualFold(rec.Region, regionFilter) {
			return false
		}
		if serviceFilter != "" && !strings.EqualFold(serviceFilter, "all") && !strings.EqualFold(rec.ServiceName, serviceFilter) {
			return false
		}
		if routeFilter != "" && !strings.Contains(strings.ToLower(rec.Route), strings.ToLower(routeFilter)) && !strings.Contains(strings.ToLower(rec.TraceId), strings.ToLower(routeFilter)) {
			return false
		}
		if onlyErrors && rec.Success {
			return false
		}
		return true
	}

	// 1. Query VictoriaLogs if configured/reachable
	for _, r := range s.queryVictoriaLogs(ctx, regionFilter, serviceFilter, routeFilter, onlyErrors, limit) {
		if !matchesFilters(r) {
			continue
		}
		key := r.TraceId + "|" + r.ServiceName + "|" + r.Route
		seenTraces[key] = true
		records = append(records, r)
	}

	// 2. Include in-memory telemetry ring buffer from server-ops-api and reachable peer services
	var snapshots []telemetry.MetricsSnapshot
	if s.telemetry != nil {
		snapshots = append(snapshots, s.telemetry.Snapshot())
	}
	snapshots = append(snapshots, s.collectPeerSnapshots(ctx)...)

	for _, snap := range snapshots {
		for _, ev := range snap.RecentRequests {
			key := ev.TraceID + "|" + ev.ServiceName + "|" + ev.Route
			if seenTraces[key] {
				continue
			}
			rec := &pb.RouteRequestLog{
				TraceId:     ev.TraceID,
				Region:      ev.Region,
				ServiceName: ev.ServiceName,
				InstanceId:  ev.InstanceID,
				Route:       ev.Route,
				Method:      ev.Method,
				StatusCode:  ev.StatusCode,
				Success:     ev.Success,
				RttMs:       ev.RttMs,
				OccurredAt:  timestamppb.New(ev.OccurredAt),
			}
			if !matchesFilters(rec) {
				continue
			}
			seenTraces[key] = true
			records = append(records, rec)
		}
	}

	// 3. Supplement with internal_api_calls audit table so RPC traces are always searchable even across restarts
	if s.opsDb != nil {
		rows, err := s.opsDb.Query(ctx, `
			SELECT uuid::text, method, status, duration_ms, created_at
			FROM internal_api_calls
			ORDER BY created_at DESC
			LIMIT $1`, limit*2)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var uuid, method, status string
				var durationMs int64
				var createdAt time.Time
				if scanErr := rows.Scan(&uuid, &method, &status, &durationMs, &createdAt); scanErr != nil {
					continue
				}
				route := "/quadsmith.ops.OpsService/" + method
				key := uuid + "|server-ops-api|" + route
				if seenTraces[key] {
					continue
				}
				statusCode := int32(200)
				success := true
				if status != "OK" {
					statusCode = 500
					success = false
				}
				rec := &pb.RouteRequestLog{
					TraceId:     uuid,
					Region:      currentRegion,
					ServiceName: "server-ops-api",
					InstanceId:  "server-ops-api-1",
					Route:       route,
					Method:      "POST",
					StatusCode:  statusCode,
					Success:     success,
					RttMs:       float64(durationMs),
					OccurredAt:  timestamppb.New(createdAt),
				}
				if !matchesFilters(rec) {
					continue
				}
				seenTraces[key] = true
				records = append(records, rec)
				if len(records) >= limit*2 {
					break
				}
			}
		}
	}

	sort.Slice(records, func(i, j int) bool {
		return records[i].GetOccurredAt().AsTime().After(records[j].GetOccurredAt().AsTime())
	})

	filtered, err := filterAndSortRouteRequests(records, req.Msg.GetFilter(), req.Msg.GetSort())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	if len(filtered) > limit {
		filtered = filtered[:limit]
	}

	return connect.NewResponse(&pb.ListRouteRequestsResponse{
		Requests: filtered,
	}), nil
}

func (s *OpsService) queryVictoriaLogs(ctx context.Context, region, service, route string, onlyErrors bool, limit int) []*pb.RouteRequestLog {
	vlURL := strings.TrimSpace(os.Getenv("QS_VICTORIA_LOGS_URL"))
	if vlURL == "" {
		return nil
	}

	var parts []string
	parts = append(parts, `_msg:"http_request"`)
	if region != "" && !strings.EqualFold(region, "all") {
		parts = append(parts, fmt.Sprintf(`region:%q`, region))
	}
	if service != "" && !strings.EqualFold(service, "all") {
		parts = append(parts, fmt.Sprintf(`service:%q`, service))
	}
	if route != "" {
		parts = append(parts, fmt.Sprintf(`route:%q`, route))
	}
	if onlyErrors {
		parts = append(parts, `success:false`)
	}

	query := strings.Join(parts, " AND ")
	form := url.Values{}
	form.Set("query", query)
	form.Set("limit", fmt.Sprintf("%d", limit))

	client := &http.Client{Timeout: 800 * time.Millisecond}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(vlURL, "/")+"/select/logsql/query", bytes.NewBufferString(form.Encode()))
	if err != nil {
		return nil
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil
	}

	var results []*pb.RouteRequestLog
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		var row struct {
			Time       string  `json:"_time"`
			Region     string  `json:"region"`
			Service    string  `json:"service"`
			InstanceID string  `json:"instance_id"`
			TraceID    string  `json:"trace_id"`
			Route      string  `json:"route"`
			Method     string  `json:"method"`
			StatusCode int32   `json:"status_code"`
			Success    bool    `json:"success"`
			RttMs      float64 `json:"rtt_ms"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &row); err != nil || row.TraceID == "" {
			continue
		}
		ts := time.Now().UTC()
		if parsed, err := time.Parse(time.RFC3339Nano, row.Time); err == nil {
			ts = parsed
		}
		results = append(results, &pb.RouteRequestLog{
			TraceId:     row.TraceID,
			Region:      row.Region,
			ServiceName: row.Service,
			InstanceId:  row.InstanceID,
			Route:       row.Route,
			Method:      row.Method,
			StatusCode:  row.StatusCode,
			Success:     row.Success,
			RttMs:       row.RttMs,
			OccurredAt:  timestamppb.New(ts),
		})
	}
	return results
}
