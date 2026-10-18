package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"os"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"quadsmith/shared/pkg/authcontext"
)

const (
	TraceIDHeader          = "X-Trace-Id"
	InternalHopTokenHeader = "X-Internal-Hop-Token"
	DefaultVersion         = "0.1.0"
)

var (
	uuidRegex = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

	histogramBuckets = []float64{1, 2.5, 5, 10, 25, 50, 100, 250, 500, 1000, 2500}
)

type traceIDContextKey struct{}
type loggerContextKey struct{}

// NewTraceID generates a new RFC 9562 time-ordered UUID v7 string.
func NewTraceID() string {
	return authcontext.NewUUIDv7()
}

// IsValidUUID returns true if s is a valid canonical UUID string.
func IsValidUUID(s string) bool {
	return uuidRegex.MatchString(strings.TrimSpace(s))
}

// CurrentRegion returns the active deployment region from QS_REGION,
// defaulting to "us-central1" in staging/prod or "local" in sandbox.
func CurrentRegion() string {
	if r := strings.TrimSpace(os.Getenv("QS_REGION")); r != "" {
		return r
	}
	env := strings.ToLower(strings.TrimSpace(os.Getenv("QS_ENV")))
	if env == "staging" || env == "prod" || env == "production" {
		return "us-central1"
	}
	return "local"
}

// CurrentInstanceID returns the active instance identifier from QS_INSTANCE_ID or HOSTNAME.
func CurrentInstanceID(serviceName string) string {
	if id := strings.TrimSpace(os.Getenv("QS_INSTANCE_ID")); id != "" {
		return id
	}
	if host := strings.TrimSpace(os.Getenv("HOSTNAME")); host != "" {
		return host
	}
	return fmt.Sprintf("%s-%s", serviceName, CurrentRegion())
}

// InternalHopToken returns the shared internal hop authentication secret.
func InternalHopToken() string {
	if tok := strings.TrimSpace(os.Getenv("QS_INTERNAL_HOP_TOKEN")); tok != "" {
		return tok
	}
	if tok := strings.TrimSpace(os.Getenv("QS_JOB_SERVER_AUTH_TOKEN")); tok != "" {
		return tok
	}
	return "qs_robot_sandbox_token_67890"
}

// WithTraceID returns a new context carrying the given traceID.
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDContextKey{}, traceID)
}

// TraceIDFromContext extracts the active trace_id from ctx, or returns empty string if unset.
func TraceIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if id, ok := ctx.Value(traceIDContextKey{}).(string); ok {
		return id
	}
	return ""
}

// WithLogger attaches a request-scoped structured logger to ctx.
func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerContextKey{}, logger)
}

// LoggerFromContext returns the request-scoped slog.Logger from ctx, or slog.Default().
func LoggerFromContext(ctx context.Context) *slog.Logger {
	if ctx != nil {
		if l, ok := ctx.Value(loggerContextKey{}).(*slog.Logger); ok && l != nil {
			return l
		}
	}
	return slog.Default()
}

// TracedTransport is an http.RoundTripper that propagates X-Trace-Id and X-Internal-Hop-Token
// across outbound internal service-to-service requests.
type TracedTransport struct {
	Base             http.RoundTripper
	InternalHopToken string
	BearerToken      string
}

func (t *TracedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.Base
	if base == nil {
		base = http.DefaultTransport
	}
	clone := req.Clone(req.Context())
	traceID := TraceIDFromContext(req.Context())
	if !IsValidUUID(traceID) {
		traceID = NewTraceID()
	}
	clone.Header.Set(TraceIDHeader, traceID)
	if t.InternalHopToken != "" {
		clone.Header.Set(InternalHopTokenHeader, t.InternalHopToken)
	}
	if t.BearerToken != "" && clone.Header.Get("Authorization") == "" {
		clone.Header.Set("Authorization", "Bearer "+t.BearerToken)
	}
	return base.RoundTrip(clone)
}

// NewTracedHTTPClient creates an *http.Client that automatically propagates X-Trace-Id
// and X-Internal-Hop-Token on every outbound request.
func NewTracedHTTPClient(timeout time.Duration, internalHopToken, bearerToken string) *http.Client {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &http.Client{
		Timeout: timeout,
		Transport: &TracedTransport{
			Base:             http.DefaultTransport,
			InternalHopToken: internalHopToken,
			BearerToken:      bearerToken,
		},
	}
}

// HealthCheckResult represents a single dependency check result in /healthz.
type HealthCheckResult struct {
	Name      string  `json:"name"`
	Status    string  `json:"status"`
	LatencyMs float64 `json:"latency_ms"`
	Message   string  `json:"message,omitempty"`
}

// HealthResponse is the JSON payload returned by GET /healthz.
type HealthResponse struct {
	Status        string              `json:"status"`
	Service       string              `json:"service"`
	Region        string              `json:"region"`
	InstanceID    string              `json:"instance_id"`
	Version       string              `json:"version"`
	Environment   string              `json:"environment"`
	UptimeSeconds int64               `json:"uptime_seconds"`
	CheckedAt     string              `json:"checked_at"`
	Checks        []HealthCheckResult `json:"checks"`
}

// RouteMetricSnapshot represents aggregated availability and RTT percentiles for a route.
type RouteMetricSnapshot struct {
	Route               string  `json:"route"`
	Method              string  `json:"method"`
	Region              string  `json:"region"`
	ServiceName         string  `json:"service_name"`
	TotalRequests       int64   `json:"total_requests"`
	SuccessCount        int64   `json:"success_count"`
	ErrorCount          int64   `json:"error_count"`
	AvailabilityPercent float64 `json:"availability_percent"`
	P50RttMs            float64 `json:"p50_rtt_ms"`
	P95RttMs            float64 `json:"p95_rtt_ms"`
	P99RttMs            float64 `json:"p99_rtt_ms"`
}

// RouteRequestEntry represents a single traced request log entry.
type RouteRequestEntry struct {
	TraceID     string    `json:"trace_id"`
	Region      string    `json:"region"`
	ServiceName string    `json:"service_name"`
	InstanceID  string    `json:"instance_id"`
	Route       string    `json:"route"`
	Method      string    `json:"method"`
	StatusCode  int32     `json:"status_code"`
	Success     bool      `json:"success"`
	RttMs       float64   `json:"rtt_ms"`
	OccurredAt  time.Time `json:"occurred_at"`
}

// TraceLogEntry represents a single structured log line emitted during a traced request.
type TraceLogEntry struct {
	TraceID     string            `json:"trace_id"`
	Timestamp   time.Time         `json:"timestamp"`
	Level       string            `json:"level"`
	ServiceName string            `json:"service_name"`
	InstanceID  string            `json:"instance_id"`
	Region      string            `json:"region"`
	Message     string            `json:"message"`
	Attributes  map[string]string `json:"attributes,omitempty"`
}

// MetricsSnapshot is the structured JSON representation returned by GET /metrics?format=json.
type MetricsSnapshot struct {
	ServiceName          string                `json:"service_name"`
	Region               string                `json:"region"`
	InstanceID           string                `json:"instance_id"`
	Version              string                `json:"version"`
	Environment          string                `json:"environment"`
	HealthStatus         int32                 `json:"health_status"`
	UptimeSeconds        int64                 `json:"uptime_seconds"`
	MemoryAllocatedBytes int64                 `json:"memory_allocated_bytes"`
	MemorySysBytes       int64                 `json:"memory_sys_bytes"`
	NumGoroutines        int32                 `json:"num_goroutines"`
	DBAcquiredConns      int32                 `json:"db_acquired_conns"`
	DBIdleConns          int32                 `json:"db_idle_conns"`
	DBMaxConns           int32                 `json:"db_max_conns"`
	SchedulerTasks       int32                 `json:"scheduler_tasks"`
	SchedulerRuns        int64                 `json:"scheduler_runs"`
	SchedulerFailures    int64                 `json:"scheduler_failures"`
	Routes               []RouteMetricSnapshot `json:"routes"`
	RecentRequests       []RouteRequestEntry   `json:"recent_requests"`
	RecentLogs           []TraceLogEntry       `json:"recent_logs,omitempty"`
}

type registeredCheck struct {
	name       string
	isUpstream bool
	fn         func(ctx context.Context) error
}

type routeKey struct {
	route  string
	method string
}

type statusCounterKey struct {
	route      string
	method     string
	statusCode int
	success    bool
}

type routeStats struct {
	total        int64
	successCount int64
	errorCount   int64
	sumMs        float64
	buckets      []int64
	samples      []float64
	sampleIdx    int
}

const (
	maxRouteSamples   = 512
	maxRingEntries    = 500
	maxLogRingEntries = 1000
)

// Registry manages health checks, Prometheus metrics, ingress Trace ID generation,
// and asynchronous VictoriaLogs shipping for a Quadsmith Go binary.
type Registry struct {
	serviceName      string
	environment      string
	region           string
	instanceID       string
	version          string
	startTime        time.Time
	trustInternalHop bool
	internalHopToken string

	checksMu sync.RWMutex
	checks   []registeredCheck

	lastHealthMu       sync.RWMutex
	lastHealthStatus   int32
	lastCheckLatencies map[string]float64

	dbPoolsMu sync.RWMutex
	dbPools   []*pgxpool.Pool

	schedulerTasks    atomic.Int32
	schedulerRuns     atomic.Int64
	schedulerFailures atomic.Int64

	metricsMu      sync.RWMutex
	routes         map[routeKey]*routeStats
	statusCounters map[statusCounterKey]int64
	recentRing     []RouteRequestEntry
	recentLogs     []TraceLogEntry

	logCh           chan []byte
	victoriaLogsURL string
	httpClient      *http.Client
}

// NewRegistry creates a new Telemetry Registry for the specified service.
func NewRegistry(serviceName, environment string, trustInternalHop bool) *Registry {
	if strings.TrimSpace(environment) == "" {
		environment = os.Getenv("QS_ENV")
		if environment == "" {
			environment = "sandbox"
		}
	}
	vlURL := strings.TrimRight(strings.TrimSpace(os.Getenv("QS_VICTORIA_LOGS_URL")), "/")
	r := &Registry{
		serviceName:        serviceName,
		environment:        environment,
		region:             CurrentRegion(),
		instanceID:         CurrentInstanceID(serviceName),
		version:            DefaultVersion,
		startTime:          time.Now(),
		trustInternalHop:   trustInternalHop,
		internalHopToken:   InternalHopToken(),
		lastHealthStatus:   1,
		lastCheckLatencies: make(map[string]float64),
		routes:             make(map[routeKey]*routeStats),
		statusCounters:     make(map[statusCounterKey]int64),
		recentRing:         make([]RouteRequestEntry, 0, maxRingEntries),
		recentLogs:         make([]TraceLogEntry, 0, maxLogRingEntries),
		victoriaLogsURL:    vlURL,
		httpClient:         &http.Client{Timeout: 2 * time.Second},
	}

	if vlURL != "" {
		r.logCh = make(chan []byte, 2048)
		go r.runVictoriaLogsShipper()
	}

	return r
}

// ServiceName returns the service name registered with this Registry.
func (r *Registry) ServiceName() string {
	return r.serviceName
}

// Region returns the deployment region for this Registry.
func (r *Registry) Region() string {
	return r.region
}

// InstanceID returns the instance ID for this Registry.
func (r *Registry) InstanceID() string {
	return r.instanceID
}

// AddCheck registers a local dependency check (e.g. PostgreSQL ping or static file check).
func (r *Registry) AddCheck(name string, fn func(ctx context.Context) error) {
	r.checksMu.Lock()
	defer r.checksMu.Unlock()
	r.checks = append(r.checks, registeredCheck{
		name:       name,
		isUpstream: false,
		fn:         fn,
	})
}

// AddUpstreamCheck registers an upstream HTTP dependency check that is skipped when ?shallow=1 is set.
func (r *Registry) AddUpstreamCheck(name string, fn func(ctx context.Context) error) {
	r.checksMu.Lock()
	defer r.checksMu.Unlock()
	r.checks = append(r.checks, registeredCheck{
		name:       name,
		isUpstream: true,
		fn:         fn,
	})
}

// SetDBPools registers one or more pgxpool.Pool instances for database connection metrics.
func (r *Registry) SetDBPools(pools ...*pgxpool.Pool) {
	r.dbPoolsMu.Lock()
	defer r.dbPoolsMu.Unlock()
	r.dbPools = nil
	for _, p := range pools {
		if p != nil {
			r.dbPools = append(r.dbPools, p)
		}
	}
}

// SetSchedulerRegisteredTasks sets the gauge of registered cron tasks for server-cron.
func (r *Registry) SetSchedulerRegisteredTasks(count int32) {
	r.schedulerTasks.Store(count)
}

// IncSchedulerRun increments the total cron run counter and (if failed) the failure counter.
func (r *Registry) IncSchedulerRun(failed bool) {
	r.schedulerRuns.Add(1)
	if failed {
		r.schedulerFailures.Add(1)
	}
}

// PostgresPingCheck returns a dependency check function that pings pool.
func PostgresPingCheck(pool *pgxpool.Pool) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		if pool == nil {
			return fmt.Errorf("database pool is nil")
		}
		return pool.Ping(ctx)
	}
}

// FileExistsCheck returns a dependency check function that verifies a file exists and is non-empty.
func FileExistsCheck(path string) func(ctx context.Context) error {
	return func(_ context.Context) error {
		info, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("missing file %s: %w", path, err)
		}
		if info.IsDir() {
			return fmt.Errorf("%s is a directory, expected file", path)
		}
		return nil
	}
}

// HTTPHealthCheck returns a dependency check function that probes targetURL with a 2s timeout.
func HTTPHealthCheck(targetURL string) func(ctx context.Context) error {
	client := NewTracedHTTPClient(2*time.Second, InternalHopToken(), "")
	return func(ctx context.Context) error {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
		if err != nil {
			return err
		}
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("upstream %s returned HTTP %d", targetURL, resp.StatusCode)
		}
		return nil
	}
}

// EvaluateHealth runs all registered dependency checks in parallel with a 2s timeout per check.
func (r *Registry) EvaluateHealth(ctx context.Context, shallow bool) HealthResponse {
	r.checksMu.RLock()
	var activeChecks []registeredCheck
	for _, c := range r.checks {
		if shallow && c.isUpstream {
			continue
		}
		activeChecks = append(activeChecks, c)
	}
	r.checksMu.RUnlock()

	results := make([]HealthCheckResult, len(activeChecks))
	var wg sync.WaitGroup
	for i, chk := range activeChecks {
		wg.Add(1)
		go func(idx int, c registeredCheck) {
			defer wg.Done()
			cCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
			defer cancel()
			t0 := time.Now()
			err := c.fn(cCtx)
			latMs := math.Round(float64(time.Since(t0).Microseconds())/10.0) / 100.0
			res := HealthCheckResult{
				Name:      c.name,
				Status:    "HEALTHY",
				LatencyMs: latMs,
			}
			if err != nil {
				res.Status = "UNHEALTHY"
				res.Message = err.Error()
			}
			results[idx] = res
		}(i, chk)
	}
	wg.Wait()

	overallStatus := "HEALTHY"
	healthGauge := int32(1)
	r.lastHealthMu.Lock()
	for _, res := range results {
		r.lastCheckLatencies[res.Name] = res.LatencyMs
		if res.Status != "HEALTHY" {
			overallStatus = "UNHEALTHY"
			healthGauge = 0
		}
	}
	r.lastHealthStatus = healthGauge
	r.lastHealthMu.Unlock()

	return HealthResponse{
		Status:        overallStatus,
		Service:       r.serviceName,
		Region:        r.region,
		InstanceID:    r.instanceID,
		Version:       r.version,
		Environment:   r.environment,
		UptimeSeconds: int64(time.Since(r.startTime).Seconds()),
		CheckedAt:     time.Now().UTC().Format(time.RFC3339),
		Checks:        results,
	}
}

// HealthzHandler returns an http.HandlerFunc serving structured JSON deep health checks on /healthz.
func (r *Registry) HealthzHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		q := req.URL.Query().Get("shallow")
		shallow := q == "1" || strings.EqualFold(q, "true")
		resp := r.EvaluateHealth(req.Context(), shallow)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		if resp.Status == "HEALTHY" {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		_ = json.NewEncoder(w).Encode(resp)
	}
}

type statusCapturingResponseWriter struct {
	http.ResponseWriter
	statusCode  int
	wroteHeader bool
}

func (w *statusCapturingResponseWriter) WriteHeader(code int) {
	if !w.wroteHeader {
		w.statusCode = code
		w.wroteHeader = true
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusCapturingResponseWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.statusCode = http.StatusOK
		w.wroteHeader = true
	}
	return w.ResponseWriter.Write(b)
}

func (w *statusCapturingResponseWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Middleware enforces strict ingress X-Trace-Id generation (ignoring any client-supplied X-Trace-Id
// unless authenticated by X-Internal-Hop-Token on an internal service), sets X-Trace-Id on the response,
// records route metrics, and ships structured JSON logs to VictoriaLogs.
func (r *Registry) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		var traceID string
		hopType := "ingress"
		incomingHopToken := req.Header.Get(InternalHopTokenHeader)
		incomingTraceID := strings.TrimSpace(req.Header.Get(TraceIDHeader))

		if r.trustInternalHop && r.internalHopToken != "" && incomingHopToken == r.internalHopToken && IsValidUUID(incomingTraceID) {
			traceID = incomingTraceID
			hopType = "internal_hop"
		} else {
			// Strictly ignore any client-supplied X-Trace-Id and generate a fresh UUID v7 at ingress.
			traceID = NewTraceID()
		}

		// Ensure req.Header also reflects the authoritative traceID so downstream handlers see it.
		req.Header.Set(TraceIDHeader, traceID)
		w.Header().Set(TraceIDHeader, traceID)

		ctx := WithTraceID(req.Context(), traceID)
		reqLogger := slog.Default().With(
			slog.String("trace_id", traceID),
			slog.String("region", r.region),
			slog.String("service", r.serviceName),
			slog.String("instance", r.instanceID),
		)
		ctx = WithLogger(ctx, reqLogger)
		req = req.WithContext(ctx)

		route := normalizeRoute(req.URL.Path)
		if route != "/metrics" && route != "/healthz" && req.Method != http.MethodOptions {
			r.EmitLog(ctx, "INFO", "request_started", map[string]any{
				"route":    route,
				"method":   req.Method,
				"hop_type": hopType,
			})
		}

		rw := &statusCapturingResponseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		t0 := time.Now()
		next.ServeHTTP(rw, req)
		rttMs := math.Round(float64(time.Since(t0).Microseconds())/10.0) / 100.0

		if route == "/metrics" || req.Method == http.MethodOptions {
			return
		}

		statusCode := rw.statusCode
		if statusCode == 0 {
			statusCode = http.StatusOK
		}
		success := statusCode >= 200 && statusCode < 400
		if strings.HasPrefix(route, "/quadsmith.") && statusCode != http.StatusOK {
			success = false
		}

		r.RecordRequest(traceID, route, req.Method, statusCode, success, rttMs)
	})
}

func normalizeRoute(path string) string {
	clean := strings.TrimSpace(path)
	if clean == "" {
		return "/"
	}
	if strings.HasPrefix(clean, "/quadsmith.") {
		return clean
	}
	if clean == "/healthz" || clean == "/metrics" {
		return clean
	}
	if strings.HasPrefix(clean, "/api/blobs") {
		return "/api/blobs"
	}
	if strings.HasPrefix(clean, "/blobs/") {
		return "/blobs/:id"
	}
	if strings.HasPrefix(clean, "/api/account/export") {
		return "/api/account/export"
	}
	if strings.HasPrefix(clean, "/grafana") {
		return "/grafana/"
	}
	if strings.HasPrefix(clean, "/assets/") {
		return "/assets/*"
	}
	if strings.HasPrefix(clean, "/builds/") {
		return "/builds/:id"
	}
	if strings.HasPrefix(clean, "/users/") {
		return "/users/:id"
	}
	if strings.HasPrefix(clean, "/components/") {
		return "/components/*"
	}
	return clean
}

// RecordRequest records an HTTP/RPC request in the Prometheus metrics registry, in-memory ring buffer,
// and ships a structured JSON access log to VictoriaLogs.
func (r *Registry) RecordRequest(traceID, route, method string, statusCode int, success bool, rttMs float64) {
	if !IsValidUUID(traceID) {
		traceID = NewTraceID()
	}
	if method == "" {
		method = http.MethodPost
	}
	now := time.Now().UTC()

	r.metricsMu.Lock()
	rk := routeKey{route: route, method: method}
	st, exists := r.routes[rk]
	if !exists {
		st = &routeStats{
			buckets: make([]int64, len(histogramBuckets)),
			samples: make([]float64, 0, maxRouteSamples),
		}
		r.routes[rk] = st
	}
	st.total++
	if success {
		st.successCount++
	} else {
		st.errorCount++
	}
	st.sumMs += rttMs
	for i, b := range histogramBuckets {
		if rttMs <= b {
			st.buckets[i]++
		}
	}
	if len(st.samples) < maxRouteSamples {
		st.samples = append(st.samples, rttMs)
	} else {
		st.samples[st.sampleIdx] = rttMs
		st.sampleIdx = (st.sampleIdx + 1) % maxRouteSamples
	}

	sck := statusCounterKey{
		route:      route,
		method:     method,
		statusCode: statusCode,
		success:    success,
	}
	r.statusCounters[sck]++

	entry := RouteRequestEntry{
		TraceID:     traceID,
		Region:      r.region,
		ServiceName: r.serviceName,
		InstanceID:  r.instanceID,
		Route:       route,
		Method:      method,
		StatusCode:  int32(statusCode),
		Success:     success,
		RttMs:       rttMs,
		OccurredAt:  now,
	}
	// Avoid letting frequent /healthz checks flood the ring buffer and evict application requests
	updatedHealthzInPlace := false
	if route == "/healthz" && success {
		for idx := range r.recentRing {
			if r.recentRing[idx].Route == "/healthz" && r.recentRing[idx].Success {
				r.recentRing[idx] = entry
				updatedHealthzInPlace = true
				break
			}
		}
	}
	if !updatedHealthzInPlace {
		if len(r.recentRing) >= maxRingEntries {
			copy(r.recentRing, r.recentRing[1:])
			r.recentRing[len(r.recentRing)-1] = entry
		} else {
			r.recentRing = append(r.recentRing, entry)
		}
	}
	r.metricsMu.Unlock()

	if route == "/healthz" && success && updatedHealthzInPlace {
		return
	}

	level := "INFO"
	if !success {
		level = "ERROR"
	}
	r.EmitLog(WithTraceID(context.Background(), traceID), level, "http_request", map[string]any{
		"route":       route,
		"method":      method,
		"status_code": statusCode,
		"success":     success,
		"rtt_ms":      rttMs,
	})
}

// EmitLog writes a structured JSON log record (carrying trace_id and region) to the in-memory ring buffer
// and queues it for asynchronous batch shipping to VictoriaLogs.
func (r *Registry) EmitLog(ctx context.Context, level, msg string, attrs map[string]any) {
	traceID := TraceIDFromContext(ctx)
	now := time.Now().UTC()
	payload := map[string]any{
		"time":     now.Format(time.RFC3339Nano),
		"level":    level,
		"msg":      msg,
		"region":   r.region,
		"service":  r.serviceName,
		"instance": r.instanceID,
	}
	if traceID != "" {
		payload["trace_id"] = traceID
	}
	strAttrs := make(map[string]string, len(attrs))
	for k, v := range attrs {
		payload[k] = v
		strAttrs[k] = fmt.Sprintf("%v", v)
	}

	if traceID != "" {
		logEntry := TraceLogEntry{
			TraceID:     traceID,
			Timestamp:   now,
			Level:       level,
			ServiceName: r.serviceName,
			InstanceID:  r.instanceID,
			Region:      r.region,
			Message:     msg,
			Attributes:  strAttrs,
		}
		r.metricsMu.Lock()
		if len(r.recentLogs) >= maxLogRingEntries {
			copy(r.recentLogs, r.recentLogs[1:])
			r.recentLogs[len(r.recentLogs)-1] = logEntry
		} else {
			r.recentLogs = append(r.recentLogs, logEntry)
		}
		r.metricsMu.Unlock()
	}

	line, err := json.Marshal(payload)
	if err != nil {
		return
	}
	if r.logCh != nil {
		select {
		case r.logCh <- line:
		default:
			// Drop if channel is full so application goroutines never block
		}
	}
}

func (r *Registry) runVictoriaLogsShipper() {
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	var batch [][]byte
	flush := func() {
		if len(batch) == 0 {
			return
		}
		var buf bytes.Buffer
		for _, line := range batch {
			buf.Write(line)
			buf.WriteByte('\n')
		}
		batch = batch[:0]

		insertURL := r.victoriaLogsURL + "/insert/jsonline?_stream_fields=region,service,instance&_msg_field=msg&_time_field=time"
		req, err := http.NewRequest(http.MethodPost, insertURL, &buf)
		if err != nil {
			return
		}
		req.Header.Set("Content-Type", "application/stream+json")
		resp, err := r.httpClient.Do(req)
		if err == nil {
			_ = resp.Body.Close()
		}
	}

	for {
		select {
		case line, ok := <-r.logCh:
			if !ok {
				flush()
				return
			}
			batch = append(batch, line)
			if len(batch) >= 50 {
				flush()
			}
		case <-ticker.C:
			flush()
		}
	}
}

func computePercentiles(samples []float64) (p50, p95, p99 float64) {
	if len(samples) == 0 {
		return 0, 0, 0
	}
	sorted := make([]float64, len(samples))
	copy(sorted, samples)
	sort.Float64s(sorted)

	pick := func(q float64) float64 {
		idx := int(math.Ceil(q*float64(len(sorted)))) - 1
		if idx < 0 {
			idx = 0
		}
		if idx >= len(sorted) {
			idx = len(sorted) - 1
		}
		return math.Round(sorted[idx]*100) / 100
	}
	return pick(0.50), pick(0.95), pick(0.99)
}

func (r *Registry) dbPoolStats() (acquired, idle, maxConns int32) {
	r.dbPoolsMu.RLock()
	pools := r.dbPools
	r.dbPoolsMu.RUnlock()

	for _, p := range pools {
		if p != nil {
			st := p.Stat()
			acquired += st.AcquiredConns()
			idle += st.IdleConns()
			maxConns += st.MaxConns()
		}
	}
	return acquired, idle, maxConns
}

// Snapshot returns the current MetricsSnapshot for this binary.
func (r *Registry) Snapshot() MetricsSnapshot {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	acquired, idle, maxConns := r.dbPoolStats()

	r.lastHealthMu.RLock()
	healthStatus := r.lastHealthStatus
	r.lastHealthMu.RUnlock()

	r.metricsMu.RLock()
	routes := make([]RouteMetricSnapshot, 0, len(r.routes))
	for rk, st := range r.routes {
		avail := 100.0
		if st.total > 0 {
			avail = math.Round((float64(st.successCount)/float64(st.total))*10000) / 100
		}
		p50, p95, p99 := computePercentiles(st.samples)
		routes = append(routes, RouteMetricSnapshot{
			Route:               rk.route,
			Method:              rk.method,
			Region:              r.region,
			ServiceName:         r.serviceName,
			TotalRequests:       st.total,
			SuccessCount:        st.successCount,
			ErrorCount:          st.errorCount,
			AvailabilityPercent: avail,
			P50RttMs:            p50,
			P95RttMs:            p95,
			P99RttMs:            p99,
		})
	}
	sort.Slice(routes, func(i, j int) bool {
		return routes[i].Route < routes[j].Route
	})

	recent := make([]RouteRequestEntry, len(r.recentRing))
	for i := range r.recentRing {
		// Reverse order (newest first)
		recent[i] = r.recentRing[len(r.recentRing)-1-i]
	}

	recentLogs := make([]TraceLogEntry, len(r.recentLogs))
	copy(recentLogs, r.recentLogs)
	r.metricsMu.RUnlock()

	return MetricsSnapshot{
		ServiceName:          r.serviceName,
		Region:               r.region,
		InstanceID:           r.instanceID,
		Version:              r.version,
		Environment:          r.environment,
		HealthStatus:         healthStatus,
		UptimeSeconds:        int64(time.Since(r.startTime).Seconds()),
		MemoryAllocatedBytes: int64(mem.Alloc),
		MemorySysBytes:       int64(mem.Sys),
		NumGoroutines:        int32(runtime.NumGoroutine()),
		DBAcquiredConns:      acquired,
		DBIdleConns:          idle,
		DBMaxConns:           maxConns,
		SchedulerTasks:       r.schedulerTasks.Load(),
		SchedulerRuns:        r.schedulerRuns.Load(),
		SchedulerFailures:    r.schedulerFailures.Load(),
		Routes:               routes,
		RecentRequests:       recent,
		RecentLogs:           recentLogs,
	}
}

// MetricsHandler serves Prometheus exposition format (0.0.4) at GET /metrics
// (and JSON snapshot when ?format=json is requested internally).
func (r *Registry) MetricsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Query().Get("format") == "json" {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(r.Snapshot())
			return
		}

		r.lastHealthMu.RLock()
		hasCheckLatencies := len(r.lastCheckLatencies) > 0
		r.lastHealthMu.RUnlock()
		if !hasCheckLatencies {
			_ = r.EvaluateHealth(req.Context(), true)
		}

		var mem runtime.MemStats
		runtime.ReadMemStats(&mem)
		uptime := int64(time.Since(r.startTime).Seconds())
		goroutines := runtime.NumGoroutine()
		acquired, idle, maxConns := r.dbPoolStats()

		r.lastHealthMu.RLock()
		healthStatus := r.lastHealthStatus
		checkLatencies := make(map[string]float64, len(r.lastCheckLatencies))
		for k, v := range r.lastCheckLatencies {
			checkLatencies[k] = v
		}
		r.lastHealthMu.RUnlock()

		baseLabels := fmt.Sprintf(`region=%q,service=%q,instance=%q`, r.region, r.serviceName, r.instanceID)

		var b strings.Builder
		b.WriteString("# HELP quadsmith_health_status Current deep health check status (1 = healthy, 0 = unhealthy).\n")
		b.WriteString("# TYPE quadsmith_health_status gauge\n")
		fmt.Fprintf(&b, "quadsmith_health_status{%s} %d\n", baseLabels, healthStatus)

		b.WriteString("# HELP quadsmith_dependency_check_latency_ms Latency of dependency checks in milliseconds.\n")
		b.WriteString("# TYPE quadsmith_dependency_check_latency_ms gauge\n")
		for chkName, lat := range checkLatencies {
			fmt.Fprintf(&b, "quadsmith_dependency_check_latency_ms{%s,check=%q} %.2f\n", baseLabels, chkName, lat)
		}

		b.WriteString("# HELP quadsmith_uptime_seconds Process uptime in seconds.\n")
		b.WriteString("# TYPE quadsmith_uptime_seconds gauge\n")
		fmt.Fprintf(&b, "quadsmith_uptime_seconds{%s} %d\n", baseLabels, uptime)

		b.WriteString("# HELP quadsmith_memory_allocated_bytes Bytes of allocated heap objects.\n")
		b.WriteString("# TYPE quadsmith_memory_allocated_bytes gauge\n")
		fmt.Fprintf(&b, "quadsmith_memory_allocated_bytes{%s} %d\n", baseLabels, mem.Alloc)

		b.WriteString("# HELP quadsmith_memory_sys_bytes Total bytes of memory obtained from the OS.\n")
		b.WriteString("# TYPE quadsmith_memory_sys_bytes gauge\n")
		fmt.Fprintf(&b, "quadsmith_memory_sys_bytes{%s} %d\n", baseLabels, mem.Sys)

		b.WriteString("# HELP quadsmith_goroutines Number of active Go routines.\n")
		b.WriteString("# TYPE quadsmith_goroutines gauge\n")
		fmt.Fprintf(&b, "quadsmith_goroutines{%s} %d\n", baseLabels, goroutines)

		b.WriteString("# HELP quadsmith_gc_cpu_fraction Fraction of CPU time used by the GC.\n")
		b.WriteString("# TYPE quadsmith_gc_cpu_fraction gauge\n")
		fmt.Fprintf(&b, "quadsmith_gc_cpu_fraction{%s} %.6f\n", baseLabels, mem.GCCPUFraction)

		b.WriteString("# HELP quadsmith_db_acquired_conns Number of acquired PostgreSQL connections.\n")
		b.WriteString("# TYPE quadsmith_db_acquired_conns gauge\n")
		fmt.Fprintf(&b, "quadsmith_db_acquired_conns{%s} %d\n", baseLabels, acquired)

		b.WriteString("# HELP quadsmith_db_idle_conns Number of idle PostgreSQL connections.\n")
		b.WriteString("# TYPE quadsmith_db_idle_conns gauge\n")
		fmt.Fprintf(&b, "quadsmith_db_idle_conns{%s} %d\n", baseLabels, idle)

		b.WriteString("# HELP quadsmith_db_max_conns Maximum configured PostgreSQL connections.\n")
		b.WriteString("# TYPE quadsmith_db_max_conns gauge\n")
		fmt.Fprintf(&b, "quadsmith_db_max_conns{%s} %d\n", baseLabels, maxConns)

		b.WriteString("# HELP quadsmith_scheduler_registered_tasks Number of registered cron tasks.\n")
		b.WriteString("# TYPE quadsmith_scheduler_registered_tasks gauge\n")
		fmt.Fprintf(&b, "quadsmith_scheduler_registered_tasks{%s} %d\n", baseLabels, r.schedulerTasks.Load())

		b.WriteString("# HELP quadsmith_scheduler_runs_total Total number of scheduled cron executions.\n")
		b.WriteString("# TYPE quadsmith_scheduler_runs_total counter\n")
		fmt.Fprintf(&b, "quadsmith_scheduler_runs_total{%s} %d\n", baseLabels, r.schedulerRuns.Load())

		b.WriteString("# HELP quadsmith_scheduler_failures_total Total number of failed scheduled cron executions.\n")
		b.WriteString("# TYPE quadsmith_scheduler_failures_total counter\n")
		fmt.Fprintf(&b, "quadsmith_scheduler_failures_total{%s} %d\n", baseLabels, r.schedulerFailures.Load())

		r.metricsMu.RLock()
		b.WriteString("# HELP quadsmith_http_requests_total Total number of HTTP/RPC requests by route and status.\n")
		b.WriteString("# TYPE quadsmith_http_requests_total counter\n")
		for sck, count := range r.statusCounters {
			fmt.Fprintf(&b, "quadsmith_http_requests_total{%s,route=%q,method=%q,status_code=\"%d\",success=\"%t\"} %d\n",
				baseLabels, sck.route, sck.method, sck.statusCode, sck.success, count)
		}

		b.WriteString("# HELP quadsmith_http_request_duration_ms HTTP/RPC request duration histogram in milliseconds.\n")
		b.WriteString("# TYPE quadsmith_http_request_duration_ms histogram\n")
		for rk, st := range r.routes {
			for i, le := range histogramBuckets {
				fmt.Fprintf(&b, "quadsmith_http_request_duration_ms_bucket{%s,route=%q,method=%q,le=\"%g\"} %d\n",
					baseLabels, rk.route, rk.method, le, st.buckets[i])
			}
			fmt.Fprintf(&b, "quadsmith_http_request_duration_ms_bucket{%s,route=%q,method=%q,le=\"+Inf\"} %d\n",
				baseLabels, rk.route, rk.method, st.total)
			fmt.Fprintf(&b, "quadsmith_http_request_duration_ms_sum{%s,route=%q,method=%q} %.2f\n",
				baseLabels, rk.route, rk.method, st.sumMs)
			fmt.Fprintf(&b, "quadsmith_http_request_duration_ms_count{%s,route=%q,method=%q} %d\n",
				baseLabels, rk.route, rk.method, st.total)
		}
		r.metricsMu.RUnlock()

		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(b.String()))
	}
}
