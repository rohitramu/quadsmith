package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"quadsmith/api/internal/authcontext"
)

// DBTX abstracts database operations so Enqueue works with both *pgxpool.Pool and pgx.Tx.
type DBTX interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// Enqueue inserts a job into the background_jobs queue within the provided DB connection or transaction.
func Enqueue[T any](ctx context.Context, db DBTX, jobType string, payload T, opts ...EnqueueOption) (string, error) {
	cfg := EnqueueConfig{
		ScheduledAt: time.Now(),
		MaxAttempts: 3,
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	var payloadBytes []byte
	var err error
	switch v := any(payload).(type) {
	case EmptyPayload, struct{}:
		payloadBytes = []byte("{}")
	case json.RawMessage:
		if len(v) == 0 {
			payloadBytes = []byte("{}")
		} else {
			payloadBytes = v
		}
	case []byte:
		if len(v) == 0 {
			payloadBytes = []byte("{}")
		} else {
			payloadBytes = v
		}
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			payloadBytes = []byte("{}")
		} else {
			payloadBytes = []byte(trimmed)
		}
	default:
		payloadBytes, err = json.Marshal(payload)
		if err != nil {
			return "", fmt.Errorf("failed to marshal job payload: %w", err)
		}
	}

	jobUUID := authcontext.NewUUIDv7()

	if cfg.UniqueKey != nil && *cfg.UniqueKey != "" {
		// Idempotent insertion: only insert if no pending/running job exists with this (job_type, unique_key)
		query := `
			WITH ins AS (
				INSERT INTO background_jobs (
					uuid, job_type, payload, status, unique_key, max_attempts, scheduled_at, created_at, updated_at
				)
				SELECT $1, $2, $3::jsonb, 'pending', $4, $5, $6, NOW(), NOW()
				WHERE NOT EXISTS (
					SELECT 1 FROM background_jobs
					WHERE job_type = $2 AND unique_key = $4 AND status IN ('pending', 'running')
				)
				RETURNING uuid
			)
			SELECT uuid FROM ins
			UNION ALL
			SELECT uuid FROM background_jobs
			WHERE job_type = $2 AND unique_key = $4 AND status IN ('pending', 'running')
			LIMIT 1;`

		var returnedUUID string
		err = db.QueryRow(ctx, query, jobUUID, jobType, string(payloadBytes), *cfg.UniqueKey, cfg.MaxAttempts, cfg.ScheduledAt).Scan(&returnedUUID)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return "", fmt.Errorf("failed to enqueue unique job %s: %w", jobType, err)
		}
		if returnedUUID != "" {
			return returnedUUID, nil
		}
		return jobUUID, nil
	}

	// Standard non-unique enqueue
	query := `
		INSERT INTO background_jobs (
			uuid, job_type, payload, status, unique_key, max_attempts, scheduled_at, created_at, updated_at
		)
		VALUES ($1, $2, $3::jsonb, 'pending', NULL, $4, $5, NOW(), NOW())
		RETURNING uuid;`

	var returnedUUID string
	err = db.QueryRow(ctx, query, jobUUID, jobType, string(payloadBytes), cfg.MaxAttempts, cfg.ScheduledAt).Scan(&returnedUUID)
	if err != nil {
		return "", fmt.Errorf("failed to enqueue job %s: %w", jobType, err)
	}

	return returnedUUID, nil
}
