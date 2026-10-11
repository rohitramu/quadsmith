package worker

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"quadsmith/api/internal/authcontext"
)

// PoolOption configures the WorkerPool.
type PoolOption func(*WorkerPool)

// WithConcurrency sets the number of concurrent worker goroutines.
func WithConcurrency(c int) PoolOption {
	return func(p *WorkerPool) {
		if c > 0 {
			p.concurrency = c
		}
	}
}

// WithPollInterval sets the polling sleep duration when no jobs are pending.
func WithPollInterval(d time.Duration) PoolOption {
	return func(p *WorkerPool) {
		if d > 0 {
			p.pollInterval = d
		}
	}
}

// WithStalledTimeout sets the threshold past which running jobs are treated as crashed/stalled.
func WithStalledTimeout(d time.Duration) PoolOption {
	return func(p *WorkerPool) {
		if d > 0 {
			p.stalledTimeout = d
		}
	}
}

// WithWorkerID sets a custom identifier for this worker instance.
func WithWorkerID(id string) PoolOption {
	return func(p *WorkerPool) {
		if id != "" {
			p.workerID = id
		}
	}
}

// WorkerPool manages distributed queue polling, concurrent execution, and crash recovery.
type WorkerPool struct {
	db             *pgxpool.Pool
	registry       *Registry
	concurrency    int
	pollInterval   time.Duration
	stalledTimeout time.Duration
	workerID       string

	runningCount int64
	stopChan     chan struct{}
	wg           sync.WaitGroup
}

// NewPool constructs a new WorkerPool.
func NewPool(db *pgxpool.Pool, registry *Registry, opts ...PoolOption) *WorkerPool {
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "worker"
	}
	workerID := fmt.Sprintf("%s:%d:%s", hostname, os.Getpid(), authcontext.NewUUIDv7()[:8])

	p := &WorkerPool{
		db:             db,
		registry:       registry,
		concurrency:    5,
		pollInterval:   500 * time.Millisecond,
		stalledTimeout: 10 * time.Minute,
		workerID:       workerID,
		stopChan:       make(chan struct{}),
	}

	for _, opt := range opts {
		opt(p)
	}

	return p
}

// WorkerID returns this pool's unique instance identifier.
func (p *WorkerPool) WorkerID() string {
	return p.workerID
}

// RunningCount returns the current count of active in-flight jobs.
func (p *WorkerPool) RunningCount() int32 {
	return int32(atomic.LoadInt64(&p.runningCount))
}

// Start launches worker execution routines and stalled-job recovery.
func (p *WorkerPool) Start(ctx context.Context) {
	for i := 0; i < p.concurrency; i++ {
		p.wg.Add(1)
		go p.workerLoop(ctx)
	}

	p.wg.Add(1)
	go p.stalledSweeperLoop(ctx)
}

// Stop signals all workers to stop polling and waits for in-flight jobs to complete.
func (p *WorkerPool) Stop() {
	select {
	case <-p.stopChan:
		// Already stopped
		return
	default:
		close(p.stopChan)
	}
	p.wg.Wait()
}

func (p *WorkerPool) workerLoop(ctx context.Context) {
	defer p.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case <-p.stopChan:
			return
		default:
		}

		claimed, err := p.claimAndExecute(ctx)
		if err != nil {
			if !errors.Is(err, context.Canceled) {
				log.Printf("[WorkerPool %s] Error claiming/executing job: %v", p.workerID, err)
			}
		}

		if !claimed {
			select {
			case <-ctx.Done():
				return
			case <-p.stopChan:
				return
			case <-time.After(p.pollInterval):
			}
		}
	}
}

func (p *WorkerPool) claimAndExecute(ctx context.Context) (bool, error) {
	// Step 1: Claim the next available pending job atomically with FOR UPDATE SKIP LOCKED
	claimQuery := `
		WITH next_job AS (
			SELECT uuid
			FROM background_jobs
			WHERE status = 'pending' AND scheduled_at <= NOW()
			ORDER BY scheduled_at ASC
			FOR UPDATE SKIP LOCKED
			LIMIT 1
		)
		UPDATE background_jobs
		SET status = 'running', updated_at = NOW()
		WHERE uuid IN (SELECT uuid FROM next_job)
		RETURNING uuid, job_type, payload, max_attempts, scheduled_at, created_at;`

	var raw RawJob
	var payloadStr string
	err := p.db.QueryRow(ctx, claimQuery).Scan(
		&raw.UUID,
		&raw.JobType,
		&payloadStr,
		&raw.MaxAttempts,
		&raw.ScheduledAt,
		&raw.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	raw.Payload = []byte(payloadStr)

	// Step 2: Record new attempt entry in background_job_attempts
	attemptUUID := authcontext.NewUUIDv7()
	attemptInsertQuery := `
		INSERT INTO background_job_attempts (
			uuid, job_uuid, attempt_number, worker_id, started_at, status
		)
		VALUES (
			$1, 
			$2, 
			(SELECT COALESCE(MAX(attempt_number), 0) + 1 FROM background_job_attempts WHERE job_uuid = $2),
			$3, 
			NOW(), 
			'running'
		)
		RETURNING attempt_number;`

	err = p.db.QueryRow(ctx, attemptInsertQuery, attemptUUID, raw.UUID, p.workerID).Scan(&raw.Attempt)
	if err != nil {
		// Failed to log attempt; reset job to pending so it can be retried
		_, _ = p.db.Exec(ctx, `UPDATE background_jobs SET status = 'pending', updated_at = NOW() WHERE uuid = $1`, raw.UUID)
		return true, fmt.Errorf("failed to record job attempt for %s: %w", raw.UUID, err)
	}

	atomic.AddInt64(&p.runningCount, 1)
	defer atomic.AddInt64(&p.runningCount, -1)

	// Step 3: Execute handler
	handler, exists := p.registry.GetHandler(raw.JobType)
	var execErr error
	if !exists {
		execErr = fmt.Errorf("no handler registered for job type %q", raw.JobType)
	} else {
		execErr = p.runSafe(ctx, handler, raw)
	}

	// Step 4: Resolve attempt and job status
	if execErr == nil {
		// Succeeded
		_, _ = p.db.Exec(ctx, `
			UPDATE background_job_attempts 
			SET status = 'completed', completed_at = NOW() 
			WHERE uuid = $1`, attemptUUID)

		_, _ = p.db.Exec(ctx, `
			UPDATE background_jobs 
			SET status = 'completed', updated_at = NOW() 
			WHERE uuid = $1`, raw.UUID)
	} else {
		// Failed
		errMsg := execErr.Error()
		_, _ = p.db.Exec(ctx, `
			UPDATE background_job_attempts 
			SET status = 'failed', completed_at = NOW(), error_message = $1 
			WHERE uuid = $2`, errMsg, attemptUUID)

		if raw.Attempt < raw.MaxAttempts {
			backoff := calculateBackoff(raw.Attempt)
			_, _ = p.db.Exec(ctx, `
				UPDATE background_jobs 
				SET status = 'pending', scheduled_at = NOW() + $1, updated_at = NOW() 
				WHERE uuid = $2`, backoff, raw.UUID)
		} else {
			_, _ = p.db.Exec(ctx, `
				UPDATE background_jobs 
				SET status = 'failed', updated_at = NOW() 
				WHERE uuid = $1`, raw.UUID)
		}
	}

	return true, nil
}

func (p *WorkerPool) runSafe(ctx context.Context, handler rawHandler, raw RawJob) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("job panic: %v", r)
		}
	}()
	return handler(ctx, raw)
}

func calculateBackoff(attempt int) time.Duration {
	if attempt <= 0 {
		attempt = 1
	}
	// Exponential backoff: 2^(attempt-1) * 5s, capped at 15 minutes
	secs := math.Pow(2, float64(attempt-1)) * 5.0
	if secs > 900.0 {
		secs = 900.0
	}
	return time.Duration(secs) * time.Second
}

func (p *WorkerPool) stalledSweeperLoop(ctx context.Context) {
	defer p.wg.Done()
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-p.stopChan:
			return
		case <-ticker.C:
			p.reapStalledJobs(ctx)
		}
	}
}

func (p *WorkerPool) reapStalledJobs(ctx context.Context) {
	// Find jobs marked 'running' that haven't been updated in stalledTimeout
	query := `
		SELECT j.uuid, j.max_attempts, COALESCE(MAX(a.attempt_number), 0)
		FROM background_jobs j
		LEFT JOIN background_job_attempts a ON a.job_uuid = j.uuid
		WHERE j.status = 'running' 
		  AND j.updated_at < NOW() - $1::interval
		GROUP BY j.uuid, j.max_attempts;`

	timeoutStr := fmt.Sprintf("%d seconds", int(p.stalledTimeout.Seconds()))
	rows, err := p.db.Query(ctx, query, timeoutStr)
	if err != nil {
		return
	}
	defer rows.Close()

	type stalledJob struct {
		uuid        string
		maxAttempts int
		attempts    int
	}

	var stalled []stalledJob
	for rows.Next() {
		var s stalledJob
		if err := rows.Scan(&s.uuid, &s.maxAttempts, &s.attempts); err == nil {
			stalled = append(stalled, s)
		}
	}

	for _, s := range stalled {
		// Mark active running attempt as failed due to timeout
		_, _ = p.db.Exec(ctx, `
			UPDATE background_job_attempts
			SET status = 'failed', completed_at = NOW(), error_message = 'worker timeout or process crash'
			WHERE job_uuid = $1 AND status = 'running'`, s.uuid)

		if s.attempts < s.maxAttempts {
			// Requeue
			_, _ = p.db.Exec(ctx, `
				UPDATE background_jobs
				SET status = 'pending', scheduled_at = NOW() + INTERVAL '30 seconds', updated_at = NOW()
				WHERE uuid = $1`, s.uuid)
		} else {
			// Exhausted attempts
			_, _ = p.db.Exec(ctx, `
				UPDATE background_jobs
				SET status = 'failed', updated_at = NOW()
				WHERE uuid = $1`, s.uuid)
		}
	}
}
