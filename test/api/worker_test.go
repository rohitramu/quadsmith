package api

import (
	"context"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	connect "connectrpc.com/connect"
	"github.com/jackc/pgx/v5/pgxpool"

	pb "quadsmith/api/gen/quadsmith_internal"
	"quadsmith/api/gen/quadsmith_internal/quadsmith_internalconnect"
)

func getWorkerURL() string {
	if url := os.Getenv("QS_WORKER_URL"); url != "" {
		return url
	}
	return "http://127.0.0.1:8081"
}

func TestWorkerDaemonStatusAndTrigger(t *testing.T) {
	workerUrl := getWorkerURL()
	client := &http.Client{Timeout: 2 * time.Second}
	_, err := client.Get(workerUrl + "/healthz")
	if err != nil {
		t.Fatalf("Worker daemon not running at %s: %v", workerUrl, err)
	}

	workerClient := quadsmith_internalconnect.NewWorkerServiceClient(
		http.DefaultClient,
		workerUrl,
	)

	ctx := context.Background()

	// 1. Verify Status
	statusRes, err := workerClient.GetWorkerStatus(ctx, connect.NewRequest(&pb.GetWorkerStatusRequest{}))
	if err != nil {
		t.Fatalf("Failed to get worker status: %v", err)
	}

	if !statusRes.Msg.GetIsHealthy() {
		t.Errorf("Expected worker to be healthy, got unhealthy")
	}

	tasks := statusRes.Msg.GetRegisteredTasks()
	expectedTasks := []string{
		"auth.cleanup_expired_otps",
		"auth.cleanup_expired_tokens",
		"auth.username_release_warning",
		"blob.cleanup_orphaned_blobs",
		"worker.cleanup_completed_jobs",
	}

	for _, exp := range expectedTasks {
		found := false
		for _, act := range tasks {
			if act == exp {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected task %q in registered tasks: %v", exp, tasks)
		}
	}

	// 2. Trigger a test task
	triggerRes, err := workerClient.TriggerJob(ctx, connect.NewRequest(&pb.TriggerJobRequest{
		JobType:     "auth.cleanup_expired_otps",
		PayloadJson: "{}",
	}))
	if err != nil {
		t.Fatalf("Failed to trigger job: %v", err)
	}

	jobUUID := triggerRes.Msg.GetJobUuid()
	if jobUUID == "" {
		t.Fatalf("Expected non-empty job UUID")
	}

	// 3. Poll until completed or timeout
	var job *pb.BackgroundJob
	for i := 0; i < 20; i++ {
		time.Sleep(200 * time.Millisecond)
		jobRes, err := workerClient.GetJob(ctx, connect.NewRequest(&pb.GetJobRequest{JobUuid: jobUUID}))
		if err == nil {
			job = jobRes.Msg.GetJob()
			if job.GetStatus() == pb.JobStatus_JOB_STATUS_COMPLETED {
				break
			}
		}
	}

	if job == nil || job.GetStatus() != pb.JobStatus_JOB_STATUS_COMPLETED {
		t.Errorf("Expected job %s to reach COMPLETED, got status: %v", jobUUID, job.GetStatus())
	}

	if len(job.GetAttempts()) == 0 {
		t.Errorf("Expected at least 1 attempt recorded for job %s", jobUUID)
	} else {
		attempt := job.GetAttempts()[0]
		if attempt.GetStatus() != pb.AttemptStatus_ATTEMPT_STATUS_COMPLETED {
			t.Errorf("Expected attempt status COMPLETED, got %v (error: %s)", attempt.GetStatus(), attempt.GetErrorMessage())
		}
	}

	// 4. Verify Database Isolation: Job lives in quadsmith_internal, NOT in quadsmith
	if internalDBURL := os.Getenv("QS_INTERNAL_DATABASE_URL"); internalDBURL != "" {
		internalPool, err := pgxpool.New(ctx, internalDBURL)
		if err == nil {
			defer internalPool.Close()
			var jobCount int
			_ = internalPool.QueryRow(ctx, "SELECT count(*) FROM background_jobs WHERE uuid = $1", jobUUID).Scan(&jobCount)
			if jobCount != 1 {
				t.Errorf("Expected job %s to exist in quadsmith_internal background_jobs, found count %d", jobUUID, jobCount)
			}
		}
	}

	if publicDBURL := os.Getenv("QS_DATABASE_URL"); publicDBURL != "" {
		publicPool, err := pgxpool.New(ctx, publicDBURL)
		if err == nil {
			defer publicPool.Close()
			var tableCount int
			_ = publicPool.QueryRow(ctx, "SELECT count(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'background_jobs'").Scan(&tableCount)
			if tableCount != 0 {
				t.Errorf("Expected background_jobs table to NOT exist in quadsmith public database, found %d", tableCount)
			}
		}
	}
}

func resolveQSOpsPath(t *testing.T) string {
	t.Helper()
	for _, p := range []string{"../../bin/qsops", "../bin/qsops", "bin/qsops"} {
		abs, err := filepath.Abs(p)
		if err == nil {
			if _, err := os.Stat(abs); err == nil {
				return abs
			}
		}
	}
	t.Fatalf("Failed to resolve qsops binary path (checked ../../bin/qsops, ../bin/qsops, bin/qsops)")
	return ""
}

func TestQsopsCLIExecution(t *testing.T) {
	workerUrl := getWorkerURL()
	qsopsPath := resolveQSOpsPath(t)
	cmd := exec.Command(qsopsPath, "workers", "status", "--worker-url", workerUrl)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run qsops workers status: %v, output: %s", err, string(output))
	}

	outStr := string(output)
	if !containsString(outStr, "Worker Daemon Health: HEALTHY") {
		t.Errorf("Expected HEALTHY in qsops output: %s", outStr)
	}
	if !containsString(outStr, "auth.cleanup_expired_otps") {
		t.Errorf("Expected auth.cleanup_expired_otps in registered tasks output: %s", outStr)
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || (len(s) > len(substr) && stringContains(s, substr)))
}

func stringContains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
