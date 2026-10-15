package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	OtpTTL            = 3 * time.Minute
	MagicLinkTTL      = 15 * time.Minute
	EmailCooldown     = 20 * time.Second
	ExhaustedCooldown = 15 * time.Second
	MaxOtpAttempts    = 3
	MaxIPFailures     = 3
	IPLockoutDuration = 60 * time.Minute
)

// GenerateOTP generates a cryptographically random 6-digit numeric OTP and its verification hash
func GenerateOTP() (code string, hash string, err error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", "", err
	}
	code = fmt.Sprintf("%06d", n.Int64())
	hash = HashOTP(code)
	return code, hash, nil
}

// HashOTP hashes the 6-digit code with a dedicated derived key to prevent tampering
func HashOTP(code string) string {
	mac := hmac.New(sha256.New, deriveKey("otp_hash"))
	mac.Write([]byte("otp:" + code))
	return hex.EncodeToString(mac.Sum(nil))
}

// VerifyOTP verifies a candidate code against the stored hash
func VerifyOTP(code, hash string) bool {
	expectedHash := HashOTP(code)
	return hmac.Equal([]byte(expectedHash), []byte(hash))
}

// GenerateMagicLinkToken generates a secure 32-byte single-use magic link token and its hash
func GenerateMagicLinkToken() (token string, hash string, err error) {
	rawBytes := make([]byte, 32)
	if _, err := rand.Read(rawBytes); err != nil {
		return "", "", err
	}
	token = hex.EncodeToString(rawBytes)
	hash = HashToken(token)
	return token, hash, nil
}

// ---------------------------------------------------------
// IP-Bound Rate Limiting & DoS Prevention
// ---------------------------------------------------------

type ipAttemptTracker struct {
	mu       sync.Mutex
	failures map[string][]time.Time
}

var globalIPTracker = &ipAttemptTracker{
	failures: make(map[string][]time.Time),
}

// ResetIPTracker clears all tracked IP failures (used in tests)
func ResetIPTracker() {
	globalIPTracker.mu.Lock()
	defer globalIPTracker.mu.Unlock()
	globalIPTracker.failures = make(map[string][]time.Time)
	ResetIPSignInTracker()
}

// ---------------------------------------------------------
// IP Sign-in Throttling (Anti-Sybil Protection)
// ---------------------------------------------------------

const MaxSignInsPerIP24h = 5

type ipAccountSignInTracker struct {
	mu       sync.Mutex
	accounts map[string]map[string]time.Time
}

var globalIPSignInTracker = &ipAccountSignInTracker{
	accounts: make(map[string]map[string]time.Time),
}

// ResetIPSignInTracker clears all tracked IP sign-ins (used in tests)
func ResetIPSignInTracker() {
	globalIPSignInTracker.mu.Lock()
	defer globalIPSignInTracker.mu.Unlock()
	globalIPSignInTracker.accounts = make(map[string]map[string]time.Time)
}

// CheckAndRecordIPSignIn checks if an IP has exceeded the limit of 5 independent account sign-ins per rolling 24 hours.
func CheckAndRecordIPSignIn(ip, accountKey string) error {
	if ip == "" || accountKey == "" {
		return nil
	}
	normalizedAccount := strings.ToLower(strings.TrimSpace(accountKey))

	globalIPSignInTracker.mu.Lock()
	defer globalIPSignInTracker.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-24 * time.Hour)

	accountMap, ok := globalIPSignInTracker.accounts[ip]
	if !ok {
		accountMap = make(map[string]time.Time)
		globalIPSignInTracker.accounts[ip] = accountMap
	}

	// Evict entries older than 24h
	for acc, t := range accountMap {
		if t.Before(cutoff) {
			delete(accountMap, acc)
		}
	}

	// If account is already active in the window, refresh timestamp and allow
	if _, exists := accountMap[normalizedAccount]; exists {
		accountMap[normalizedAccount] = now
		return nil
	}

	// Limit to max 5 distinct accounts per IP in 24h
	if len(accountMap) >= MaxSignInsPerIP24h {
		return errors.New("maximum number of independent account sign-ins from this IP exceeded (max 5 accounts per 24 hours)")
	}

	accountMap[normalizedAccount] = now
	return nil
}

// CheckIPRateLimit checks if an IP has exceeded the failure limit
func CheckIPRateLimit(ip string) error {
	if ip == "" {
		return nil
	}

	globalIPTracker.mu.Lock()
	defer globalIPTracker.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-IPLockoutDuration)

	timestamps := globalIPTracker.failures[ip]
	valid := make([]time.Time, 0, len(timestamps))
	for _, t := range timestamps {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}
	globalIPTracker.failures[ip] = valid

	if len(valid) >= MaxIPFailures {
		return errors.New("too many failed verification attempts. Please wait 60 minutes before trying again")
	}

	return nil
}

// RecordIPFailure logs a failed attempt for an IP
func RecordIPFailure(ip string) {
	if ip == "" {
		return
	}
	globalIPTracker.mu.Lock()
	defer globalIPTracker.mu.Unlock()

	now := time.Now()
	globalIPTracker.failures[ip] = append(globalIPTracker.failures[ip], now)
}

// RecordIPSuccess records a successful login attempt and resets failure history
func RecordIPSuccess(ip string) {
	if ip == "" {
		return
	}
	globalIPTracker.mu.Lock()
	defer globalIPTracker.mu.Unlock()
	delete(globalIPTracker.failures, ip)
}

// ---------------------------------------------------------
// Email Send Cooldown
// ---------------------------------------------------------

// CheckEmailCooldown verifies whether an email was sent within the 20-second cooldown window,
// or if a previous challenge was exhausted (3 failed attempts) within the 60-second lockout window.
func CheckEmailCooldown(ctx context.Context, db *pgxpool.Pool, email string, customCooldown ...time.Duration) (int32, error) {
	var createdAt time.Time
	var usedAt *time.Time
	var failedAttempts int32
	query := `SELECT created_at, used_at, failed_attempts FROM email_otps WHERE email = $1 ORDER BY created_at DESC LIMIT 1`
	err := db.QueryRow(ctx, query, email).Scan(&createdAt, &usedAt, &failedAttempts)
	if err != nil {
		// No previous OTPs sent for this email
		return 0, nil
	}

	// Enforce 15-second cooldown starting when the 3rd failed challenge happened
	if failedAttempts >= MaxOtpAttempts {
		refTime := createdAt
		if usedAt != nil {
			refTime = *usedAt
		}
		elapsedExhausted := time.Since(refTime)
		if elapsedExhausted < ExhaustedCooldown {
			remaining := int32((ExhaustedCooldown - elapsedExhausted).Seconds())
			if remaining < 1 {
				remaining = 1
			}
			return remaining, nil
		}
		return 0, nil
	}

	cd := EmailCooldown
	if len(customCooldown) > 0 && customCooldown[0] > 0 {
		cd = customCooldown[0]
	}

	// Enforce cooldown from when the email was sent
	elapsed := time.Since(createdAt)
	if elapsed < cd {
		remaining := int32((cd - elapsed).Seconds())
		if remaining < 1 {
			remaining = 1
		}
		return remaining, nil
	}

	return 0, nil
}
