package auth

import (
	"context"
)

type contextKey string

const userContextKey = contextKey("quadsmith_user_context")

// UserContext carries the authenticated user's identity through context.Context
type UserContext struct {
	UserUUID    string
	UserID      string // Public pilot handle (e.g. "fpv_pilot")
	Email       string
	Role        string
	SessionUUID string
}

// WithUserContext injects UserContext into context.Context
func WithUserContext(ctx context.Context, u *UserContext) context.Context {
	return context.WithValue(ctx, userContextKey, u)
}

// UserContextFromContext extracts UserContext from context.Context
func UserContextFromContext(ctx context.Context) *UserContext {
	if u, ok := ctx.Value(userContextKey).(*UserContext); ok && u != nil {
		return u
	}
	return nil
}
