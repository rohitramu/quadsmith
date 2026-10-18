package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"quadsmith/shared/pkg/config"
	"quadsmith/shared/pkg/telemetry"
)

func TestCronServer_HealthzAndMetrics(t *testing.T) {
	cfg := config.NewTestCronServerConfig(t)
	reg := telemetry.NewRegistry("server-cron", "sandbox", false)
	reg.AddCheck("scheduler_loop", func(_ context.Context) error {
		return nil
	})
	reg.AddUpstreamCheck("upstream:server-ops-api", func(_ context.Context) error {
		return nil
	})

	handler := newCronHTTPHandler(cfg, reg)

	// 1. /healthz
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from /healthz, got %d", rec.Code)
	}
	var report telemetry.HealthResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &report); err != nil {
		t.Fatalf("failed to decode /healthz JSON: %v", err)
	}
	if report.Service != "server-cron" || report.Status != "HEALTHY" {
		t.Errorf("unexpected health report: %+v", report)
	}

	// 2. /metrics
	reqMetrics := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	recMetrics := httptest.NewRecorder()
	handler.ServeHTTP(recMetrics, reqMetrics)

	if recMetrics.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from /metrics, got %d", recMetrics.Code)
	}
	if !strings.Contains(recMetrics.Body.String(), `service="server-cron"`) {
		t.Errorf("expected /metrics to contain service=\"server-cron\", got:\n%s", recMetrics.Body.String())
	}
}
