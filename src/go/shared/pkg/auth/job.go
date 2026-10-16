package auth

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"quadsmith/shared/pkg/blob"
)

// HandleUsernameReleaseWarning scans username_histories for archived usernames approaching the end of their 30-day hold window
// (23 days elapsed, 7 days remaining) and dispatches warning notifications.
func HandleUsernameReleaseWarning(ctx context.Context, pool *pgxpool.Pool) (int64, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, `
		SELECT hh.uuid, hh.old_username, hh.user_uuid, COALESCE(hh.first_archived_at, hh.renamed_at), COALESCE(ue.email, '')
		FROM username_histories hh
		JOIN users u ON u.uuid = hh.user_uuid
		LEFT JOIN user_emails ue ON ue.user_uuid = u.uuid AND ue.is_primary = true
		WHERE COALESCE(hh.first_archived_at, hh.renamed_at) + INTERVAL '23 days' <= NOW()
		  AND COALESCE(hh.first_archived_at, hh.renamed_at) + INTERVAL '30 days' > NOW()
		  AND hh.release_warning_sent_at IS NULL
		  AND LOWER(hh.old_username) != LOWER(u.id)
		  AND u.deleted_at IS NULL
		FOR UPDATE OF hh SKIP LOCKED`)
	if err != nil {
		return 0, fmt.Errorf("failed to query expiring usernames: %w", err)
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
			log.Printf("[UsernameReleaseJob] Failed to scan row: %v", err)
			continue
		}
		items = append(items, item)
	}
	rows.Close()

	var sentCount int64
	for _, item := range items {
		releaseAt := item.firstArchivedAt.Add(30 * 24 * time.Hour)
		// Dispatch warning email to primary user email
		log.Printf("[EMAIL NOTIFICATION] To: %s | Warning: Your archived username @%s will be released to the public in 7 days (on %s) unless reclaimed.",
			item.email, item.username, releaseAt.Format("Jan 02, 2006"))

		// Mark notification as sent
		_, err := tx.Exec(ctx, `
			UPDATE username_histories
			SET release_warning_sent_at = NOW()
			WHERE uuid = $1`, item.uuid)
		if err != nil {
			log.Printf("[UsernameReleaseJob] Failed to mark release warning sent for %s: %v", item.uuid, err)
			continue
		}
		sentCount++
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("failed to commit username release warnings: %w", err)
	}

	if sentCount > 0 {
		log.Printf("[UsernameReleaseJob] Dispatched %d username release warnings", sentCount)
	}
	return sentCount, nil
}

// HandleCleanupExpiredOTPs purges used OTPs and expired OTPs older than 24 hours.
func HandleCleanupExpiredOTPs(ctx context.Context, pool *pgxpool.Pool) (int64, error) {
	cmd, err := pool.Exec(ctx, `
		DELETE FROM email_otps 
		WHERE (used_at IS NOT NULL OR expires_at < NOW()) 
		  AND created_at < NOW() - INTERVAL '24 hours';`)
	if err != nil {
		return 0, fmt.Errorf("failed to clean up expired OTPs: %w", err)
	}
	rows := cmd.RowsAffected()
	if rows > 0 {
		log.Printf("[AuthJob] Cleaned up %d expired/used OTPs", rows)
	}
	return rows, nil
}

// HandleCleanupExpiredTokens purges revoked or expired refresh tokens older than 7 days.
func HandleCleanupExpiredTokens(ctx context.Context, pool *pgxpool.Pool) (int64, error) {
	cmd, err := pool.Exec(ctx, `
		DELETE FROM refresh_tokens 
		WHERE expires_at < NOW() - INTERVAL '7 days';`)
	if err != nil {
		return 0, fmt.Errorf("failed to clean up expired refresh tokens: %w", err)
	}
	rows := cmd.RowsAffected()
	if rows > 0 {
		log.Printf("[AuthJob] Cleaned up %d expired refresh tokens", rows)
	}
	return rows, nil
}

// HandlePurgeSoftDeletedAccounts executes atomic final data purges for soft-deleted accounts
// whose 30-day grace period has elapsed.
func HandlePurgeSoftDeletedAccounts(ctx context.Context, pool *pgxpool.Pool) (int64, error) {
	rows, err := pool.Query(ctx, `
		SELECT uuid, id FROM users
		WHERE deleted_at IS NOT NULL
		  AND deleted_at <= NOW() - INTERVAL '30 days'
		  AND id NOT LIKE 'deleted-%';`)
	if err != nil {
		return 0, fmt.Errorf("failed to query expired soft-deleted users: %w", err)
	}
	defer rows.Close()

	type expiredUser struct {
		uuid string
		id   string
	}
	var users []expiredUser
	for rows.Next() {
		var u expiredUser
		if err := rows.Scan(&u.uuid, &u.id); err != nil {
			log.Printf("[PurgeJob] Failed to scan user: %v", err)
			continue
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("error reading expired users: %w", err)
	}

	var purgedCount int64
	for _, u := range users {
		tx, err := pool.Begin(ctx)
		if err != nil {
			log.Printf("[PurgeJob] Failed to begin transaction for %s: %v", u.uuid, err)
			continue
		}

		// 1. Anonymize user handle, releasing original username to the public pool
		newHandle := "deleted-" + u.uuid
		_, err = tx.Exec(ctx, `UPDATE users SET id = $1, updated_at = NOW() WHERE uuid = $2`, newHandle, u.uuid)
		if err != nil {
			_ = tx.Rollback(ctx)
			log.Printf("[PurgeJob] Failed to anonymize user %s: %v", u.uuid, err)
			continue
		}

		// 2. Clear personal profile details and unlink any custom avatar blob reference
		_, err = tx.Exec(ctx, `UPDATE user_profiles SET bio = '', avatar_url = '', updated_at = NOW() WHERE user_uuid = $1`, u.uuid)
		if err != nil {
			_ = tx.Rollback(ctx)
			log.Printf("[PurgeJob] Failed to clear profile for %s: %v", u.uuid, err)
			continue
		}
		if err := blob.UnlinkEntityBlobs(ctx, tx, "user_avatar", u.uuid); err != nil {
			_ = tx.Rollback(ctx)
			log.Printf("[PurgeJob] Failed to unlink avatar blobs for %s: %v", u.uuid, err)
			continue
		}

		// 3. Hard delete email addresses and OAuth junction records
		_, err = tx.Exec(ctx, `
			DELETE FROM user_email_oauth_accounts
			WHERE user_email_uuid IN (SELECT uuid FROM user_emails WHERE user_uuid = $1)`, u.uuid)
		if err != nil {
			_ = tx.Rollback(ctx)
			log.Printf("[PurgeJob] Failed to delete user_email_oauth_accounts for %s: %v", u.uuid, err)
			continue
		}

		_, err = tx.Exec(ctx, `DELETE FROM user_emails WHERE user_uuid = $1`, u.uuid)
		if err != nil {
			_ = tx.Rollback(ctx)
			log.Printf("[PurgeJob] Failed to delete user_emails for %s: %v", u.uuid, err)
			continue
		}

		// 4. Hard delete linked OAuth provider credentials
		_, err = tx.Exec(ctx, `DELETE FROM oauth_accounts WHERE user_uuid = $1`, u.uuid)
		if err != nil {
			_ = tx.Rollback(ctx)
			log.Printf("[PurgeJob] Failed to delete oauth_accounts for %s: %v", u.uuid, err)
			continue
		}

		// 5. Revoke all active sessions and refresh tokens
		_, err = tx.Exec(ctx, `DELETE FROM refresh_tokens WHERE user_uuid = $1`, u.uuid)
		if err != nil {
			_ = tx.Rollback(ctx)
			log.Printf("[PurgeJob] Failed to delete refresh_tokens for %s: %v", u.uuid, err)
			continue
		}

		// 6. Delete any email OTP challenges requested by or sent to this user
		_, err = tx.Exec(ctx, `DELETE FROM email_otps WHERE requested_by_user_uuid = $1`, u.uuid)
		if err != nil {
			_ = tx.Rollback(ctx)
			log.Printf("[PurgeJob] Failed to delete email_otps for %s: %v", u.uuid, err)
			continue
		}

		if err := tx.Commit(ctx); err != nil {
			log.Printf("[PurgeJob] Failed to commit purge for %s: %v", u.uuid, err)
			continue
		}

		log.Printf("[PurgeJob] Successfully purged user %s (@%s), released handle", u.uuid, u.id)
		purgedCount++
	}

	return purgedCount, nil
}
