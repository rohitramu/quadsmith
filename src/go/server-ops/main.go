package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"quadsmith/server-ops/pkg/ops"
	"quadsmith/shared/pkg/config"
	"quadsmith/shared/pkg/opsauth"

	"quadsmith/shared/gen/quadsmith/ops/opsconnect"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "", "Path to ops server configuration textproto file (required)")
	flag.Parse()

	if configPath == "" {
		if envFile := os.Getenv("QS_CONFIG"); envFile != "" {
			configPath = envFile
		} else {
			log.Fatal("server-ops: --config flag is required; no auto-resolution allowed")
		}
	}

	cfg, err := config.LoadOpsServerConfig(configPath)
	if err != nil {
		log.Fatalf("server-ops: configuration validation failed: %v", err)
	}

	port := 8080
	maxReqBytes := 1 << 20
	var readTimeout, writeTimeout, idleTimeout time.Duration
	if cfg.Server != nil {
		if cfg.Server.GetPort() > 0 {
			port = int(cfg.Server.GetPort())
		}
		if cfg.Server.GetMaxRequestBytes() > 0 {
			maxReqBytes = int(cfg.Server.GetMaxRequestBytes())
		}
		if cfg.Server.ReadTimeout != nil {
			readTimeout = cfg.Server.ReadTimeout.AsDuration()
		}
		if cfg.Server.WriteTimeout != nil {
			writeTimeout = cfg.Server.WriteTimeout.AsDuration()
		}
		if cfg.Server.IdleTimeout != nil {
			idleTimeout = cfg.Server.IdleTimeout.AsDuration()
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	publicPool, err := pgxpool.New(ctx, cfg.Database.GetUrl())
	if err != nil {
		log.Fatalf("Unable to connect to public database (%s): %v\n", cfg.Database.GetName(), err)
	}
	defer publicPool.Close()

	internalPool, err := pgxpool.New(ctx, cfg.InternalDatabase.GetUrl())
	if err != nil {
		log.Fatalf("Unable to connect to internal database (%s): %v\n", cfg.InternalDatabase.GetName(), err)
	}
	defer internalPool.Close()

	service := ops.NewOpsService(publicPool, internalPool, cfg)
	interceptor := opsauth.NewOpsInterceptor(internalPool, cfg.BootstrapTokens, cfg.Groups, cfg.Jit)

	handlerOpts := []connect.HandlerOption{
		connect.WithInterceptors(interceptor),
		connect.WithReadMaxBytes(maxReqBytes),
	}

	mux := http.NewServeMux()
	path, handler := opsconnect.NewOpsServiceHandler(service, handlerOpts...)
	mux.Handle(path, handler)

	// Healthcheck endpoint
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	server := &http.Server{
		Addr:              ":" + strconv.Itoa(port),
		Handler:           h2c.NewHandler(mux, &http2.Server{}),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("Starting Quadsmith Ops Server on port %d...", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	<-stop
	log.Println("Shutting down ops server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server shutdown warning: %v", err)
	}

	log.Println("Ops server stopped gracefully")
}
