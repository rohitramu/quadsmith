package authcontext

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"
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

// NewUUIDv7 generates a standard RFC 9562 UUID v7
func NewUUIDv7() string {
	var randBytes [16]byte
	_, _ = rand.Read(randBytes[:])
	ms := time.Now().UnixMilli()
	randBytes[0] = byte(ms >> 40)
	randBytes[1] = byte(ms >> 32)
	randBytes[2] = byte(ms >> 24)
	randBytes[3] = byte(ms >> 16)
	randBytes[4] = byte(ms >> 8)
	randBytes[5] = byte(ms)
	randBytes[6] = (randBytes[6] & 0x0f) | 0x70 // UUID v7
	randBytes[8] = (randBytes[8] & 0x3f) | 0x80 // RFC 4122 variant
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", randBytes[0:4], randBytes[4:6], randBytes[6:8], randBytes[8:10], randBytes[10:16])
}
