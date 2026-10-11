package auth

import (
	"strings"
	"testing"
	"time"
)

func TestOTPGenerationAndVerification(t *testing.T) {
	code, hash, err := GenerateOTP()
	if err != nil {
		t.Fatalf("GenerateOTP failed: %v", err)
	}
	if len(code) != 6 {
		t.Fatalf("expected 6-digit OTP, got %s", code)
	}

	if !VerifyOTP(code, hash) {
		t.Fatalf("VerifyOTP failed on valid code")
	}

	if VerifyOTP("999999", hash) && code != "999999" {
		t.Fatalf("VerifyOTP passed on invalid code")
	}
}

func TestIPRateLimiting(t *testing.T) {
	ResetIPTracker()
	testIP := "203.0.113.42"

	// Initially clean
	if err := CheckIPRateLimit(testIP); err != nil {
		t.Fatalf("unexpected rate limit error: %v", err)
	}

	// Record 1 failure (limit is 3 failed codes)
	RecordIPFailure(testIP)
	if err := CheckIPRateLimit(testIP); err != nil {
		t.Fatalf("should not be rate limited after 1 failure: %v", err)
	}

	// Record 2nd failure (verifying no premature lockout after 2 failed codes!)
	RecordIPFailure(testIP)
	if err := CheckIPRateLimit(testIP); err != nil {
		t.Fatalf("should not be rate limited after 2 failures: %v", err)
	}

	// 3rd failure triggers lockout
	RecordIPFailure(testIP)
	if err := CheckIPRateLimit(testIP); err == nil {
		t.Fatalf("expected rate limit lockout after 3 failures")
	} else if !strings.Contains(err.Error(), "60 minutes") {
		t.Fatalf("expected 60 minutes in error message, got: %v", err)
	}

	// RecordIPSuccess clears failures for that IP
	RecordIPSuccess(testIP)
	if err := CheckIPRateLimit(testIP); err != nil {
		t.Fatalf("rate limit should be cleared after RecordIPSuccess: %v", err)
	}

	// Reset tracker clears failures
	ResetIPTracker()
	if err := CheckIPRateLimit(testIP); err != nil {
		t.Fatalf("rate limit should be cleared after reset: %v", err)
	}
}

func TestPluralAttempts(t *testing.T) {
	if pluralAttempts(1) != "attempt" {
		t.Errorf("expected 'attempt' for 1, got %s", pluralAttempts(1))
	}
	if pluralAttempts(2) != "attempts" {
		t.Errorf("expected 'attempts' for 2, got %s", pluralAttempts(2))
	}
	if pluralAttempts(0) != "attempts" {
		t.Errorf("expected 'attempts' for 0, got %s", pluralAttempts(0))
	}
}

func TestCooldownDurations(t *testing.T) {
	if EmailCooldown != 20*time.Second {
		t.Errorf("expected EmailCooldown to be 20s, got %v", EmailCooldown)
	}
	if ExhaustedCooldown != 15*time.Second {
		t.Errorf("expected ExhaustedCooldown to be 15s, got %v", ExhaustedCooldown)
	}
}

func TestIPAccountSignInThrottling(t *testing.T) {
	ResetIPSignInTracker()
	testIP := "198.51.100.25"

	// Sign in with 5 different accounts - all should succeed
	for i := 1; i <= 5; i++ {
		email := strings.ToLower(string(rune('a'+i))) + "@example.com"
		if err := CheckAndRecordIPSignIn(testIP, email); err != nil {
			t.Fatalf("account %d should succeed within limit: %v", i, err)
		}
	}

	// Signing in again with one of the already-recorded accounts should succeed (same account)
	if err := CheckAndRecordIPSignIn(testIP, "b@example.com"); err != nil {
		t.Fatalf("re-signing into existing account should succeed: %v", err)
	}

	// 6th different account from the same IP should fail
	if err := CheckAndRecordIPSignIn(testIP, "sybil6@example.com"); err == nil {
		t.Fatalf("expected 6th distinct account from same IP to be blocked by Sybil rate limit")
	} else if !strings.Contains(err.Error(), "max 5 accounts per 24 hours") {
		t.Fatalf("unexpected error message: %v", err)
	}

	// Different IP should still be able to sign in
	if err := CheckAndRecordIPSignIn("198.51.100.99", "sybil6@example.com"); err != nil {
		t.Fatalf("different IP should succeed: %v", err)
	}

	// Reset clears tracker
	ResetIPSignInTracker()
	if err := CheckAndRecordIPSignIn(testIP, "sybil6@example.com"); err != nil {
		t.Fatalf("after reset, sign-in should succeed: %v", err)
	}
}
