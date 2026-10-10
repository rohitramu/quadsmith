package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

const (
	AccessTokenTTL  = 1 * time.Hour
	RefreshTokenTTL = 5 * 24 * time.Hour // 5-day sliding window
)

func getJWTSecret() []byte {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "quadsmith-dev-secret-key-32bytes-long!"
	}
	return []byte(secret)
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

type jwtHeader struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
}

type JWTClaims struct {
	UserUUID    string `json:"user_uuid"`
	UserID      string `json:"user_id"` // Public username (e.g. "fpv_pilot")
	Email       string `json:"email"`
	Role        string `json:"role"`
	SessionUUID string `json:"session_uuid"`
	Exp         int64  `json:"exp"`
	Iat         int64  `json:"iat"`
}

// SignAccessToken signs a new 1-hour access JWT using HMAC-SHA256
func SignAccessToken(userUUID, userID, email, role, sessionUUID string) (string, error) {
	now := time.Now()
	claims := JWTClaims{
		UserUUID:    userUUID,
		UserID:      userID,
		Email:       email,
		Role:        role,
		SessionUUID: sessionUUID,
		Iat:         now.Unix(),
		Exp:         now.Add(AccessTokenTTL).Unix(),
	}

	header := jwtHeader{Alg: "HS256", Typ: "JWT"}
	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)
	claimsB64 := base64.RawURLEncoding.EncodeToString(claimsJSON)

	payload := headerB64 + "." + claimsB64
	sig := hmacSHA256([]byte(payload), getJWTSecret())
	sigB64 := base64.RawURLEncoding.EncodeToString(sig)

	return payload + "." + sigB64, nil
}

// VerifyAccessToken verifies an access JWT and returns its claims
func VerifyAccessToken(tokenStr string) (*UserContext, error) {
	parts := strings.Split(tokenStr, ".")
	if len(parts) != 3 {
		return nil, errors.New("invalid token format")
	}

	headerB64, claimsB64, sigB64 := parts[0], parts[1], parts[2]
	expectedSig := hmacSHA256([]byte(headerB64+"."+claimsB64), getJWTSecret())
	actualSig, err := base64.RawURLEncoding.DecodeString(sigB64)
	if err != nil {
		return nil, errors.New("invalid signature encoding")
	}

	if !hmac.Equal(expectedSig, actualSig) {
		return nil, errors.New("token signature mismatch")
	}

	claimsJSON, err := base64.RawURLEncoding.DecodeString(claimsB64)
	if err != nil {
		return nil, errors.New("invalid claims encoding")
	}

	var claims JWTClaims
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		return nil, errors.New("invalid claims json")
	}

	if time.Now().Unix() > claims.Exp {
		return nil, errors.New("token has expired")
	}

	return &UserContext{
		UserUUID:    claims.UserUUID,
		UserID:      claims.UserID,
		Email:       claims.Email,
		Role:        claims.Role,
		SessionUUID: claims.SessionUUID,
	}, nil
}

// GenerateRefreshToken generates a secure 32-byte random token and its SHA-256 hash
func GenerateRefreshToken() (token string, hash string, err error) {
	rawBytes := make([]byte, 32)
	if _, err := rand.Read(rawBytes); err != nil {
		return "", "", err
	}
	token = hex.EncodeToString(rawBytes)
	hash = HashToken(token)
	return token, hash, nil
}

// HashToken hashes a token using SHA-256 for database storage
func HashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

func hmacSHA256(data, secret []byte) []byte {
	mac := hmac.New(sha256.New, secret)
	mac.Write(data)
	return mac.Sum(nil)
}

type MagicLinkClaims struct {
	OtpUUID string `json:"otp_uuid"`
	Email   string `json:"email"`
	Exp     int64  `json:"exp"`
}

// SignMagicLinkToken creates an HMAC-SHA256 signed magic link token
func SignMagicLinkToken(otpUUID, email string, exp time.Time) (string, error) {
	claims := MagicLinkClaims{
		OtpUUID: otpUUID,
		Email:   email,
		Exp:     exp.Unix(),
	}
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	payload := base64.RawURLEncoding.EncodeToString(claimsJSON)
	sig := hmacSHA256([]byte(payload), getJWTSecret())
	sigB64 := base64.RawURLEncoding.EncodeToString(sig)
	return payload + "." + sigB64, nil
}

// VerifyMagicLinkToken verifies a magic link token and returns the otpUUID and email
func VerifyMagicLinkToken(tokenStr string) (string, string, error) {
	parts := strings.Split(tokenStr, ".")
	if len(parts) != 2 {
		return "", "", errors.New("invalid token format")
	}
	payload, sigB64 := parts[0], parts[1]
	expectedSig := hmacSHA256([]byte(payload), getJWTSecret())
	actualSig, err := base64.RawURLEncoding.DecodeString(sigB64)
	if err != nil {
		return "", "", errors.New("invalid signature encoding")
	}
	if !hmac.Equal(expectedSig, actualSig) {
		return "", "", errors.New("invalid token signature")
	}
	claimsJSON, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return "", "", errors.New("invalid payload encoding")
	}
	var claims MagicLinkClaims
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		return "", "", errors.New("invalid claims json")
	}
	if time.Now().Unix() > claims.Exp {
		return "", "", errors.New("magic link has expired")
	}
	return claims.OtpUUID, claims.Email, nil
}
