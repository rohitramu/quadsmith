package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"
)

type rawHandler func(ctx context.Context, raw RawJob) error

// PeriodicSchedule holds metadata for recurring tasks.
type PeriodicSchedule struct {
	JobType  string
	Interval time.Duration
	Payload  []byte
}

// Registry maintains type-erased handlers and recurring task schedules.
type Registry struct {
	mu        sync.RWMutex
	handlers  map[string]rawHandler
	schedules []PeriodicSchedule
}

// NewRegistry initializes an empty Registry.
func NewRegistry() *Registry {
	return &Registry{
		handlers: make(map[string]rawHandler),
	}
}

// Register registers a type-safe job handler into the registry.
func Register[T any](r *Registry, jobType string, handler Handler[T]) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.handlers[jobType] = func(ctx context.Context, raw RawJob) error {
		var payload T
		if len(raw.Payload) > 0 && string(raw.Payload) != "{}" && string(raw.Payload) != "null" {
			if err := json.Unmarshal(raw.Payload, &payload); err != nil {
				return fmt.Errorf("failed to unmarshal payload for job %s: %w", jobType, err)
			}
		}
		return handler(ctx, Job[T]{
			UUID:        raw.UUID,
			JobType:     raw.JobType,
			Payload:     payload,
			Attempt:     raw.Attempt,
			MaxAttempts: raw.MaxAttempts,
			ScheduledAt: raw.ScheduledAt,
			CreatedAt:   raw.CreatedAt,
		})
	}
}

// RegisterPeriodic registers a type-safe recurring job with an execution interval.
func RegisterPeriodic[T any](r *Registry, jobType string, interval time.Duration, defaultPayload T, handler Handler[T]) {
	Register(r, jobType, handler)

	var payloadBytes []byte
	var err error
	switch any(defaultPayload).(type) {
	case EmptyPayload, struct{}:
		payloadBytes = []byte("{}")
	default:
		payloadBytes, err = json.Marshal(defaultPayload)
		if err != nil {
			payloadBytes = []byte("{}")
		}
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.schedules = append(r.schedules, PeriodicSchedule{
		JobType:  jobType,
		Interval: interval,
		Payload:  payloadBytes,
	})
}

// GetHandler retrieves the execution handler for a given job type.
func (r *Registry) GetHandler(jobType string) (rawHandler, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	h, ok := r.handlers[jobType]
	return h, ok
}

// PeriodicSchedules returns all registered periodic schedules.
func (r *Registry) PeriodicSchedules() []PeriodicSchedule {
	r.mu.RLock()
	defer r.mu.RUnlock()
	cp := make([]PeriodicSchedule, len(r.schedules))
	copy(cp, r.schedules)
	return cp
}

// TaskTypes returns a sorted list of registered task names.
func (r *Registry) TaskTypes() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	types := make([]string, 0, len(r.handlers))
	for t := range r.handlers {
		types = append(types, t)
	}
	sort.Strings(types)
	return types
}
