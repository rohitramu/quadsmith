package worker

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Scheduler periodically enqueues registered recurring jobs with interval-bucket unique keys.
type Scheduler struct {
	db          *pgxpool.Pool
	registry    *Registry
	checkPeriod time.Duration
	stopChan    chan struct{}
	wg          sync.WaitGroup
}

// NewScheduler creates a new periodic task Scheduler.
func NewScheduler(db *pgxpool.Pool, registry *Registry) *Scheduler {
	return &Scheduler{
		db:          db,
		registry:    registry,
		checkPeriod: 15 * time.Second,
		stopChan:    make(chan struct{}),
	}
}

// Start launches the periodic evaluation loop.
func (s *Scheduler) Start(ctx context.Context) {
	s.wg.Add(1)
	go s.run(ctx)
}

// Stop cleanly terminates the scheduler loop.
func (s *Scheduler) Stop() {
	select {
	case <-s.stopChan:
		return
	default:
		close(s.stopChan)
	}
	s.wg.Wait()
}

func (s *Scheduler) run(ctx context.Context) {
	defer s.wg.Done()

	// Initial evaluation on startup
	s.evaluateSchedules(ctx)

	ticker := time.NewTicker(s.checkPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.evaluateSchedules(ctx)
		}
	}
}

func (s *Scheduler) evaluateSchedules(ctx context.Context) {
	schedules := s.registry.PeriodicSchedules()
	now := time.Now()

	for _, sched := range schedules {
		if sched.Interval <= 0 {
			continue
		}

		// Derive bucket based on schedule interval
		bucket := now.Truncate(sched.Interval).Unix()
		uniqueKey := fmt.Sprintf("periodic:%s:%d", sched.JobType, bucket)

		// Enqueue with idempotency key
		_, err := Enqueue(ctx, s.db, sched.JobType, sched.Payload, WithUniqueKey(uniqueKey))
		if err != nil {
			log.Printf("[Scheduler] Error scheduling periodic job %s: %v", sched.JobType, err)
		}
	}
}
