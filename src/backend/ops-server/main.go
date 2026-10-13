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

	connect "connectrpc.com/connect"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"quadsmith/api/gen/quadsmith/ops/opsconnect"
	"quadsmith/api/pkg/ops"
	bconfig "quadsmith/backend/pkg/config"
)

func main() {
	var configPath string
	var envName string
	flag.StringVar(&configPath, "config", "", "Path to ops server configuration textproto file")
	flag.StringVar(&envName, "env", "", "Active environment (sandbox, test, staging, prod)")
	flag.Parse()

	cfg, err := bconfig.LoadOpsServerConfig(configPath, envName)
	if err != nil {
		log.Fatalf("Failed to load ops server config: %v", err)
	}

	port := 8081
	if cfg.Server != nil && cfg.Server.Port > 0 {
		port = int(cfg.Server.Port)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	publicPool, err := pgxpool.New(ctx, cfg.Database.Url)
	if err != nil {
		log.Fatalf("Unable to connect to public database (%s): %v\n", cfg.Database.Name, err)
	}
	defer publicPool.Close()

	internalPool, err := pgxpool.New(ctx, cfg.InternalDatabase.Url)
	if err != nil {
		log.Fatalf("Unable to connect to internal database (%s): %v\n", cfg.InternalDatabase.Name, err)
	}
	defer internalPool.Close()

	service := ops.NewOpsService(publicPool, internalPool, cfg)
	interceptor := ops.NewOpsInterceptor(internalPool, cfg)

	handlerOpts := []connect.HandlerOption{
		connect.WithInterceptors(interceptor),
		connect.WithReadMaxBytes(1 << 20),
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
