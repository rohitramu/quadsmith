package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/proto"

	"quadsmith/shared/pkg/config"
	"quadsmith/shared/pkg/telemetry"

	"google.golang.org/protobuf/types/known/timestamppb"
	opspb "quadsmith/shared/gen/quadsmith/ops"
	"quadsmith/shared/gen/quadsmith/ops/opsconnect"
)

type mockOpsAPIServer struct {
	opsconnect.UnimplementedOpsServiceHandler
}

func (m *mockOpsAPIServer) GetCurrentOperator(ctx context.Context, req *connect.Request[opspb.GetCurrentOperatorRequest]) (*connect.Response[opspb.GetCurrentOperatorResponse], error) {
	auth := req.Header().Get("Authorization")
	switch auth {
	case "Bearer tok-with-jit":
		return connect.NewResponse(&opspb.GetCurrentOperatorResponse{
			Operator:     "rohitramu",
			OperatorType: "HUMAN",
			GroupName:    "operators",
			ActiveJitGrant: &opspb.JitGrantRecord{
				Id:        "jit_active_1",
				Operator:  "rohitramu",
				Status:    "ACTIVE",
				Operation: "*",
				ExpiresAt: timestamppb.New(time.Now().Add(2 * time.Hour)),
			},
		}), nil
	case "Bearer tok-no-jit":
		return connect.NewResponse(&opspb.GetCurrentOperatorResponse{
			Operator:     "rohitramu",
			OperatorType: "HUMAN",
			GroupName:    "operators",
		}), nil
	default:
		return nil, connect.NewError(connect.CodeUnauthenticated, nil)
	}
}

func TestOpsWebServer_JitGatedGrafanaAndHealthz(t *testing.T) {
	tempDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tempDir, "index.html"), []byte("<html><head><title>Quadsmith Ops Console</title></head><body>ops</body></html>"), 0644); err != nil {
		t.Fatalf("failed to write index.html: %v", err)
	}

	// Mock server-ops-api
	opsMux := http.NewServeMux()
	opsPath, opsHandler := opsconnect.NewOpsServiceHandler(&mockOpsAPIServer{})
	opsMux.Handle(opsPath, opsHandler)
	opsMux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"HEALTHY","service":"server-ops-api"}`))
	})
	opsMux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("quadsmith_health_status{service=\"server-ops-api\"} 1\n"))
	})
	opsAPISrv := httptest.NewServer(opsMux)
	defer opsAPISrv.Close()

	// Mock Grafana container
	var receivedWebAuthUser string
	var receivedTraceID string
	grafanaSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/health" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"database":"ok"}`))
			return
		}
		receivedWebAuthUser = r.Header.Get("X-WEBAUTH-USER")
		receivedTraceID = r.Header.Get(telemetry.TraceIDHeader)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Grafana Dashboard Loaded"))
	}))
	defer grafanaSrv.Close()

	cfg := config.NewTestOpsWebServerConfig(t)
	cfg.StaticDir = proto.String(tempDir)
	cfg.OpsApiUrl = proto.String(opsAPISrv.URL)
	cfg.GrafanaUrl = proto.String(grafanaSrv.URL)

	reg := telemetry.NewRegistry("server-ops-web", "sandbox", false)
	handler := newOpsWebHandler(cfg, reg)

	// 1. /healthz returns 200 HEALTHY
	reqHealth := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	recHealth := httptest.NewRecorder()
	handler.ServeHTTP(recHealth, reqHealth)
	if recHealth.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from /healthz, got %d: %s", recHealth.Code, recHealth.Body.String())
	}
	var hr telemetry.HealthResponse
	if err := json.Unmarshal(recHealth.Body.Bytes(), &hr); err != nil {
		t.Fatalf("failed to decode /healthz JSON: %v", err)
	}
	if hr.Status != "HEALTHY" || hr.Service != "server-ops-web" {
		t.Errorf("unexpected health report: %+v", hr)
	}

	// 2. Unauthenticated request to /grafana/ -> 403 Forbidden
	reqUnauth := httptest.NewRequest(http.MethodGet, "/grafana/d/quadsmith-health", nil)
	recUnauth := httptest.NewRecorder()
	handler.ServeHTTP(recUnauth, reqUnauth)
	if recUnauth.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden without token on /grafana/*, got %d", recUnauth.Code)
	}

	// 3. Authenticated operator WITHOUT active JIT lease -> 403 Forbidden
	reqNoJit := httptest.NewRequest(http.MethodGet, "/grafana/d/quadsmith-health", nil)
	reqNoJit.AddCookie(&http.Cookie{Name: "qs_ops_token", Value: "tok-no-jit"})
	recNoJit := httptest.NewRecorder()
	handler.ServeHTTP(recNoJit, reqNoJit)
	if recNoJit.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden without active JIT grant on /grafana/*, got %d", recNoJit.Code)
	}

	// 4. Authenticated operator WITH active JIT lease -> 200 OK proxied to Grafana with X-WEBAUTH-USER
	reqWithJit := httptest.NewRequest(http.MethodGet, "/grafana/d/quadsmith-health", nil)
	reqWithJit.AddCookie(&http.Cookie{Name: "qs_ops_token", Value: "tok-with-jit"})
	reqWithJit.Header.Set("X-WEBAUTH-USER", "spoofed-admin")
	recWithJit := httptest.NewRecorder()
	handler.ServeHTTP(recWithJit, reqWithJit)
	if recWithJit.Code != http.StatusOK {
		t.Fatalf("expected 200 OK with active JIT grant on /grafana/*, got %d: %s", recWithJit.Code, recWithJit.Body.String())
	}
	if !strings.Contains(recWithJit.Body.String(), "Grafana Dashboard Loaded") {
		t.Errorf("expected proxied Grafana body, got %q", recWithJit.Body.String())
	}
	if receivedWebAuthUser != "rohitramu" {
		t.Errorf("expected X-WEBAUTH-USER 'rohitramu' (spoofed header stripped), got %q", receivedWebAuthUser)
	}
	if !telemetry.IsValidUUID(receivedTraceID) {
		t.Errorf("expected valid UUID trace ID forwarded to Grafana, got %q", receivedTraceID)
	}

	// 5. Unauthenticated request to /services/server-ops-api/healthz -> 403 Forbidden
	reqSvcUnauth := httptest.NewRequest(http.MethodGet, "/services/server-ops-api/healthz", nil)
	recSvcUnauth := httptest.NewRecorder()
	handler.ServeHTTP(recSvcUnauth, reqSvcUnauth)
	if recSvcUnauth.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden without token on /services/*, got %d", recSvcUnauth.Code)
	}

	// 6. Authenticated operator (no JIT required) can access /services/<service>/healthz and /metrics
	reqSvcHealth := httptest.NewRequest(http.MethodGet, "/services/server-ops-api/healthz", nil)
	reqSvcHealth.AddCookie(&http.Cookie{Name: "qs_ops_token", Value: "tok-no-jit"})
	recSvcHealth := httptest.NewRecorder()
	handler.ServeHTTP(recSvcHealth, reqSvcHealth)
	if recSvcHealth.Code != http.StatusOK || !strings.Contains(recSvcHealth.Body.String(), `"service":"server-ops-api"`) {
		t.Fatalf("expected 200 OK from /services/server-ops-api/healthz, got %d: %s", recSvcHealth.Code, recSvcHealth.Body.String())
	}

	reqSvcMetrics := httptest.NewRequest(http.MethodGet, "/services/server-ops-api/metrics", nil)
	reqSvcMetrics.AddCookie(&http.Cookie{Name: "qs_ops_token", Value: "tok-no-jit"})
	recSvcMetrics := httptest.NewRecorder()
	handler.ServeHTTP(recSvcMetrics, reqSvcMetrics)
	if recSvcMetrics.Code != http.StatusOK || !strings.Contains(recSvcMetrics.Body.String(), "quadsmith_health_status") {
		t.Fatalf("expected 200 OK from /services/server-ops-api/metrics, got %d: %s", recSvcMetrics.Code, recSvcMetrics.Body.String())
	}

	reqSelfMetrics := httptest.NewRequest(http.MethodGet, "/services/server-ops-web/metrics", nil)
	reqSelfMetrics.AddCookie(&http.Cookie{Name: "qs_ops_token", Value: "tok-no-jit"})
	recSelfMetrics := httptest.NewRecorder()
	handler.ServeHTTP(recSelfMetrics, reqSelfMetrics)
	if recSelfMetrics.Code != http.StatusOK || !strings.Contains(recSelfMetrics.Body.String(), "quadsmith_health_status") {
		t.Fatalf("expected 200 OK from /services/server-ops-web/metrics, got %d: %s", recSelfMetrics.Code, recSelfMetrics.Body.String())
	}
}
