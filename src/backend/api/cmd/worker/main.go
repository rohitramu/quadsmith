package main

import (
	"context"
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

	"quadsmith/api/gen/quadsmith/quadsmithconnect"
	"quadsmith/api/pkg/auth"
	"quadsmith/api/pkg/worker"
)

func main() {
	dbUrl := os.Getenv("DATABASE_URL")
	if dbUrl == "" {
		dbUrl = "postgres://postgres:postgres@localhost:5432/quadsmith"
	}

	port := os.Getenv("WORKER_PORT")
	if port == "" {
		port = "8081"
	}

	concurrency := 5
	if cStr := os.Getenv("WORKER_CONCURRENCY"); cStr != "" {
		if c, err := strconv.Atoi(cStr); err == nil && c > 0 {
			concurrency = c
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := pgxpool.New(ctx, dbUrl)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer pool.Close()

	// 1. Initialize Task Registry
	reg := worker.NewRegistry()

	// 2. Register Built-in Tasks
	auth.RegisterWorkers(reg, pool)

	// Periodic 24h task: prune completed background jobs older than 7d and failed jobs older than 30d
	worker.RegisterPeriodic(reg, "worker.cleanup_completed_jobs", 24*time.Hour, worker.EmptyPayload{}, func(ctx context.Context, job worker.Job[worker.EmptyPayload]) error {
		cmd1, err1 := pool.Exec(ctx, `DELETE FROM background_jobs WHERE status = 'completed' AND updated_at < NOW() - INTERVAL '7 days';`)
		if err1 != nil {
			log.Printf("[WorkerCleanup] Error pruning completed jobs: %v", err1)
		}
		cmd2, err2 := pool.Exec(ctx, `DELETE FROM background_jobs WHERE status = 'failed' AND updated_at < NOW() - INTERVAL '30 days';`)
		if err2 != nil {
			log.Printf("[WorkerCleanup] Error pruning failed jobs: %v", err2)
		}
		if rows := cmd1.RowsAffected() + cmd2.RowsAffected(); rows > 0 {
			log.Printf("[WorkerCleanup] Pruned %d old background jobs", rows)
		}
		return nil
	})

	// 3. Initialize Worker Pool & Scheduler
	workerPool := worker.NewPool(pool, reg, worker.WithConcurrency(concurrency))
	scheduler := worker.NewScheduler(pool, reg)

	// 4. Start execution loops
	workerPool.Start(ctx)
	scheduler.Start(ctx)

	log.Printf("[WorkerDaemon] Started worker pool (concurrency: %d) and scheduler with %d registered tasks",
		concurrency, len(reg.TaskTypes()))

	// 5. ConnectRPC Service HTTP Server
	mux := http.NewServeMux()
	workerService := worker.NewWorkerService(pool, reg, workerPool)
	path, handler := quadsmithconnect.NewWorkerServiceHandler(workerService)
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
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		log.Printf("[WorkerDaemon] Listening on port %s...", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	// 6. Graceful Shutdown on SIGINT / SIGTERM
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	select {
	case err := <-serverErr:
		log.Fatalf("[WorkerDaemon] HTTP server error: %v", err)
	case sig := <-sigChan:
		log.Printf("[WorkerDaemon] Received signal %v, shutting down gracefully...", sig)
	}

	cancel()

	// Drain HTTP server
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	_ = server.Shutdown(shutdownCtx)

	// Stop scheduler and wait for active jobs to complete
	scheduler.Stop()
	workerPool.Stop()

	log.Printf("[WorkerDaemon] Shutdown complete.")
}
