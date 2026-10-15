package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"google.golang.org/protobuf/proto"

	"quadsmith/shared/pkg/config"

	configpb "quadsmith/shared/gen/quadsmith/config"
)

func TestWebServer_ConfigValidation(t *testing.T) {
	// Empty config path without QS_CONFIG should fail
	t.Setenv("QS_CONFIG", "")
	_, err := config.LoadWebServerConfig("")
	if err == nil {
		t.Fatal("expected error when config path is empty")
	}

	// Valid sandbox config should load
	cfgPath := filepath.Join("config", "sandbox.textproto")
	cfg, err := config.LoadWebServerConfig(cfgPath)
	if err != nil {
		t.Fatalf("failed to load web server sandbox config: %v", err)
	}
	if cfg.Server.GetPort() != 8080 {
		t.Errorf("expected port 8080, got %d", cfg.Server.GetPort())
	}
	if cfg.GetApiUrl() != "http://127.0.0.1:8082" {
		t.Errorf("expected apiUrl http://127.0.0.1:8082, got %s", cfg.GetApiUrl())
	}
	if cfg.GetStaticDir() != "src/react/app-web/dist" {
		t.Errorf("expected staticDir src/react/app-web/dist, got %s", cfg.GetStaticDir())
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

	cfg := &configpb.WebServerConfig{
		ApiUrl:    proto.String("http://127.0.0.1:8082"),
		StaticDir: proto.String(tempDir),
		Server: &configpb.ServerConfig{
			Port:            proto.Int32(8080),
			MaxRequestBytes: proto.Int64(10485760),
		},
	}
	handler := buildHandler(cfg)

	// Test /healthz
	reqHealth := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	recHealth := httptest.NewRecorder()
	handler.ServeHTTP(recHealth, reqHealth)
	if recHealth.Code != http.StatusOK {
		t.Errorf("expected status 200 for /healthz, got %d", recHealth.Code)
	}
	if recHealth.Body.String() != "ok\n" {
		t.Errorf("expected 'ok\\n', got %q", recHealth.Body.String())
	}

	// Test SPA fallback on /login
	reqSPA := httptest.NewRequest(http.MethodGet, "/login", nil)
	recSPA := httptest.NewRecorder()
	handler.ServeHTTP(recSPA, reqSPA)
	if recSPA.Code != http.StatusOK {
		t.Errorf("expected status 200 for /login, got %d", recSPA.Code)
	}
	body := recSPA.Body.String()
	if !strings.Contains(body, `<div id="root"></div>`) {
		t.Errorf("expected SPA index.html in response, got: %s", body)
	}
}
