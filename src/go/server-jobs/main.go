package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/robfig/cron/v3"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"quadsmith/server-jobs/pkg/jobs"
	"quadsmith/shared/pkg/config"
	"quadsmith/shared/pkg/opsauth"

	"google.golang.org/protobuf/types/known/durationpb"
	"quadsmith/shared/gen/quadsmith/jobs/jobsconnect"
	opspb "quadsmith/shared/gen/quadsmith/ops"
	"quadsmith/shared/gen/quadsmith/ops/opsconnect"
)

type bearerAuthTransport struct {
	token string
	base  http.RoundTripper
}

func (t *bearerAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+t.token)
	return t.base.RoundTrip(req)
}

func scheduleToCron(cronStr string, interval *durationpb.Duration) (string, error) {
	if cronStr != "" {
		return cronStr, nil
	}
	if interval != nil {
		return config.DurationPbToCron(interval)
	}
	return "", fmt.Errorf("no schedule provided")
}

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "", "Path to job server configuration textproto file (required)")
	flag.Parse()

	if configPath == "" {
		if envFile := os.Getenv("QS_CONFIG"); envFile != "" {
			configPath = envFile
		} else {
			log.Fatal("server-jobs: --config flag is required; no auto-resolution allowed")
		}
	}

	cfg, err := config.LoadJobServerConfig(configPath)
	if err != nil {
		log.Fatalf("server-jobs: configuration validation failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	poolCfg, err := pgxpool.ParseConfig(cfg.GetInternalDatabase().GetUrl())
	if err != nil {
		log.Fatalf("Invalid internal database URL (%s): %v\n", cfg.GetInternalDatabase().GetName(), err)
	}
	if cfg.GetInternalDatabase().GetMaxConnections() > 0 {
		poolCfg.MaxConns = cfg.GetInternalDatabase().GetMaxConnections()
	}
	if cfg.GetInternalDatabase().GetMinConnections() > 0 {
		poolCfg.MinConns = cfg.GetInternalDatabase().GetMinConnections()
	}
	dbPool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		log.Fatalf("Unable to connect to internal database (%s): %v\n", cfg.GetInternalDatabase().GetName(), err)
	}
	defer dbPool.Close()

	interceptor := opsauth.NewOpsInterceptor(dbPool, cfg.BootstrapTokens, cfg.Groups, nil)
	service := jobs.NewService(dbPool, cfg)

	maxReqBytes := 1 << 20
	if cfg.Server != nil && cfg.Server.GetMaxRequestBytes() > 0 {
		maxReqBytes = int(cfg.Server.GetMaxRequestBytes())
	}

	mux := http.NewServeMux()
	path, handler := jobsconnect.NewJobsServiceHandler(
		service,
		connect.WithInterceptors(interceptor),
		connect.WithReadMaxBytes(maxReqBytes),
	)
	mux.Handle(path, handler)

	// Healthcheck endpoint
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	port := 8080
	var readTimeout, writeTimeout, idleTimeout time.Duration
	if cfg.Server != nil {
		if cfg.Server.GetPort() > 0 {
			port = int(cfg.Server.GetPort())
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

	server := &http.Server{
		Addr:              ":" + strconv.Itoa(port),
		Handler:           h2c.NewHandler(mux, &http2.Server{}),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}

	transport := &bearerAuthTransport{
		token: cfg.GetAuthToken(),
		base:  http.DefaultTransport,
	}
	httpClient := &http.Client{
		Transport: transport,
		Timeout:   60 * time.Second,
	}

	opsClient := opsconnect.NewOpsServiceClient(httpClient, cfg.GetOpsServerUrl())

	cronRunner := cron.New(cron.WithLocation(time.UTC))

	if s := cfg.GetCleanupExpiredOtps(); s != nil && s.GetEnabled() {
		spec, err := scheduleToCron(s.GetCron(), s.GetInterval())
		if err != nil {
			log.Fatalf("server-jobs: invalid schedule for cleanup_expired_otps: %v", err)
		}
		_, err = cronRunner.AddFunc(spec, func() {
			log.Printf("[JobRunner] Triggering cleanup_expired_otps...")
			t0 := time.Now()
			res, err := opsClient.CleanupExpiredOtps(ctx, connect.NewRequest(&opspb.CleanupExpiredOtpsRequest{}))
			dur := time.Since(t0)
			if err != nil {
				log.Printf("[JobRunner] cleanup_expired_otps failed after %v: %v", dur, err)
			} else {
				log.Printf("[JobRunner] cleanup_expired_otps completed in %v (cleaned: %d)", dur, res.Msg.GetCleanedCount())
			}
		})
		if err != nil {
			log.Fatalf("server-jobs: failed to register cleanup_expired_otps cron %q: %v", spec, err)
		}
		log.Printf("[JobRunner] Registered cleanup_expired_otps on schedule %q", spec)
	}

	if s := cfg.GetCleanupExpiredTokens(); s != nil && s.GetEnabled() {
		spec, err := scheduleToCron(s.GetCron(), s.GetInterval())
		if err != nil {
			log.Fatalf("server-jobs: invalid schedule for cleanup_expired_tokens: %v", err)
		}
		_, err = cronRunner.AddFunc(spec, func() {
			log.Printf("[JobRunner] Triggering cleanup_expired_tokens...")
			t0 := time.Now()
			res, err := opsClient.CleanupExpiredTokens(ctx, connect.NewRequest(&opspb.CleanupExpiredTokensRequest{}))
			dur := time.Since(t0)
			if err != nil {
				log.Printf("[JobRunner] cleanup_expired_tokens failed after %v: %v", dur, err)
			} else {
				log.Printf("[JobRunner] cleanup_expired_tokens completed in %v (cleaned: %d)", dur, res.Msg.GetCleanedCount())
			}
		})
		if err != nil {
			log.Fatalf("server-jobs: failed to register cleanup_expired_tokens cron %q: %v", spec, err)
		}
		log.Printf("[JobRunner] Registered cleanup_expired_tokens on schedule %q", spec)
	}

	if s := cfg.GetCleanupOrphanedBlobs(); s != nil && s.GetEnabled() {
		spec, err := scheduleToCron(s.GetCron(), s.GetInterval())
		if err != nil {
			log.Fatalf("server-jobs: invalid schedule for cleanup_orphaned_blobs: %v", err)
		}
		_, err = cronRunner.AddFunc(spec, func() {
			log.Printf("[JobRunner] Triggering cleanup_orphaned_blobs...")
			t0 := time.Now()
			res, err := opsClient.CleanupOrphanedBlobs(ctx, connect.NewRequest(&opspb.CleanupOrphanedBlobsRequest{}))
			dur := time.Since(t0)
			if err != nil {
				log.Printf("[JobRunner] cleanup_orphaned_blobs failed after %v: %v", dur, err)
			} else {
				log.Printf("[JobRunner] cleanup_orphaned_blobs completed in %v (cleaned: %d)", dur, res.Msg.GetCleanedCount())
			}
		})
		if err != nil {
			log.Fatalf("server-jobs: failed to register cleanup_orphaned_blobs cron %q: %v", spec, err)
		}
		log.Printf("[JobRunner] Registered cleanup_orphaned_blobs on schedule %q", spec)
	}

	if s := cfg.GetPurgeSoftDeletedAccounts(); s != nil && s.GetEnabled() {
		spec, err := scheduleToCron(s.GetCron(), s.GetInterval())
		if err != nil {
			log.Fatalf("server-jobs: invalid schedule for purge_soft_deleted_accounts: %v", err)
		}
		_, err = cronRunner.AddFunc(spec, func() {
			log.Printf("[JobRunner] Triggering purge_soft_deleted_accounts...")
			t0 := time.Now()
			res, err := opsClient.PurgeSoftDeletedAccounts(ctx, connect.NewRequest(&opspb.PurgeSoftDeletedAccountsRequest{}))
			dur := time.Since(t0)
			if err != nil {
				log.Printf("[JobRunner] purge_soft_deleted_accounts failed after %v: %v", dur, err)
			} else {
				log.Printf("[JobRunner] purge_soft_deleted_accounts completed in %v (purged: %d)", dur, res.Msg.GetPurgedCount())
			}
		})
		if err != nil {
			log.Fatalf("server-jobs: failed to register purge_soft_deleted_accounts cron %q: %v", spec, err)
		}
		log.Printf("[JobRunner] Registered purge_soft_deleted_accounts on schedule %q", spec)
	}

	if s := cfg.GetSendUsernameReleaseWarnings(); s != nil && s.GetEnabled() {
		spec, err := scheduleToCron(s.GetCron(), s.GetInterval())
		if err != nil {
			log.Fatalf("server-jobs: invalid schedule for send_username_release_warnings: %v", err)
		}
		_, err = cronRunner.AddFunc(spec, func() {
			log.Printf("[JobRunner] Triggering send_username_release_warnings...")
			t0 := time.Now()
			res, err := opsClient.SendUsernameReleaseWarnings(ctx, connect.NewRequest(&opspb.SendUsernameReleaseWarningsRequest{}))
			dur := time.Since(t0)
			if err != nil {
				log.Printf("[JobRunner] send_username_release_warnings failed after %v: %v", dur, err)
			} else {
				log.Printf("[JobRunner] send_username_release_warnings completed in %v (warnings: %d)", dur, res.Msg.GetWarningCount())
			}
		})
		if err != nil {
			log.Fatalf("server-jobs: failed to register send_username_release_warnings cron %q: %v", spec, err)
		}
		log.Printf("[JobRunner] Registered send_username_release_warnings on schedule %q", spec)
	}

	if s := cfg.GetRunDatabaseMaintenance(); s != nil && s.GetEnabled() {
		spec, err := scheduleToCron(s.GetCron(), s.GetInterval())
		if err != nil {
			log.Fatalf("server-jobs: invalid schedule for run_database_maintenance: %v", err)
		}
		fullVacuum := s.GetFullVacuum()
		_, err = cronRunner.AddFunc(spec, func() {
			log.Printf("[JobRunner] Triggering run_database_maintenance (full_vacuum=%v)...", fullVacuum)
			t0 := time.Now()
			res, err := opsClient.RunDatabaseMaintenance(ctx, connect.NewRequest(&opspb.RunDatabaseMaintenanceRequest{
				FullVacuum: fullVacuum,
			}))
			dur := time.Since(t0)
			if err != nil {
				log.Printf("[JobRunner] run_database_maintenance failed after %v: %v", dur, err)
			} else {
				log.Printf("[JobRunner] run_database_maintenance completed in %v (%s)", dur, res.Msg.GetMessage())
			}
		})
		if err != nil {
			log.Fatalf("server-jobs: failed to register run_database_maintenance cron %q: %v", spec, err)
		}
		log.Printf("[JobRunner] Registered run_database_maintenance on schedule %q", spec)
	}

	cronRunner.Start()

	go func() {
		log.Printf("Starting Quadsmith Job Server on port %d...", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	<-stop
	log.Println("Shutting down job server...")

	cronCtx := cronRunner.Stop()
	select {
	case <-cronCtx.Done():
		log.Println("Cron runner stopped")
	case <-time.After(5 * time.Second):
		log.Println("Cron runner shutdown timed out")
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server shutdown warning: %v", err)
	}

	cancel()
	log.Println("Job server stopped gracefully")
}
