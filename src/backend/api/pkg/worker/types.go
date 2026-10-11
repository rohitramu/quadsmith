package worker

import (
	"context"
	"time"
)

// Job represents a type-safe unit of work passed to handlers.
type Job[T any] struct {
	UUID        string
	JobType     string
	Payload     T
	Attempt     int // Current attempt number (1-indexed)
	MaxAttempts int
	ScheduledAt time.Time
	CreatedAt   time.Time
}

// Handler is a strongly-typed job execution function.
type Handler[T any] func(ctx context.Context, job Job[T]) error

// EmptyPayload represents jobs that require no input parameters (e.g. periodic cleanup).
type EmptyPayload struct{}

// RawJob represents the untyped database record before JSON deserialization.
type RawJob struct {
	UUID        string
	JobType     string
	Payload     []byte
	Attempt     int
	MaxAttempts int
	ScheduledAt time.Time
	CreatedAt   time.Time
}

// EnqueueOption configures how a job is enqueued.
type EnqueueOption func(*EnqueueConfig)

// EnqueueConfig holds parameters for Enqueue.
type EnqueueConfig struct {
	ScheduledAt time.Time
	UniqueKey   *string
	MaxAttempts int
}

// WithScheduledAt specifies an absolute time when the job should be executed.
func WithScheduledAt(t time.Time) EnqueueOption {
	return func(c *EnqueueConfig) {
		c.ScheduledAt = t
	}
}

// WithDelay specifies a relative delay from now before the job runs.
func WithDelay(d time.Duration) EnqueueOption {
	return func(c *EnqueueConfig) {
		c.ScheduledAt = time.Now().Add(d)
	}
}

// WithUniqueKey sets an idempotency key preventing duplicate pending/running jobs of this type.
func WithUniqueKey(key string) EnqueueOption {
	return func(c *EnqueueConfig) {
		if key != "" {
			c.UniqueKey = &key
		}
	}
}

// WithMaxAttempts overrides the maximum retry attempts for the job.
func WithMaxAttempts(max int) EnqueueOption {
	return func(c *EnqueueConfig) {
		if max > 0 {
			c.MaxAttempts = max
		}
	}
}
