package auth

import (
	"context"
	"net/http"
	"testing"

	"connectrpc.com/connect"

	pb "quadsmith/shared/gen/quadsmith/api"
)

func TestSessionRPCs_Unauthenticated(t *testing.T) {
	svc := &AuthService{}
	ctx := context.Background()

	// 1. ListActiveSessions
	_, err := svc.ListActiveSessions(ctx, connect.NewRequest(&pb.ListActiveSessionsRequest{}))
	if err == nil {
		t.Fatal("expected unauthenticated error for ListActiveSessions, got nil")
	}
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("expected CodeUnauthenticated, got %v", connect.CodeOf(err))
	}

	// 2. RevokeSession
	_, err = svc.RevokeSession(ctx, connect.NewRequest(&pb.RevokeSessionRequest{
		SessionUuid: "some-uuid",
	}))
	if err == nil {
		t.Fatal("expected unauthenticated error for RevokeSession, got nil")
	}
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("expected CodeUnauthenticated, got %v", connect.CodeOf(err))
	}

	// 3. RevokeOtherSessions
	_, err = svc.RevokeOtherSessions(ctx, connect.NewRequest(&pb.RevokeOtherSessionsRequest{}))
	if err == nil {
		t.Fatal("expected unauthenticated error for RevokeOtherSessions, got nil")
	}
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("expected CodeUnauthenticated, got %v", connect.CodeOf(err))
	}
}

func TestRevokeSession_InvalidArgument(t *testing.T) {
	svc := &AuthService{}
	userCtx := &UserContext{
		UserUUID:    "test-user-uuid",
		UserID:      "testuser",
		SessionUUID: "curr-session-uuid",
	}
	ctx := WithUserContext(context.Background(), userCtx)

	_, err := svc.RevokeSession(ctx, connect.NewRequest(&pb.RevokeSessionRequest{
		SessionUuid: "   ",
	}))
	if err == nil {
		t.Fatal("expected error for empty session_uuid, got nil")
	}
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("expected CodeInvalidArgument, got %v", connect.CodeOf(err))
	}
}

func TestDeviceHintParsingComprehensive(t *testing.T) {
	tests := []struct {
		ua       string
		expected string
	}{
		{
			ua:       "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			expected: "Chrome on macOS",
		},
		{
			ua:       "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1",
			expected: "Safari on iOS",
		},
		{
			ua:       "Mozilla/5.0 (Linux; Android 14; Pixel 8) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.6099.43 Mobile Safari/537.36",
			expected: "Chrome on Android",
		},
		{
			ua:       "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 Edg/120.0.0.0",
			expected: "Edge on Windows",
		},
		{
			ua:       "Mozilla/5.0 (X11; Linux x86_64; rv:109.0) Gecko/20100101 Firefox/119.0",
			expected: "Firefox on Linux",
		},
		{
			ua:       "qs/1.0 (QuadsmithCLI; linux/amd64)",
			expected: "Quadsmith CLI",
		},
		{
			ua:       "Chrome on macOS",
			expected: "Chrome on macOS",
		},
		{
			ua:       "",
			expected: "Web Browser",
		},
	}

	for _, tt := range tests {
		got := parseDeviceHint(tt.ua)
		if got != tt.expected {
			t.Errorf("parseDeviceHint(%q) = %q, expected %q", tt.ua, got, tt.expected)
		}
	}
}

func TestLocationHintExtractionComprehensive(t *testing.T) {
	header := http.Header{}
	header.Set("CF-IPCity", "San Francisco")
	header.Set("CF-IPCountry", "US")

	loc := extractLocationHint(header, "8.8.8.8")
	if loc != "San Francisco, US" {
		t.Errorf("expected 'San Francisco, US', got %q", loc)
	}

	header2 := http.Header{}
	header2.Set("CF-IPCountry", "Germany")
	loc2 := extractLocationHint(header2, "1.2.3.4")
	if loc2 != "Germany" {
		t.Errorf("expected 'Germany', got %q", loc2)
	}

	locPrivate := extractLocationHint(header, "192.168.1.10")
	if locPrivate != "Local Network" {
		t.Errorf("expected 'Local Network', got %q", locPrivate)
	}

	locEmpty := extractLocationHint(nil, "8.8.8.8")
	if locEmpty != "Unknown Location" {
		t.Errorf("expected 'Unknown Location', got %q", locEmpty)
	}
}
