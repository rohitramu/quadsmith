package auth

import (
	"testing"
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
	}

	// Successful login resets failures
	RecordIPSuccess(testIP)
	if err := CheckIPRateLimit(testIP); err != nil {
		t.Fatalf("rate limit should be cleared after success: %v", err)
	}
}
