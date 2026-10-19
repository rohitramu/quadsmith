package telemetry

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestIngressTraceID_IgnoresClientHeaderAndTrustsInternalHop(t *testing.T) {
	t.Setenv("QS_REGION", "us-central1")
	t.Setenv("QS_INTERNAL_HOP_TOKEN", "secret-hop-token-123")

	// 1. Pure ingress service (trustInternalHop = false): ALWAYS ignores X-Trace-Id even if hop token is sent.
	ingressReg := NewRegistry("server-web", "prod", false)
	var capturedIngressTraceID string
	ingressHandler := ingressReg.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedIngressTraceID = TraceIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	spoofedID := "018f0000-0000-7000-8000-000000000999"
	req1 := httptest.NewRequest(http.MethodGet, "/builds/test", nil)
	req1.Header.Set(TraceIDHeader, spoofedID)
	req1.Header.Set(InternalHopTokenHeader, "secret-hop-token-123")
	rec1 := httptest.NewRecorder()
	ingressHandler.ServeHTTP(rec1, req1)

	respTrace1 := rec1.Header().Get(TraceIDHeader)
	if !IsValidUUID(respTrace1) {
		t.Fatalf("expected valid UUID v7 in X-Trace-Id response header, got %q", respTrace1)
	}
	if respTrace1 == spoofedID || capturedIngressTraceID == spoofedID {
		t.Fatalf("expected ingress service to ignore client-supplied X-Trace-Id %q, but it reused it", spoofedID)
	}

	// 2. Downstream service (trustInternalHop = true): ignores client X-Trace-Id when X-Internal-Hop-Token is missing or wrong.
	apiReg := NewRegistry("server-api", "prod", true)
	var capturedAPITraceID string
	apiHandler := apiReg.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAPITraceID = TraceIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req2 := httptest.NewRequest(http.MethodPost, "/quadsmith.api.BuildService/GetBuild", nil)
	req2.Header.Set(TraceIDHeader, spoofedID)
	rec2 := httptest.NewRecorder()
	apiHandler.ServeHTTP(rec2, req2)

	if rec2.Header().Get(TraceIDHeader) == spoofedID || capturedAPITraceID == spoofedID {
		t.Fatalf("expected server-api to ignore untrusted client X-Trace-Id %q when hop token is absent", spoofedID)
	}

	// 3. Downstream service (trustInternalHop = true): preserves X-Trace-Id when valid X-Internal-Hop-Token is present.
	trustedTraceID := NewTraceID()
	req3 := httptest.NewRequest(http.MethodPost, "/quadsmith.api.BuildService/GetBuild", nil)
	req3.Header.Set(TraceIDHeader, trustedTraceID)
	req3.Header.Set(InternalHopTokenHeader, "secret-hop-token-123")
	rec3 := httptest.NewRecorder()
	apiHandler.ServeHTTP(rec3, req3)

	if rec3.Header().Get(TraceIDHeader) != trustedTraceID || capturedAPITraceID != trustedTraceID {
		t.Fatalf("expected server-api to preserve trusted internal hop X-Trace-Id %q, got %q", trustedTraceID, capturedAPITraceID)
	}
}

func TestHealthzAndMetricsEndpoints(t *testing.T) {
	t.Setenv("QS_REGION", "europe-west1")
	reg := NewRegistry("server-ops-api", "staging", true)

	healthy := true
	reg.AddCheck("postgres:quadsmith_ops", func(ctx context.Context) error {
		if !healthy {
			return errors.New("connection refused")
		}
		return nil
	})

	// 1. Healthy check returns 200 OK with region and checks
	reqH := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	recH := httptest.NewRecorder()
	reg.HealthzHandler().ServeHTTP(recH, reqH)
	if recH.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from /healthz, got %d", recH.Code)
	}
	var hResp HealthResponse
	if err := json.Unmarshal(recH.Body.Bytes(), &hResp); err != nil {
		t.Fatalf("failed to decode /healthz JSON: %v", err)
	}
	if hResp.Status != "HEALTHY" || hResp.Region != "europe-west1" || hResp.Service != "server-ops-api" {
		t.Errorf("unexpected healthz response: %+v", hResp)
	}

	// 2. Unhealthy check returns 503 Service Unavailable
	healthy = false
	recU := httptest.NewRecorder()
	reg.HealthzHandler().ServeHTTP(recU, reqH)
	if recU.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 Service Unavailable when check fails, got %d", recU.Code)
	}

	// 3. Record requests and verify Prometheus /metrics output includes region label
	reg.RecordRequest(NewTraceID(), "/quadsmith.ops.OpsService/ListJobs", "POST", 200, true, 4.2)
	reg.RecordRequest(NewTraceID(), "/quadsmith.ops.OpsService/ListJobs", "POST", 500, false, 18.5)

	reqM := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	recM := httptest.NewRecorder()
	reg.MetricsHandler().ServeHTTP(recM, reqM)
	if recM.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from /metrics, got %d", recM.Code)
	}
	body := recM.Body.String()
	for _, want := range []string{
		`quadsmith_health_status{region="europe-west1",service="server-ops-api"`,
		`quadsmith_http_requests_total{region="europe-west1",service="server-ops-api"`,
		`route="/quadsmith.ops.OpsService/ListJobs"`,
		`quadsmith_http_request_duration_ms_bucket{region="europe-west1",service="server-ops-api"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("expected /metrics output to contain %q, got:\n%s", want, body)
		}
	}
}

func TestCompactSQLString_NeverEllipsizes(t *testing.T) {
	longSQL := `
		SELECT
			uuid, id, brand, model, stator_diameter_mm, stator_height_mm, kv,
			weight_g, max_power_w, max_current_a, min_cells_s, max_cells_s,
			shaft_diameter_mm, mounting_pattern_mm, retail_price_usd, product_url
		FROM motors
		WHERE stator_diameter_mm >= 22 AND kv BETWEEN 1700 AND 2500
		ORDER BY brand ASC, model ASC, kv DESC
	`
	compacted := compactSQLString(longSQL)
	if strings.Contains(compacted, "...") {
		t.Fatalf("expected compactSQLString not to ellipsize long SQL query, got %q", compacted)
	}
	if !strings.HasSuffix(compacted, "ORDER BY brand ASC, model ASC, kv DESC") {
		t.Fatalf("expected full SQL suffix to be preserved, got %q", compacted)
	}
}
