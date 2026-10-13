package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"

	connect "connectrpc.com/connect"
	"github.com/jackc/pgx/v5/pgxpool"
)

var publicProcedures = map[string]bool{
	"/quadsmith.AuthService/SendEmailOtp":             true,
	"/quadsmith.AuthService/VerifyEmailOtp":           true,
	"/quadsmith.AuthService/VerifyMagicLink":          true,
	"/quadsmith.AuthService/GetLoginChallengeDetails": true,
	"/quadsmith.AuthService/ApproveLoginChallenge":    true,
	"/quadsmith.AuthService/DenyLoginChallenge":       true,
	"/quadsmith.AuthService/GetLoginChallengeStatus":  true,
	"/quadsmith.AuthService/CompleteAccountSetup":     true,
	"/quadsmith.AuthService/RefreshToken":             true,
	"/quadsmith.UserService/CheckUsernameAvailable":   true,

	// Public social queries
	"/quadsmith.SocialService/ListReviews":            true,
	"/quadsmith.SocialService/ListReviewComments":     true,
	"/quadsmith.SocialService/ListQuestions":          true,
	"/quadsmith.SocialService/ListAnswers":            true,
	"/quadsmith.SocialService/GetEntitySocialSummary": true,
	"/quadsmith.SocialService/GetCollection":          true,
	"/quadsmith.SocialService/ListUserCollections":    true,
	"/quadsmith.SocialService/ListCollectionItems":    true,
}

// NewAuthInterceptor creates a ConnectRPC unary interceptor for authentication
func NewAuthInterceptor(pool *pgxpool.Pool) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			procedure := req.Spec().Procedure
			header := req.Header()
			httpReq := &http.Request{Header: header}
			userCtx := AuthenticateHTTPRequest(ctx, pool, httpReq)

			// If authenticated, inject user context
			if userCtx != nil {
				ctx = WithUserContext(ctx, userCtx)
			}

			// 3. Enforce authentication on protected endpoints
			if !publicProcedures[procedure] && userCtx == nil {
				return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("authentication required"))
			}

			return next(ctx, req)
		}
	}
}

// AuthenticateHTTPRequest validates an incoming HTTP request via Bearer token or session cookie.
func AuthenticateHTTPRequest(ctx context.Context, pool *pgxpool.Pool, r *http.Request) *UserContext {
	// 1. Try Bearer token in Authorization header
	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
		tokenStr := strings.TrimSpace(authHeader[7:])
		if claims, err := VerifyAccessToken(tokenStr); err == nil && claims != nil {
			return claims
		}
	}

	// 2. If no valid Bearer token, try session cookie (qs_session)
	if c, err := r.Cookie("qs_session"); err == nil && c.Value != "" {
		tokenHash := HashToken(c.Value)
		var sessionUUID, userUUID, username, email, role string
		err := pool.QueryRow(ctx, `
			SELECT r.uuid, u.uuid, u.id, u.email, u.role
			FROM refresh_tokens r
			JOIN users u ON u.uuid = r.user_uuid
			WHERE r.token_hash = $1 AND r.expires_at > NOW() AND u.deleted_at IS NULL
			LIMIT 1`, tokenHash).Scan(&sessionUUID, &userUUID, &username, &email, &role)
		if err == nil {
			return &UserContext{
				UserUUID:    userUUID,
				UserID:      username,
				Email:       email,
				Role:        role,
				SessionUUID: sessionUUID,
			}
		}
	}

	return nil
}
