package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"

	"quadsmith/shared/pkg/config"
	"quadsmith/shared/pkg/telemetry"

	configpb "quadsmith/shared/gen/quadsmith/config"
	opspb "quadsmith/shared/gen/quadsmith/ops"
	"quadsmith/shared/gen/quadsmith/ops/opsconnect"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "", "Path to ops web server configuration textproto file (required)")
	flag.Parse()

	if configPath == "" {
		if envFile := os.Getenv("QS_CONFIG"); envFile != "" {
			configPath = envFile
		} else {
			log.Fatal("server-ops-web: --config flag is required; no auto-resolution allowed")
		}
	}

	cfg, err := config.LoadOpsWebServerConfig(configPath)
	if err != nil {
		log.Fatalf("server-ops-web: configuration validation failed: %v", err)
	}

	port := strconv.Itoa(int(cfg.Server.GetPort()))
	fmt.Printf("Starting Quadsmith Ops Web Server on port %s serving %s...\n", port, cfg.GetStaticDir())

	reg := telemetry.NewRegistry("server-ops-web", cfg.GetEnvironment(), false)
	handler := newOpsWebHandler(cfg, reg)

	readTimeout := 15 * time.Second
	if cfg.Server.ReadTimeout != nil && cfg.Server.ReadTimeout.AsDuration() > 0 {
		readTimeout = cfg.Server.ReadTimeout.AsDuration()
	}
	writeTimeout := 15 * time.Second
	if cfg.Server.WriteTimeout != nil && cfg.Server.WriteTimeout.AsDuration() > 0 {
		writeTimeout = cfg.Server.WriteTimeout.AsDuration()
	}
	idleTimeout := 120 * time.Second
	if cfg.Server.IdleTimeout != nil && cfg.Server.IdleTimeout.AsDuration() > 0 {
		idleTimeout = cfg.Server.IdleTimeout.AsDuration()
	}

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Ops Web server failed: %v", err)
	}
}

func newOpsWebHandler(cfg *configpb.OpsWebServerConfig, reg *telemetry.Registry) http.Handler {
	staticDir := cfg.GetStaticDir()

	reg.AddCheck("static_assets", telemetry.FileExistsCheck(filepath.Join(staticDir, "index.html")))
	reg.AddUpstreamCheck("upstream:server-ops-api", telemetry.HTTPHealthCheck(strings.TrimRight(cfg.GetOpsApiUrl(), "/")+"/healthz"))
	reg.AddUpstreamCheck("upstream:grafana", telemetry.HTTPHealthCheck(strings.TrimRight(cfg.GetGrafanaUrl(), "/")+"/api/health"))

	opsHTTPClient := telemetry.NewTracedHTTPClient(5*time.Second, telemetry.InternalHopToken(), "")
	opsClient := opsconnect.NewOpsServiceClient(opsHTTPClient, cfg.GetOpsApiUrl())

	grafanaURL, _ := url.Parse(cfg.GetGrafanaUrl())
	var grafanaProxy *httputil.ReverseProxy
	if grafanaURL != nil {
		grafanaProxy = httputil.NewSingleHostReverseProxy(grafanaURL)
	}

	fileServer := http.FileServer(http.Dir(staticDir))

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", reg.HealthzHandler())
	mux.HandleFunc("/metrics", reg.MetricsHandler())

	grafanaHandler := func(w http.ResponseWriter, r *http.Request) {
		token := extractOpsToken(r)
		if token == "" {
			writeGrafanaForbidden(w, "Authentication and active JIT elevation lease required to access /grafana/*")
			return
		}

		req := connect.NewRequest(&opspb.GetCurrentOperatorRequest{})
		req.Header().Set("Authorization", "Bearer "+token)

		resp, err := opsClient.GetCurrentOperator(r.Context(), req)
		if err != nil {
			writeGrafanaForbidden(w, "Invalid operator session or JIT verification failed")
			return
		}

		grant := resp.Msg.GetActiveJitGrant()
		if grant == nil || grant.GetStatus() != "ACTIVE" {
			writeGrafanaForbidden(w, "Active JIT elevation lease required to access /grafana/*")
			return
		}

		if grafanaProxy == nil {
			http.Error(w, "Grafana upstream not configured", http.StatusBadGateway)
			return
		}

		proxyReq := r.Clone(r.Context())
		// Prevent spoofing of Grafana auth proxy headers and avoid CSRF port-mismatch on /grafana/api/ds/query
		proxyReq.Header.Del("X-WEBAUTH-USER")
		proxyReq.Header.Del("X-WEBAUTH-ROLE")
		proxyReq.Header.Del("Origin")
		proxyReq.Header.Del("Referer")
		proxyReq.Header.Set("X-WEBAUTH-USER", resp.Msg.GetOperator())
		proxyReq.Header.Set("X-WEBAUTH-ROLE", "Admin")
		if traceID := telemetry.TraceIDFromContext(r.Context()); traceID != "" {
			proxyReq.Header.Set(telemetry.TraceIDHeader, traceID)
		}

		grafanaProxy.ServeHTTP(w, proxyReq)
	}

	mux.HandleFunc("/grafana", grafanaHandler)
	mux.HandleFunc("/grafana/", grafanaHandler)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		cleanPath := filepath.Clean(r.URL.Path)
		cleanPath = strings.TrimPrefix(cleanPath, "/")

		if cleanPath != "" && cleanPath != "." {
			fullPath := filepath.Join(staticDir, cleanPath)
			if info, err := os.Stat(fullPath); err == nil && !info.IsDir() {
				fileServer.ServeHTTP(w, r)
				return
			}
			if ext := filepath.Ext(cleanPath); ext != "" && ext != ".html" {
				http.NotFound(w, r)
				return
			}
		}

		indexPath := filepath.Join(staticDir, "index.html")
		content, err := os.ReadFile(indexPath)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(content)
	})

	maxBytes := cfg.Server.GetMaxRequestBytes()
	if maxBytes <= 0 {
		maxBytes = 10 << 20
	}

	return reg.Middleware(securityHeadersMiddleware(http.MaxBytesHandler(mux, maxBytes)))
}

func extractOpsToken(r *http.Request) string {
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(authHeader, "Bearer ") {
		if tok := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer ")); tok != "" {
			return tok
		}
	}
	if cookie, err := r.Cookie("qs_ops_token"); err == nil && strings.TrimSpace(cookie.Value) != "" {
		return strings.TrimSpace(cookie.Value)
	}
	return ""
}

func writeGrafanaForbidden(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status": "FORBIDDEN",
		"error":  message,
	})
}

func securityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		next.ServeHTTP(w, r)
	})
}
