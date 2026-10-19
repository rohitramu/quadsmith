package auth

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"quadsmith/shared/pkg/authcontext"
)

type UserContext = authcontext.UserContext

const (
	SystemUserUUID   = authcontext.SystemUserUUID
	SystemUserHandle = authcontext.SystemUserHandle

	RoleUser   = authcontext.RoleUser
	RoleAdmin  = authcontext.RoleAdmin
	RoleSystem = authcontext.RoleSystem
)

var WithUserContext = authcontext.WithUserContext
var UserContextFromContext = authcontext.UserContextFromContext
var NewUUIDv7 = authcontext.NewUUIDv7

// SystemUserDB is satisfied by both *pgxpool.Pool and pgx.Tx.
type SystemUserDB interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// EnsureSystemUser guarantees that the @quadsmith system user (role = 'SYSTEM') exists in quadsmith_api.users
// along with its profile and reputation records.
func EnsureSystemUser(ctx context.Context, db SystemUserDB) (string, error) {
	if db == nil {
		return SystemUserUUID, nil
	}
	var userUUID string
	err := db.QueryRow(ctx, `
		INSERT INTO users (uuid, id, role, deleted_at, created_at, updated_at)
		VALUES ($1::uuid, $2, 'SYSTEM', NULL, NOW() - INTERVAL '365 days', NOW())
		ON CONFLICT (id) DO UPDATE SET role = 'SYSTEM', deleted_at = NULL
		RETURNING uuid::text
	`, SystemUserUUID, SystemUserHandle).Scan(&userUUID)
	if err != nil {
		return "", err
	}

	_, _ = db.Exec(ctx, `
		INSERT INTO user_profiles (uuid, user_uuid, avatar_url, bio, updated_at)
		VALUES ($1::uuid, $2::uuid, '', 'Official Quadsmith System Account', NOW())
		ON CONFLICT (user_uuid) DO NOTHING
	`, NewUUIDv7(), userUUID)

	_, _ = db.Exec(ctx, `
		INSERT INTO user_reputations (uuid, user_uuid, score, updated_at)
		VALUES ($1::uuid, $2::uuid, 1000, NOW())
		ON CONFLICT (user_uuid) DO UPDATE SET score = GREATEST(user_reputations.score, 1000)
	`, NewUUIDv7(), userUUID)

	return userUUID, nil
}
