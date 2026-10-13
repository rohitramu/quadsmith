package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	bconfig "quadsmith/shared/pkg/config"
	"quadsmith/shared/pkg/static"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "", "Path to web server configuration textproto file (required)")
	flag.Parse()

	if configPath == "" {
		if envFile := os.Getenv("QS_CONFIG"); envFile != "" {
			configPath = envFile
		} else {
			log.Fatal("server-web: --config flag is required; no auto-resolution allowed")
		}
	}

	cfg, err := bconfig.LoadWebServerConfig(configPath)
	if err != nil {
		log.Fatalf("server-web: configuration validation failed: %v", err)
	}

	var pool *pgxpool.Pool
	if cfg.Database != nil && cfg.Database.Url != "" {
		p, err := pgxpool.New(context.Background(), cfg.Database.Url)
		if err != nil {
			log.Printf("server-web: warning: unable to connect to database (%s): %v; dynamic meta tags and session verification disabled", cfg.Database.Name, err)
		} else {
			pool = p
			defer pool.Close()
		}
	}

	clientCfg := map[string]string{
		"environment": cfg.Environment,
		"apiUrl":      cfg.ApiUrl,
		"appName":     cfg.AppName,
	}
	clientCfgJSONBytes, _ := json.Marshal(clientCfg)
	clientCfgJSON := string(clientCfgJSONBytes)

	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})

	// Public config endpoint
	mux.HandleFunc("/api/v1/config", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(clientCfgJSONBytes)
	})

	// Reverse proxy to API server
	if cfg.ApiUrl != "" {
		targetURL, err := url.Parse(cfg.ApiUrl)
		if err != nil {
			log.Fatalf("server-web: invalid api_url %q: %v", cfg.ApiUrl, err)
		}
		proxy := httputil.NewSingleHostReverseProxy(targetURL)
		mux.Handle("/quadsmith.api.", proxy)
		mux.Handle("/api/blobs", proxy)
		mux.Handle("/api/blobs/", proxy)
		mux.Handle("/blobs/", proxy)
	}

	// SPA & static asset handler
	mux.Handle("/", static.ServeSPAWithConfig(cfg.Server.StaticDir, pool, clientCfgJSON))

	port := strconv.Itoa(int(cfg.Server.Port))
	fmt.Printf("Starting Quadsmith Web Server on port %s...\n", port)

	maxBytes := cfg.Server.MaxRequestBytes
	if maxBytes <= 0 {
		maxBytes = 10 << 20 // 10MB
	}

	rootHandler := securityHeadersMiddleware(dynamicMaxBytesHandler(mux, maxBytes))

	readTimeout := 30 * time.Second
	if cfg.Server.ReadTimeout != nil && cfg.Server.ReadTimeout.AsDuration() > 0 {
		readTimeout = cfg.Server.ReadTimeout.AsDuration()
	}
	writeTimeout := 60 * time.Second
	if cfg.Server.WriteTimeout != nil && cfg.Server.WriteTimeout.AsDuration() > 0 {
		writeTimeout = cfg.Server.WriteTimeout.AsDuration()
	}
	idleTimeout := 120 * time.Second
	if cfg.Server.IdleTimeout != nil && cfg.Server.IdleTimeout.AsDuration() > 0 {
		idleTimeout = cfg.Server.IdleTimeout.AsDuration()
	}

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           h2c.NewHandler(rootHandler, &http2.Server{}),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server-web failed: %v", err)
	}
}

func dynamicMaxBytesHandler(next http.Handler, defaultLimit int64) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		limit := defaultLimit
		if strings.HasPrefix(r.URL.Path, "/api/blobs") {
			limit = 100 << 20 // 100MB for blob uploads
		}
		http.MaxBytesHandler(next, limit).ServeHTTP(w, r)
	})
}

func securityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'wasm-unsafe-eval' https://www.gstatic.com; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; frame-src https://www.youtube.com https://www.youtube-nocookie.com; frame-ancestors 'none'; object-src 'none'; base-uri 'self';")
		if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		next.ServeHTTP(w, r)
	})
}
