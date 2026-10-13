package worker

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	pb "quadsmith/api/gen/quadsmith_internal"
)

type samplePayload struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

func TestRegistryGenericHandler(t *testing.T) {
	reg := NewRegistry()

	var received Job[samplePayload]
	var handled bool

	Register(reg, "sample.test_job", func(ctx context.Context, job Job[samplePayload]) error {
		received = job
		handled = true
		return nil
	})

	handler, exists := reg.GetHandler("sample.test_job")
	if !exists {
		t.Fatal("expected handler to be registered")
	}

	raw := RawJob{
		UUID:        "test-uuid-1",
		JobType:     "sample.test_job",
		Payload:     []byte(`{"name":"quad","count":42}`),
		Attempt:     1,
		MaxAttempts: 3,
		ScheduledAt: time.Now(),
		CreatedAt:   time.Now(),
	}

	err := handler(context.Background(), raw)
	if err != nil {
		t.Fatalf("handler failed: %v", err)
	}

	if !handled {
		t.Fatal("expected handler to have been invoked")
	}
	if received.Payload.Name != "quad" || received.Payload.Count != 42 {
		t.Errorf("unexpected payload received: %+v", received.Payload)
	}
	if received.UUID != "test-uuid-1" {
		t.Errorf("expected UUID test-uuid-1, got %s", received.UUID)
	}
}

func TestRegistryPeriodicRegistration(t *testing.T) {
	reg := NewRegistry()

	RegisterPeriodic(reg, "cleanup.tokens", 1*time.Hour, EmptyPayload{}, func(ctx context.Context, job Job[EmptyPayload]) error {
		return nil
	})

	schedules := reg.PeriodicSchedules()
	if len(schedules) != 1 {
		t.Fatalf("expected 1 periodic schedule, got %d", len(schedules))
	}
	if schedules[0].JobType != "cleanup.tokens" {
		t.Errorf("expected job type cleanup.tokens, got %s", schedules[0].JobType)
	}
	if schedules[0].Interval != 1*time.Hour {
		t.Errorf("expected interval 1h, got %v", schedules[0].Interval)
	}

	types := reg.TaskTypes()
	if len(types) != 1 || types[0] != "cleanup.tokens" {
		t.Errorf("unexpected task types: %v", types)
	}
}

func TestRegistryInvalidPayloadHandling(t *testing.T) {
	reg := NewRegistry()

	Register(reg, "strict.job", func(ctx context.Context, job Job[samplePayload]) error {
		return nil
	})

	handler, _ := reg.GetHandler("strict.job")
	err := handler(context.Background(), RawJob{
		UUID:    "test-invalid",
		JobType: "strict.job",
		Payload: []byte(`{invalid-json}`),
	})
	if err == nil {
		t.Fatal("expected error on malformed JSON payload")
	}
}

func TestCalculateBackoff(t *testing.T) {
	tests := []struct {
		attempt  int
		expected time.Duration
	}{
		{attempt: 1, expected: 5 * time.Second},
		{attempt: 2, expected: 10 * time.Second},
		{attempt: 3, expected: 20 * time.Second},
		{attempt: 4, expected: 40 * time.Second},
	}

	for _, tc := range tests {
		got := calculateBackoff(tc.attempt)
		if got != tc.expected {
			t.Errorf("calculateBackoff(%d) = %v, expected %v", tc.attempt, got, tc.expected)
		}
	}

	// Verify upper cap at 15 minutes
	high := calculateBackoff(20)
	if high > 15*time.Minute {
		t.Errorf("expected backoff capped at 15m, got %v", high)
	}
}

func TestStatusEnumConversions(t *testing.T) {
	if jobStatusToString(pb.JobStatus_JOB_STATUS_PENDING) != "pending" {
		t.Error("expected pending")
	}
	if jobStatusToString(pb.JobStatus_JOB_STATUS_COMPLETED) != "completed" {
		t.Error("expected completed")
	}
	if jobStatusToString(pb.JobStatus_JOB_STATUS_FAILED) != "failed" {
		t.Error("expected failed")
	}

	if stringToJobStatus("pending") != pb.JobStatus_JOB_STATUS_PENDING {
		t.Error("expected JOB_STATUS_PENDING")
	}
	if stringToJobStatus("running") != pb.JobStatus_JOB_STATUS_RUNNING {
		t.Error("expected JOB_STATUS_RUNNING")
	}
	if stringToJobStatus("failed") != pb.JobStatus_JOB_STATUS_FAILED {
		t.Error("expected JOB_STATUS_FAILED")
	}

	if stringToAttemptStatus("running") != pb.AttemptStatus_ATTEMPT_STATUS_RUNNING {
		t.Error("expected ATTEMPT_STATUS_RUNNING")
	}
	if stringToAttemptStatus("completed") != pb.AttemptStatus_ATTEMPT_STATUS_COMPLETED {
		t.Error("expected ATTEMPT_STATUS_COMPLETED")
	}
	if stringToAttemptStatus("failed") != pb.AttemptStatus_ATTEMPT_STATUS_FAILED {
		t.Error("expected ATTEMPT_STATUS_FAILED")
	}
}

func TestRunSafePanicRecovery(t *testing.T) {
	p := &WorkerPool{}
	panickingHandler := func(ctx context.Context, raw RawJob) error {
		panic("fatal explosion inside worker")
	}

	err := p.runSafe(context.Background(), panickingHandler, RawJob{})
	if err == nil {
		t.Fatal("expected panic to be converted into error")
	}
	if !errors.Is(err, err) && err.Error() != "job panic: fatal explosion inside worker" {
		t.Errorf("unexpected panic message: %v", err)
	}
}

func TestEnqueueConfigOptions(t *testing.T) {
	cfg := EnqueueConfig{}
	now := time.Now()

	WithScheduledAt(now)(&cfg)
	if !cfg.ScheduledAt.Equal(now) {
		t.Errorf("expected scheduled at %v, got %v", now, cfg.ScheduledAt)
	}

	WithUniqueKey("key-123")(&cfg)
	if cfg.UniqueKey == nil || *cfg.UniqueKey != "key-123" {
		t.Errorf("expected unique key key-123, got %v", cfg.UniqueKey)
	}

	WithMaxAttempts(7)(&cfg)
	if cfg.MaxAttempts != 7 {
		t.Errorf("expected max attempts 7, got %d", cfg.MaxAttempts)
	}
}

func TestEmptyPayloadJSON(t *testing.T) {
	b, err := json.Marshal(EmptyPayload{})
	if err != nil {
		t.Fatalf("failed to marshal EmptyPayload: %v", err)
	}
	if string(b) != "{}" {
		t.Errorf("expected {}, got %s", string(b))
	}
}
