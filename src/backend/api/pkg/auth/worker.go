package auth

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"quadsmith/api/pkg/worker"
)

// RegisterWorkers registers all auth-related background tasks into the worker registry.
func RegisterWorkers(r *worker.Registry, pool *pgxpool.Pool) {
	// Periodic 1h task: username hold release warnings (7 days prior to release)
	worker.RegisterPeriodic(r, "auth.username_release_warning", 1*time.Hour, worker.EmptyPayload{}, func(ctx context.Context, job worker.Job[worker.EmptyPayload]) error {
		return HandleUsernameReleaseWarning(ctx, pool)
	})

	// Periodic 6h task: clean up used and expired email OTPs older than 24h
	worker.RegisterPeriodic(r, "auth.cleanup_expired_otps", 6*time.Hour, worker.EmptyPayload{}, func(ctx context.Context, job worker.Job[worker.EmptyPayload]) error {
		return HandleCleanupExpiredOTPs(ctx, pool)
	})

	// Periodic 12h task: clean up expired refresh tokens older than 7d
	worker.RegisterPeriodic(r, "auth.cleanup_expired_tokens", 12*time.Hour, worker.EmptyPayload{}, func(ctx context.Context, job worker.Job[worker.EmptyPayload]) error {
		return HandleCleanupExpiredTokens(ctx, pool)
	})
}

// HandleUsernameReleaseWarning scans username_histories for archived usernames approaching the end of their 30-day hold window
// (23 days elapsed, 7 days remaining) and dispatches warning notifications.
func HandleUsernameReleaseWarning(ctx context.Context, pool *pgxpool.Pool) error {
	rows, err := pool.Query(ctx, `
		SELECT hh.uuid, hh.old_username, hh.user_uuid, COALESCE(hh.first_archived_at, hh.renamed_at), u.email
		FROM username_histories hh
		JOIN users u ON u.uuid = hh.user_uuid
		WHERE COALESCE(hh.first_archived_at, hh.renamed_at) + INTERVAL '23 days' <= NOW()
		  AND COALESCE(hh.first_archived_at, hh.renamed_at) + INTERVAL '30 days' > NOW()
		  AND hh.release_warning_sent_at IS NULL
		  AND LOWER(hh.old_username) != LOWER(u.id)
		  AND u.deleted_at IS NULL`)
	if err != nil {
		return fmt.Errorf("failed to query expiring usernames: %w", err)
	}
	defer rows.Close()

	type expiringUsername struct {
		uuid            string
		username        string
		userUUID        string
		firstArchivedAt time.Time
		email           string
	}

	var items []expiringUsername
	for rows.Next() {
		var item expiringUsername
		if err := rows.Scan(&item.uuid, &item.username, &item.userUUID, &item.firstArchivedAt, &item.email); err != nil {
			log.Printf("[UsernameReleaseWorker] Failed to scan row: %v", err)
			continue
		}
		items = append(items, item)
	}

	for _, item := range items {
		releaseAt := item.firstArchivedAt.Add(30 * 24 * time.Hour)
		// Dispatch warning email to primary user email
		log.Printf("[EMAIL NOTIFICATION] To: %s | Warning: Your archived username @%s will be released to the public in 7 days (on %s) unless reclaimed.",
			item.email, item.username, releaseAt.Format("Jan 02, 2006"))

		// Mark notification as sent
		_, err := pool.Exec(ctx, `
			UPDATE username_histories
			SET release_warning_sent_at = NOW()
			WHERE uuid = $1`, item.uuid)
		if err != nil {
			log.Printf("[UsernameReleaseWorker] Failed to mark release warning sent for %s: %v", item.uuid, err)
		}
	}

	return nil
}

// HandleCleanupExpiredOTPs purges used OTPs and expired OTPs older than 24 hours.
func HandleCleanupExpiredOTPs(ctx context.Context, pool *pgxpool.Pool) error {
	cmd, err := pool.Exec(ctx, `
		DELETE FROM email_otps 
		WHERE (used_at IS NOT NULL OR expires_at < NOW()) 
		  AND created_at < NOW() - INTERVAL '24 hours';`)
	if err != nil {
		return fmt.Errorf("failed to clean up expired OTPs: %w", err)
	}
	if rows := cmd.RowsAffected(); rows > 0 {
		log.Printf("[AuthWorker] Cleaned up %d expired/used OTPs", rows)
	}
	return nil
}

// HandleCleanupExpiredTokens purges revoked or expired refresh tokens older than 7 days.
func HandleCleanupExpiredTokens(ctx context.Context, pool *pgxpool.Pool) error {
	cmd, err := pool.Exec(ctx, `
		DELETE FROM refresh_tokens 
		WHERE expires_at < NOW() - INTERVAL '7 days';`)
	if err != nil {
		return fmt.Errorf("failed to clean up expired refresh tokens: %w", err)
	}
	if rows := cmd.RowsAffected(); rows > 0 {
		log.Printf("[AuthWorker] Cleaned up %d expired refresh tokens", rows)
	}
	return nil
}
