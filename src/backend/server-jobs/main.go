package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	connect "connectrpc.com/connect"
	pb "quadsmith/shared/gen/quadsmith/ops"
	"quadsmith/shared/gen/quadsmith/ops/opsconnect"
	bconfig "quadsmith/shared/pkg/config"
)

type bearerAuthTransport struct {
	token string
	base  http.RoundTripper
}

func (t *bearerAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+t.token)
	return t.base.RoundTrip(req)
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

	cfg, err := bconfig.LoadJobServerConfig(configPath)
	if err != nil {
		log.Fatalf("server-jobs: configuration validation failed: %v", err)
	}

	transport := &bearerAuthTransport{
		token: cfg.GetAuthToken(),
		base:  http.DefaultTransport,
	}
	httpClient := &http.Client{
		Transport: transport,
		Timeout:   60 * time.Second,
	}

	client := opsconnect.NewOpsServiceClient(httpClient, cfg.GetOpsServerUrl())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	log.Printf("Starting Quadsmith Job Scheduler targeting %s with %d schedule(s)...",
		cfg.GetOpsServerUrl(), len(cfg.GetSchedules()))

	for _, s := range cfg.GetSchedules() {
		interval, err := time.ParseDuration(s.GetInterval())
		if err != nil {
			log.Printf("[JobServer] Invalid interval %q for schedule %q: %v. Defaulting to 1h", s.GetInterval(), s.GetName(), err)
			interval = 1 * time.Hour
		}

		wg.Add(1)
		go func(name, procedure, payload string, d time.Duration) {
			defer wg.Done()
			ticker := time.NewTicker(d)
			defer ticker.Stop()

			log.Printf("[JobServer] Registered schedule %q (procedure: %s, interval: %v)", name, procedure, d)

			for {
				select {
				case <-ctx.Done():
					log.Printf("[JobServer] Stopped schedule %q", name)
					return
				case <-ticker.C:
					log.Printf("[JobServer] Triggering scheduled job %q (%s)...", name, procedure)
					req := connect.NewRequest(&pb.TriggerJobRequest{
						JobName:     procedure,
						PayloadJson: payload,
					})

					t0 := time.Now()
					res, err := client.TriggerJob(ctx, req)
					duration := time.Since(t0)

					if err != nil {
						log.Printf("[JobServer] Scheduled job %q failed after %v: %v", name, duration, err)
					} else {
						log.Printf("[JobServer] Scheduled job %q succeeded after %v (call_id: %s)",
							name, duration, res.Msg.GetCallId())
					}
				}
			}
		}(s.GetName(), s.GetProcedure(), s.GetPayloadJson(), interval)
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	<-stop
	log.Println("Shutting down job server...")
	cancel()
	wg.Wait()
	log.Println("Job server stopped gracefully")
}
