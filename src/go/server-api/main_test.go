package main

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"quadsmith/shared/pkg/config"
)

func TestAPIServer_ConfigValidation(t *testing.T) {
	t.Setenv("QS_CONFIG", "")
	_, err := config.LoadAPIServerConfig("")
	if err == nil {
		t.Fatal("expected error when config path is empty")
	}

	cfgPath := filepath.Join("config", "sandbox.textproto")
	cfg, err := config.LoadAPIServerConfig(cfgPath)
	if err != nil {
		t.Fatalf("failed to load API server sandbox config: %v", err)
	}
	if cfg.Server.GetPort() != 8082 {
		t.Errorf("expected port 8082, got %d", cfg.Server.GetPort())
	}
}

func TestAPIServer_CORS(t *testing.T) {
	corsMiddleware := newCORSMiddleware()

	baseHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	handler := corsMiddleware.Handler(baseHandler)

	tests := []struct {
		name          string
		origin        string
		method        string
		expectedAllow bool
		isPreflight   bool
	}{
		{
			name:          "Localhost web port 8080 preflight",
			origin:        "http://localhost:8080",
			method:        http.MethodOptions,
			expectedAllow: true,
			isPreflight:   true,
		},
		{
			name:          "127.0.0.1 web port 8080 preflight",
			origin:        "http://127.0.0.1:8080",
			method:        http.MethodOptions,
			expectedAllow: true,
			isPreflight:   true,
		},
		{
			name:          "Production root domain preflight",
			origin:        "https://quadsmith.com",
			method:        http.MethodOptions,
			expectedAllow: true,
			isPreflight:   true,
		},
		{
			name:          "Staging sub-domain preflight",
			origin:        "https://staging.quadsmith.com",
			method:        http.MethodOptions,
			expectedAllow: true,
			isPreflight:   true,
		},
		{
			name:          "Disallowed external domain preflight",
			origin:        "https://evil-attacker.com",
			method:        http.MethodOptions,
			expectedAllow: false,
			isPreflight:   true,
		},
		{
			name:          "Localhost actual request with headers",
			origin:        "http://localhost:8080",
			method:        http.MethodPost,
			expectedAllow: true,
			isPreflight:   false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, "/quadsmith.api.AuthService/Login", nil)
			req.Header.Set("Origin", tc.origin)
			if tc.isPreflight {
				req.Header.Set("Access-Control-Request-Method", "POST")
				req.Header.Set("Access-Control-Request-Headers", "Content-Type, Connect-Protocol-Version")
			}

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			allowOrigin := rec.Header().Get("Access-Control-Allow-Origin")
			if tc.expectedAllow {
				if allowOrigin != tc.origin {
					t.Errorf("expected Access-Control-Allow-Origin: %q, got %q", tc.origin, allowOrigin)
				}
				if tc.isPreflight && rec.Code != http.StatusNoContent && rec.Code != http.StatusOK {
					t.Errorf("expected 200/204 status for preflight, got %d", rec.Code)
				}
			} else {
				if allowOrigin != "" {
					t.Errorf("expected empty Access-Control-Allow-Origin for disallowed origin, got %q", allowOrigin)
				}
			}
		})
	}
}
