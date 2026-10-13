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

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"quadsmith/api/gen/quadsmith_internal/quadsmith_internalconnect"
	"quadsmith/api/pkg/auth"
	"quadsmith/api/pkg/blob"
	"quadsmith/api/pkg/worker"
	bconfig "quadsmith/backend/pkg/config"
)

func main() {
	var configPath string
	var envName string
	flag.StringVar(&configPath, "config", "", "Path to backend configuration textproto file")
	flag.StringVar(&envName, "env", "", "Active environment (sandbox, test, staging, prod)")
	flag.Parse()

	cfg, err := bconfig.LoadWorkerConfig(configPath, envName)
	if err != nil {
		log.Fatalf("%v", err)
	}

	port := strconv.Itoa(int(cfg.Worker.Port))
	concurrency := int(cfg.Worker.Concurrency)

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

	// 1. Initialize Task Registry
	reg := worker.NewRegistry()

	// 2. Register Built-in Tasks (domain tasks operate on public database)
	auth.RegisterWorkers(reg, publicPool)
	blob.RegisterWorkers(reg, publicPool)

	// Periodic 24h task: prune completed background jobs older than 7d and failed jobs older than 30d (internal database)
	worker.RegisterPeriodic(reg, "worker.cleanup_completed_jobs", 24*time.Hour, worker.EmptyPayload{}, func(ctx context.Context, job worker.Job[worker.EmptyPayload]) error {
		cmd1, err1 := internalPool.Exec(ctx, `DELETE FROM background_jobs WHERE status = 'completed' AND updated_at < NOW() - INTERVAL '7 days';`)
		if err1 != nil {
			log.Printf("[WorkerCleanup] Error pruning completed jobs: %v", err1)
		}
		cmd2, err2 := internalPool.Exec(ctx, `DELETE FROM background_jobs WHERE status = 'failed' AND updated_at < NOW() - INTERVAL '30 days';`)
		if err2 != nil {
			log.Printf("[WorkerCleanup] Error pruning failed jobs: %v", err2)
		}
		if rows := cmd1.RowsAffected() + cmd2.RowsAffected(); rows > 0 {
			log.Printf("[WorkerCleanup] Pruned %d old background jobs", rows)
		}
		return nil
	})

	// 3. Initialize Worker Pool & Scheduler (operate on internal database)
	workerPool := worker.NewPool(internalPool, reg, worker.WithConcurrency(concurrency))
	scheduler := worker.NewScheduler(internalPool, reg)

	// 4. Start execution loops
	workerPool.Start(ctx)
	scheduler.Start(ctx)

	log.Printf("[WorkerDaemon] Started worker pool (concurrency: %d) and scheduler with %d registered tasks",
		concurrency, len(reg.TaskTypes()))

	// 5. ConnectRPC Service HTTP Server
	mux := http.NewServeMux()
	workerService := worker.NewWorkerService(internalPool, reg, workerPool)
	path, handler := quadsmith_internalconnect.NewWorkerServiceHandler(workerService)
	mux.Handle(path, handler)

	// Simple healthcheck endpoint
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           h2c.NewHandler(mux, &http2.Server{}),
		ReadHeaderTimeout: 5 * time.Second,
	}

	// Graceful shutdown handling
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("[WorkerDaemon] Starting Worker ConnectRPC server on port %s...", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[WorkerDaemon] HTTP server error: %v", err)
		}
	}()

	<-stop
	log.Println("[WorkerDaemon] Shutting down background worker daemon...")

	// Stop scheduler and pool first
	cancel()
	scheduler.Stop()
	workerPool.Stop()

	// Shutdown HTTP server with 5s timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("[WorkerDaemon] Server shutdown warning: %v", err)
	}

	log.Println("[WorkerDaemon] Worker daemon stopped gracefully")
}
