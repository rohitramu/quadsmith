package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	bconfig "quadsmith/shared/pkg/config"
	"quadsmith/shared/pkg/static"
)

func TestWebServer_ConfigValidation(t *testing.T) {
	// Empty config path without QS_CONFIG should fail
	t.Setenv("QS_CONFIG", "")
	_, err := bconfig.LoadWebServerConfig("")
	if err == nil {
		t.Fatal("expected error when config path is empty")
	}

	// Valid sandbox config should load
	cfgPath := filepath.Join("..", "..", "frontend", "web", "config", "sandbox.textproto")
	cfg, err := bconfig.LoadWebServerConfig(cfgPath)
	if err != nil {
		t.Fatalf("failed to load web server sandbox config: %v", err)
	}
	if cfg.Server.Port != 3000 {
		t.Errorf("expected port 3000, got %d", cfg.Server.Port)
	}
	if cfg.ApiUrl != "http://localhost:8080" {
		t.Errorf("expected apiUrl http://localhost:8080, got %s", cfg.ApiUrl)
	}
}

func TestWebServer_Endpoints(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "web_server_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	indexHTML := `<!doctype html><html><head><title>Quadsmith</title></head><body><div id="root"></div></body></html>`
	if err := os.WriteFile(filepath.Join(tempDir, "index.html"), []byte(indexHTML), 0644); err != nil {
		t.Fatalf("failed to write index.html: %v", err)
	}

	clientCfg := map[string]string{
		"environment": "sandbox",
		"apiUrl":      "http://localhost:8080",
		"appName":     "Quadsmith (Sandbox)",
	}
	clientCfgBytes, _ := json.Marshal(clientCfg)

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})
	mux.HandleFunc("/api/v1/config", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(clientCfgBytes)
	})
	mux.Handle("/", static.ServeSPAWithConfig(tempDir, nil, string(clientCfgBytes)))

	// Test /healthz
	reqHealth := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	recHealth := httptest.NewRecorder()
	mux.ServeHTTP(recHealth, reqHealth)
	if recHealth.Code != http.StatusOK {
		t.Errorf("expected status 200 for /healthz, got %d", recHealth.Code)
	}
	if recHealth.Body.String() != "ok\n" {
		t.Errorf("expected 'ok\\n', got %q", recHealth.Body.String())
	}

	// Test /api/v1/config
	reqCfg := httptest.NewRequest(http.MethodGet, "/api/v1/config", nil)
	recCfg := httptest.NewRecorder()
	mux.ServeHTTP(recCfg, reqCfg)
	if recCfg.Code != http.StatusOK {
		t.Errorf("expected status 200 for /api/v1/config, got %d", recCfg.Code)
	}
	if !strings.Contains(recCfg.Body.String(), "http://localhost:8080") {
		t.Errorf("expected api_url in response, got %s", recCfg.Body.String())
	}

	// Test SPA injection on /login
	reqSPA := httptest.NewRequest(http.MethodGet, "/login", nil)
	recSPA := httptest.NewRecorder()
	mux.ServeHTTP(recSPA, reqSPA)
	if recSPA.Code != http.StatusOK {
		t.Errorf("expected status 200 for /login, got %d", recSPA.Code)
	}
	body := recSPA.Body.String()
	if !strings.Contains(body, "window.__QUADSMITH_CONFIG__") {
		t.Errorf("expected injected window.__QUADSMITH_CONFIG__ in index.html, got: %s", body)
	}
	if !strings.Contains(body, "http://localhost:8080") {
		t.Errorf("expected injected api url in window.__QUADSMITH_CONFIG__, got: %s", body)
	}
}
