package api

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"quadsmith/api/pkg/auth"
)

type authRoundTripper struct {
	base  http.RoundTripper
	token string
}

func (rt *authRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Header.Get("Authorization") == "" && rt.token != "" {
		req.Header.Set("Authorization", "Bearer "+rt.token)
	}
	base := rt.base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(req)
}

var testAccessToken string

func TestMain(m *testing.M) {
	if secret := os.Getenv("QS_AUTH_SECRET"); secret != "" {
		auth.SetMasterSecret(secret)
	}
	token, err := auth.SignAccessToken(
		"018f0000-0000-7000-8000-000000000001",
		"test_pilot",
		"pilot@example.com",
		"user",
		"test-session-uuid",
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to generate test access token: %v\n", err)
		os.Exit(1)
	}

	testAccessToken = token
	os.Setenv("QS_ACCESS_TOKEN", token)
	http.DefaultClient.Transport = &authRoundTripper{
		base:  http.DefaultTransport,
		token: token,
	}

	// Seed test user into DB if available
	ctx := context.Background()
	dbUrl := os.Getenv("QS_DATABASE_URL")
	if dbUrl == "" {
		dbUrl = os.Getenv("DATABASE_URL")
	}
	if dbUrl == "" {
		dbUrl = "postgres://postgres:postgres@127.0.0.1:5432/quadsmith"
	}
	if pool, err := pgxpool.New(ctx, dbUrl); err == nil {
		if err := pool.Ping(ctx); err == nil {
			_, _ = pool.Exec(ctx, `
				INSERT INTO users (uuid, id, email, role, created_at, updated_at)
				VALUES ($1, $2, $3, 'user', NOW(), NOW())
				ON CONFLICT (uuid) DO NOTHING`,
				"018f0000-0000-7000-8000-000000000001", "test_pilot", "pilot@example.com")
			_, _ = pool.Exec(ctx, `
				INSERT INTO user_profiles (uuid, user_uuid, avatar_url, bio, updated_at)
				VALUES ($1, $2, '', 'E2E Test Account', NOW())
				ON CONFLICT (user_uuid) DO NOTHING`,
				"018f0000-0000-7000-8000-000000000002", "018f0000-0000-7000-8000-000000000001")
			_, _ = pool.Exec(ctx, `
				INSERT INTO user_emails (uuid, user_uuid, email, is_primary, created_at)
				VALUES ($1, $2, $3, true, NOW())
				ON CONFLICT (email) DO NOTHING`,
				"018f0000-0000-7000-8000-000000000003", "018f0000-0000-7000-8000-000000000001", "pilot@example.com")
		}
		pool.Close()
	}

	os.Exit(m.Run())
}
