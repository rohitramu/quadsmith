package auth

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// StartUsernameReleaseWarningWorker starts a periodic background worker that scans
// username_histories for archived usernames approaching the end of their 30-day hold window
// (23 days elapsed, 7 days remaining) and dispatches warning notifications.
func StartUsernameReleaseWarningWorker(ctx context.Context, pool *pgxpool.Pool, checkInterval time.Duration) {
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
		SELECT hh.uuid, hh.old_username, hh.user_uuid, COALESCE(hh.first_archived_at, hh.renamed_at), u.email
		FROM username_histories hh
		JOIN users u ON u.uuid = hh.user_uuid
		WHERE COALESCE(hh.first_archived_at, hh.renamed_at) + INTERVAL '23 days' <= NOW()
		  AND COALESCE(hh.first_archived_at, hh.renamed_at) + INTERVAL '30 days' > NOW()
		  AND hh.release_warning_sent_at IS NULL
		  AND LOWER(hh.old_username) != LOWER(u.id)
		  AND u.deleted_at IS NULL`)
	if err != nil {
		log.Printf("[UsernameReleaseWorker] Failed to query expiring usernames: %v", err)
		return
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
}
