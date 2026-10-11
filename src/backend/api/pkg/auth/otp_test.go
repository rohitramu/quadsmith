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
	testIP := "203.0.113.42"

	// Initially clean
	if err := CheckIPRateLimit(testIP); err != nil {
		t.Fatalf("unexpected rate limit error: %v", err)
	}

	// Record 4 failures (limit is 5)
	for i := 0; i < 4; i++ {
		RecordIPFailure(testIP)
	}
	if err := CheckIPRateLimit(testIP); err != nil {
		t.Fatalf("should not be rate limited after 4 failures: %v", err)
	}

	// 5th failure triggers lockout
	RecordIPFailure(testIP)
	if err := CheckIPRateLimit(testIP); err == nil {
		t.Fatalf("expected rate limit lockout after 5 failures")
	} else if !strings.Contains(err.Error(), "60 minutes") {
		t.Fatalf("expected 60 minutes in error message, got: %v", err)
	}

	// Successful login resets failures
	RecordIPSuccess(testIP)
	if err := CheckIPRateLimit(testIP); err != nil {
		t.Fatalf("rate limit should be cleared after success: %v", err)
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
	if ExhaustedCooldown != 60*time.Second {
		t.Errorf("expected ExhaustedCooldown to be 60s, got %v", ExhaustedCooldown)
	}
}
