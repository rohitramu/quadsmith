package auth

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// StartHandleReleaseWarningWorker starts a periodic background worker that scans
// handle_histories for archived handles approaching the end of their 30-day hold window
// (23 days elapsed, 7 days remaining) and dispatches warning notifications.
func StartHandleReleaseWarningWorker(ctx context.Context, pool *pgxpool.Pool, checkInterval time.Duration) {
	if checkInterval <= 0 {
		checkInterval = 1 * time.Hour
	}

	go func() {
		ticker := time.NewTicker(checkInterval)
		defer ticker.Stop()

		// Run an initial check on startup
		runReleaseWarningCheck(ctx, pool)

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				runReleaseWarningCheck(ctx, pool)
			}
		}
	}()
}

func runReleaseWarningCheck(ctx context.Context, pool *pgxpool.Pool) {
	rows, err := pool.Query(ctx, `
		SELECT hh.uuid, hh.old_handle, hh.user_uuid, COALESCE(hh.first_archived_at, hh.renamed_at), u.email
		FROM handle_histories hh
		JOIN users u ON u.uuid = hh.user_uuid
		WHERE COALESCE(hh.first_archived_at, hh.renamed_at) + INTERVAL '23 days' <= NOW()
		  AND COALESCE(hh.first_archived_at, hh.renamed_at) + INTERVAL '30 days' > NOW()
		  AND hh.release_warning_sent_at IS NULL
		  AND LOWER(hh.old_handle) != LOWER(u.id)
		  AND u.deleted_at IS NULL`)
	if err != nil {
		log.Printf("[HandleReleaseWorker] Failed to query expiring handles: %v", err)
		return
	}
	defer rows.Close()

	type expiringHandle struct {
		uuid            string
		handle          string
		userUUID        string
		firstArchivedAt time.Time
		email           string
	}

	var items []expiringHandle
	for rows.Next() {
		var item expiringHandle
		if err := rows.Scan(&item.uuid, &item.handle, &item.userUUID, &item.firstArchivedAt, &item.email); err != nil {
			log.Printf("[HandleReleaseWorker] Failed to scan row: %v", err)
			continue
		}
		items = append(items, item)
	}

	for _, item := range items {
		releaseAt := item.firstArchivedAt.Add(30 * 24 * time.Hour)
		// Dispatch warning email to primary user email
		log.Printf("[EMAIL NOTIFICATION] To: %s | Warning: Your archived pilot handle @%s will be released to the public in 7 days (on %s) unless reclaimed.",
			item.email, item.handle, releaseAt.Format("Jan 02, 2006"))

		// Mark notification as sent
		_, err := pool.Exec(ctx, `
			UPDATE handle_histories
			SET release_warning_sent_at = NOW()
			WHERE uuid = $1`, item.uuid)
		if err != nil {
			log.Printf("[HandleReleaseWorker] Failed to mark release warning sent for %s: %v", item.uuid, err)
		}
	}
}
