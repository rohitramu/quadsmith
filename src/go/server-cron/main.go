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
	"strings"
	"syscall"
	"time"

	"connectrpc.com/connect"
	"github.com/robfig/cron/v3"

	"quadsmith/shared/pkg/authcontext"
	"quadsmith/shared/pkg/config"
	"quadsmith/shared/pkg/telemetry"

	"google.golang.org/protobuf/types/known/durationpb"
	configpb "quadsmith/shared/gen/quadsmith/config"
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

func newCronHTTPHandler(cfg *configpb.CronServerConfig, reg *telemetry.Registry) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", reg.HealthzHandler())
	mux.HandleFunc("/metrics", reg.MetricsHandler())
	return reg.Middleware(mux)
}

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "", "Path to cron server configuration textproto file (required)")
	flag.Parse()

	if configPath == "" {
		if envFile := os.Getenv("QS_CONFIG"); envFile != "" {
			configPath = envFile
		} else {
			log.Fatal("server-cron: --config flag is required; no auto-resolution allowed")
		}
	}

	cfg, err := config.LoadCronServerConfig(configPath)
	if err != nil {
		log.Fatalf("server-cron: configuration validation failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	httpClient := telemetry.NewTracedHTTPClient(60*time.Second, telemetry.InternalHopToken(), cfg.GetAuthToken())

	opsClient := opsconnect.NewOpsServiceClient(httpClient, cfg.GetOpsServerUrl())

	cronRunner := cron.New(cron.WithLocation(time.UTC))

	reg := telemetry.NewRegistry("server-cron", cfg.GetEnvironment(), false)
	reg.AddCheck("scheduler_loop", func(_ context.Context) error {
		return nil
	})
	reg.AddUpstreamCheck("upstream:server-ops-api", telemetry.HTTPHealthCheck(strings.TrimRight(cfg.GetOpsServerUrl(), "/")+"/healthz"))

	runTaskWithTrace := func(taskName string, fn func(taskCtx context.Context) error) {
		traceID := authcontext.NewUUIDv7()
		taskCtx := telemetry.WithTraceID(ctx, traceID)
		t0 := time.Now()
		err := fn(taskCtx)
		dur := time.Since(t0)
		statusCode := 200
		if err != nil {
			statusCode = 500
			log.Printf("[CronRunner] task=%s trace_id=%s failed after %v: %v", taskName, traceID, dur, err)
		} else {
			log.Printf("[CronRunner] task=%s trace_id=%s completed in %v", taskName, traceID, dur)
		}
		reg.IncSchedulerRun(err != nil)
		reg.RecordRequest(traceID, "cron:"+taskName, "CRON", statusCode, err == nil, float64(dur.Microseconds())/1000.0)
	}

	var registeredTasks int32

	if s := cfg.GetCleanupExpiredOtps(); s != nil && s.GetEnabled() {
		spec, err := scheduleToCron(s.GetCron(), s.GetInterval())
		if err != nil {
			log.Fatalf("server-cron: invalid schedule for cleanup_expired_otps: %v", err)
		}
		_, err = cronRunner.AddFunc(spec, func() {
			runTaskWithTrace("cleanup_expired_otps", func(taskCtx context.Context) error {
				_, err := opsClient.CleanupExpiredOtps(taskCtx, connect.NewRequest(&opspb.CleanupExpiredOtpsRequest{}))
				return err
			})
		})
		if err != nil {
			log.Fatalf("server-cron: failed to register cleanup_expired_otps cron %q: %v", spec, err)
		}
		registeredTasks++
		log.Printf("[CronRunner] Registered cleanup_expired_otps on schedule %q", spec)
	}

	if s := cfg.GetCleanupExpiredTokens(); s != nil && s.GetEnabled() {
		spec, err := scheduleToCron(s.GetCron(), s.GetInterval())
		if err != nil {
			log.Fatalf("server-cron: invalid schedule for cleanup_expired_tokens: %v", err)
		}
		_, err = cronRunner.AddFunc(spec, func() {
			runTaskWithTrace("cleanup_expired_tokens", func(taskCtx context.Context) error {
				_, err := opsClient.CleanupExpiredTokens(taskCtx, connect.NewRequest(&opspb.CleanupExpiredTokensRequest{}))
				return err
			})
		})
		if err != nil {
			log.Fatalf("server-cron: failed to register cleanup_expired_tokens cron %q: %v", spec, err)
		}
		registeredTasks++
		log.Printf("[CronRunner] Registered cleanup_expired_tokens on schedule %q", spec)
	}

	if s := cfg.GetCleanupOrphanedBlobs(); s != nil && s.GetEnabled() {
		spec, err := scheduleToCron(s.GetCron(), s.GetInterval())
		if err != nil {
			log.Fatalf("server-cron: invalid schedule for cleanup_orphaned_blobs: %v", err)
		}
		_, err = cronRunner.AddFunc(spec, func() {
			runTaskWithTrace("cleanup_orphaned_blobs", func(taskCtx context.Context) error {
				_, err := opsClient.CleanupOrphanedBlobs(taskCtx, connect.NewRequest(&opspb.CleanupOrphanedBlobsRequest{}))
				return err
			})
		})
		if err != nil {
			log.Fatalf("server-cron: failed to register cleanup_orphaned_blobs cron %q: %v", spec, err)
		}
		registeredTasks++
		log.Printf("[CronRunner] Registered cleanup_orphaned_blobs on schedule %q", spec)
	}

	if s := cfg.GetPurgeSoftDeletedAccounts(); s != nil && s.GetEnabled() {
		spec, err := scheduleToCron(s.GetCron(), s.GetInterval())
		if err != nil {
			log.Fatalf("server-cron: invalid schedule for purge_soft_deleted_accounts: %v", err)
		}
		_, err = cronRunner.AddFunc(spec, func() {
			runTaskWithTrace("purge_soft_deleted_accounts", func(taskCtx context.Context) error {
				_, err := opsClient.PurgeSoftDeletedAccounts(taskCtx, connect.NewRequest(&opspb.PurgeSoftDeletedAccountsRequest{}))
				return err
			})
		})
		if err != nil {
			log.Fatalf("server-cron: failed to register purge_soft_deleted_accounts cron %q: %v", spec, err)
		}
		registeredTasks++
		log.Printf("[CronRunner] Registered purge_soft_deleted_accounts on schedule %q", spec)
	}

	if s := cfg.GetSendUsernameReleaseWarnings(); s != nil && s.GetEnabled() {
		spec, err := scheduleToCron(s.GetCron(), s.GetInterval())
		if err != nil {
			log.Fatalf("server-cron: invalid schedule for send_username_release_warnings: %v", err)
		}
		_, err = cronRunner.AddFunc(spec, func() {
			runTaskWithTrace("send_username_release_warnings", func(taskCtx context.Context) error {
				_, err := opsClient.SendUsernameReleaseWarnings(taskCtx, connect.NewRequest(&opspb.SendUsernameReleaseWarningsRequest{}))
				return err
			})
		})
		if err != nil {
			log.Fatalf("server-cron: failed to register send_username_release_warnings cron %q: %v", spec, err)
		}
		registeredTasks++
		log.Printf("[CronRunner] Registered send_username_release_warnings on schedule %q", spec)
	}

	reg.SetSchedulerRegisteredTasks(registeredTasks)
	cronRunner.Start()

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

	httpServer := &http.Server{
		Addr:              ":" + strconv.Itoa(port),
		Handler:           newCronHTTPHandler(cfg, reg),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}

	go func() {
		log.Printf("Starting Quadsmith Cron Server on port %d (target: %s)...", port, cfg.GetOpsServerUrl())
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	<-stop
	log.Println("Shutting down cron server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	_ = httpServer.Shutdown(shutdownCtx)

	cronCtx := cronRunner.Stop()
	select {
	case <-cronCtx.Done():
		log.Println("Cron runner stopped")
	case <-time.After(5 * time.Second):
		log.Println("Cron runner shutdown timed out")
	}

	cancel()
	log.Println("Cron server stopped gracefully")
}
