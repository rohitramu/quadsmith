package auth

import (
	"crypto/hkdf"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"
)

var (
	masterSecretMu  sync.RWMutex
	masterSecret    = []byte("quadsmith-dev-secret-key-32bytes-long!")
	accessTokenTTL  = 1 * time.Hour
	refreshTokenTTL = 5 * 24 * time.Hour
)

const (
	AccessTokenTTL  = 1 * time.Hour
	RefreshTokenTTL = 5 * 24 * time.Hour // 5-day sliding window
)

// SetMasterSecret sets the master secret for token and key derivation.
func SetMasterSecret(secret string) {
	masterSecretMu.Lock()
	defer masterSecretMu.Unlock()
	masterSecret = []byte(secret)
}

// SetTokenTTLs sets the runtime token TTLs.
func SetTokenTTLs(accessTTL, refreshTTL time.Duration) {
	masterSecretMu.Lock()
	defer masterSecretMu.Unlock()
	if accessTTL > 0 {
		accessTokenTTL = accessTTL
	}
	if refreshTTL > 0 {
		refreshTokenTTL = refreshTTL
	}
}

func getMasterSecret() []byte {
	masterSecretMu.RLock()
	defer masterSecretMu.RUnlock()
	return masterSecret
}

func deriveKey(purpose string) []byte {
	k, err := hkdf.Key(sha256.New, getMasterSecret(), nil, "quadsmith:"+purpose, 32)
	if err != nil {
		panic(fmt.Sprintf("deriveKey failed: %v", err))
	}
	return k
}

type jwtHeader struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
}

type JWTClaims struct {
	UserUUID    string `json:"user_uuid"`
	UserID      string `json:"user_id"` // Public username (e.g. "fpv_pilot")
	Email       string `json:"email"`
	SessionUUID string `json:"session_uuid"`
	Exp         int64  `json:"exp"`
	Iat         int64  `json:"iat"`
}

// SignAccessToken signs a new 1-hour access JWT using HMAC-SHA256 with a dedicated subkey
func SignAccessToken(userUUID, userID, email, sessionUUID string) (string, error) {
	return SignAccessTokenWithExpiry(userUUID, userID, email, sessionUUID, time.Time{})
}

// SignAccessTokenWithExpiry signs an access JWT whose expiration is capped at maxExpiresAt (if non-zero).
func SignAccessTokenWithExpiry(userUUID, userID, email, sessionUUID string, maxExpiresAt time.Time) (string, error) {
	now := time.Now()
	masterSecretMu.RLock()
	ttl := accessTokenTTL
	masterSecretMu.RUnlock()

	expTime := now.Add(ttl)
	if !maxExpiresAt.IsZero() && maxExpiresAt.Before(expTime) {
		expTime = maxExpiresAt
	}

	claims := JWTClaims{
		UserUUID:    userUUID,
		UserID:      userID,
		Email:       email,
		SessionUUID: sessionUUID,
		Iat:         now.Unix(),
		Exp:         expTime.Unix(),
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
	sig := hmacSHA256([]byte(payload), deriveKey("access_token"))
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
	expectedSig := hmacSHA256([]byte(headerB64+"."+claimsB64), deriveKey("access_token"))
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
		Role:        "USER",
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

// GenerateChallengeToken generates a secure 32-byte challenge token and its SHA-256 hash
func GenerateChallengeToken() (token string, hash string, err error) {
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
	OtpUUID           string `json:"otp_uuid"`
	Email             string `json:"email"`
	Purpose           string `json:"purpose,omitempty"`
	RequesterUserUUID string `json:"requester_user_uuid,omitempty"`
	Exp               int64  `json:"exp"`
}

// SignMagicLinkToken creates an HMAC-SHA256 signed magic link token bound to a purpose
func SignMagicLinkToken(otpUUID, email, purpose, requesterUserUUID string, exp time.Time) (string, error) {
	claims := MagicLinkClaims{
		OtpUUID:           otpUUID,
		Email:             strings.ToLower(strings.TrimSpace(email)),
		Purpose:           purpose,
		RequesterUserUUID: requesterUserUUID,
		Exp:               exp.Unix(),
	}
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	payload := base64.RawURLEncoding.EncodeToString(claimsJSON)
	sig := hmacSHA256([]byte(payload), deriveKey("magic_link"))
	sigB64 := base64.RawURLEncoding.EncodeToString(sig)
	return payload + "." + sigB64, nil
}

// VerifyMagicLinkToken verifies a magic link token and returns the full claims
func VerifyMagicLinkToken(tokenStr string) (*MagicLinkClaims, error) {
	parts := strings.Split(tokenStr, ".")
	if len(parts) != 2 {
		return nil, errors.New("invalid token format")
	}
	payload, sigB64 := parts[0], parts[1]
	expectedSig := hmacSHA256([]byte(payload), deriveKey("magic_link"))
	actualSig, err := base64.RawURLEncoding.DecodeString(sigB64)
	if err != nil {
		return nil, errors.New("invalid signature encoding")
	}
	if !hmac.Equal(expectedSig, actualSig) {
		return nil, errors.New("invalid token signature")
	}
	claimsJSON, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return nil, errors.New("invalid payload encoding")
	}
	var claims MagicLinkClaims
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		return nil, errors.New("invalid claims json")
	}
	if time.Now().Unix() > claims.Exp {
		return nil, errors.New("magic link has expired")
	}
	return &claims, nil
}

// SetupTokenClaims contains claims for deferred account creation after email verification
type SetupTokenClaims struct {
	Email             string   `json:"email"`
	VerifiedEmails    []string `json:"verified_emails,omitempty"`
	Provider          string   `json:"provider,omitempty"`
	ProviderAccountID string   `json:"provider_account_id,omitempty"`
	AvatarURL         string   `json:"avatar_url,omitempty"`
	ChallengeIDHash   string   `json:"challenge_id_hash,omitempty"`
	Exp               int64    `json:"exp"`
	Purpose           string   `json:"purpose"`
}

const SetupTokenTTL = 15 * time.Minute

// SignSetupToken creates a signed setup token for an email that completed OTP or magic link verification
func SignSetupToken(email string) (string, error) {
	return SignOAuthSetupToken(email, nil, "", "", "", "")
}

// SignOAuthSetupToken creates a signed setup token encoding primary email, secondary verified emails, and OAuth identity
func SignOAuthSetupToken(email string, verifiedEmails []string, provider, providerAccountID, avatarURL, challengeIDHash string) (string, error) {
	cleanEmails := make([]string, 0, len(verifiedEmails))
	for _, e := range verifiedEmails {
		norm := strings.ToLower(strings.TrimSpace(e))
		if norm != "" && !slices.Contains(cleanEmails, norm) {
			cleanEmails = append(cleanEmails, norm)
		}
	}
	claims := SetupTokenClaims{
		Email:             strings.ToLower(strings.TrimSpace(email)),
		VerifiedEmails:    cleanEmails,
		Provider:          provider,
		ProviderAccountID: providerAccountID,
		AvatarURL:         avatarURL,
		ChallengeIDHash:   challengeIDHash,
		Exp:               time.Now().Add(SetupTokenTTL).Unix(),
		Purpose:           "account_setup",
	}
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	payload := base64.RawURLEncoding.EncodeToString(claimsJSON)
	sig := hmacSHA256([]byte(payload), deriveKey("setup_token"))
	sigB64 := base64.RawURLEncoding.EncodeToString(sig)
	return payload + "." + sigB64, nil
}

// VerifySetupTokenClaims verifies a setup token and returns the parsed claims
func VerifySetupTokenClaims(tokenStr string) (*SetupTokenClaims, error) {
	parts := strings.Split(tokenStr, ".")
	if len(parts) != 2 {
		return nil, errors.New("invalid setup token format")
	}
	payload, sigB64 := parts[0], parts[1]
	expectedSig := hmacSHA256([]byte(payload), deriveKey("setup_token"))
	actualSig, err := base64.RawURLEncoding.DecodeString(sigB64)
	if err != nil {
		return nil, errors.New("invalid signature encoding")
	}
	if !hmac.Equal(expectedSig, actualSig) {
		return nil, errors.New("invalid token signature")
	}
	claimsJSON, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return nil, errors.New("invalid payload encoding")
	}
	var claims SetupTokenClaims
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		return nil, errors.New("invalid claims json")
	}
	if claims.Purpose != "account_setup" {
		return nil, errors.New("invalid token purpose")
	}
	if time.Now().Unix() > claims.Exp {
		return nil, errors.New("setup token has expired. Please verify your email again")
	}
	return &claims, nil
}

// VerifySetupToken verifies a setup token and returns the verified email
func VerifySetupToken(tokenStr string) (string, error) {
	claims, err := VerifySetupTokenClaims(tokenStr)
	if err != nil {
		return "", err
	}
	return claims.Email, nil
}

// GeneratePKCE returns a cryptographically secure random code_verifier and its RFC 7636 S256 code_challenge
func GeneratePKCE() (verifier, challenge string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}
	verifier = base64.RawURLEncoding.EncodeToString(b)
	h := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(h[:])
	return verifier, challenge, nil
}

// OAuthStateClaims contains claims embedded in the signed OAuth state token
type OAuthStateClaims struct {
	Provider          string `json:"provider"`
	Purpose           string `json:"purpose"`
	CodeVerifier      string `json:"code_verifier"`
	RedirectURI       string `json:"redirect_uri"`
	RequesterUserUUID string `json:"requester_user_uuid,omitempty"`
	ChallengeIDHash   string `json:"challenge_id_hash,omitempty"`
	Exp               int64  `json:"exp"`
}

const OAuthStateTTL = 10 * time.Minute

// SignOAuthState signs an HMAC-SHA256 state token binding the flow parameters and PKCE verifier
func SignOAuthState(provider, purpose, codeVerifier, redirectURI, requesterUserUUID, challengeIDHash string) (string, error) {
	claims := OAuthStateClaims{
		Provider:          provider,
		Purpose:           purpose,
		CodeVerifier:      codeVerifier,
		RedirectURI:       redirectURI,
		RequesterUserUUID: requesterUserUUID,
		ChallengeIDHash:   challengeIDHash,
		Exp:               time.Now().Add(OAuthStateTTL).Unix(),
	}
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	payload := base64.RawURLEncoding.EncodeToString(claimsJSON)
	sig := hmacSHA256([]byte(payload), deriveKey("oauth_state"))
	sigB64 := base64.RawURLEncoding.EncodeToString(sig)
	return payload + "." + sigB64, nil
}

// VerifyOAuthState verifies an OAuth state token signature, validity, and expiration
func VerifyOAuthState(stateStr string) (*OAuthStateClaims, error) {
	parts := strings.Split(stateStr, ".")
	if len(parts) != 2 {
		return nil, errors.New("invalid oauth state format")
	}
	payload, sigB64 := parts[0], parts[1]
	expectedSig := hmacSHA256([]byte(payload), deriveKey("oauth_state"))
	actualSig, err := base64.RawURLEncoding.DecodeString(sigB64)
	if err != nil {
		return nil, errors.New("invalid oauth state signature encoding")
	}
	if !hmac.Equal(expectedSig, actualSig) {
		return nil, errors.New("invalid oauth state signature")
	}
	claimsJSON, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return nil, errors.New("invalid oauth state payload encoding")
	}
	var claims OAuthStateClaims
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		return nil, errors.New("invalid oauth state claims json")
	}
	if time.Now().Unix() > claims.Exp {
		return nil, errors.New("oauth state has expired. Please try signing in again")
	}
	return &claims, nil
}

// MergeTicketClaims defines the cryptographically signed authorization ticket for account merges
type MergeTicketClaims struct {
	SourceUserUUID    string   `json:"source_user_uuid"`
	TargetUserUUID    string   `json:"target_user_uuid"`
	Email             string   `json:"email"`
	VerifiedEmails    []string `json:"verified_emails,omitempty"`
	Provider          string   `json:"provider,omitempty"`
	ProviderAccountID string   `json:"provider_account_id,omitempty"`
	Exp               int64    `json:"exp"`
	Purpose           string   `json:"purpose"`
}

// SignMergeTicket creates a signed ticket proving that an authenticated user verified ownership of an existing account's email
func SignMergeTicket(sourceUserUUID, targetUserUUID, email string, exp time.Time) (string, error) {
	return SignOAuthMergeTicket(sourceUserUUID, targetUserUUID, email, nil, "", "", exp)
}

// SignOAuthMergeTicket creates a signed merge ticket with optional OAuth identity provenance
func SignOAuthMergeTicket(sourceUserUUID, targetUserUUID, email string, verifiedEmails []string, provider, providerAccountID string, exp time.Time) (string, error) {
	cleanEmails := make([]string, 0, len(verifiedEmails))
	for _, e := range verifiedEmails {
		norm := strings.ToLower(strings.TrimSpace(e))
		if norm != "" && !slices.Contains(cleanEmails, norm) {
			cleanEmails = append(cleanEmails, norm)
		}
	}
	claims := MergeTicketClaims{
		SourceUserUUID:    sourceUserUUID,
		TargetUserUUID:    targetUserUUID,
		Email:             strings.ToLower(strings.TrimSpace(email)),
		VerifiedEmails:    cleanEmails,
		Provider:          provider,
		ProviderAccountID: providerAccountID,
		Exp:               exp.Unix(),
		Purpose:           "account_merge",
	}
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	payload := base64.RawURLEncoding.EncodeToString(claimsJSON)
	sig := hmacSHA256([]byte(payload), deriveKey("merge_ticket"))
	sigB64 := base64.RawURLEncoding.EncodeToString(sig)
	return payload + "." + sigB64, nil
}

// VerifyMergeTicket verifies the signature, validity, and purpose of an account merge authorization ticket
func VerifyMergeTicket(ticketStr string) (*MergeTicketClaims, error) {
	parts := strings.Split(ticketStr, ".")
	if len(parts) != 2 {
		return nil, errors.New("invalid merge ticket format")
	}
	payload, sigB64 := parts[0], parts[1]
	expectedSig := hmacSHA256([]byte(payload), deriveKey("merge_ticket"))
	actualSig, err := base64.RawURLEncoding.DecodeString(sigB64)
	if err != nil {
		return nil, errors.New("invalid signature encoding")
	}
	if !hmac.Equal(expectedSig, actualSig) {
		return nil, errors.New("invalid merge ticket signature")
	}
	claimsJSON, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return nil, errors.New("invalid payload encoding")
	}
	var claims MergeTicketClaims
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		return nil, errors.New("invalid claims json")
	}
	if claims.Purpose != "account_merge" {
		return nil, errors.New("invalid merge ticket purpose")
	}
	if time.Now().Unix() > claims.Exp {
		return nil, errors.New("merge ticket has expired. Please verify the email again")
	}
	return &claims, nil
}

// ExportTokenClaims contains claims for authorizing GDPR data archive downloads
type ExportTokenClaims struct {
	UserUUID string `json:"user_uuid"`
	UserID   string `json:"user_id"` // username
	Purpose  string `json:"purpose"`
	Exp      int64  `json:"exp"`
}

const ExportTokenTTL = 15 * time.Minute

// SignExportToken creates an HMAC-SHA256 signed token authorizing a user to download their GDPR data archive
func SignExportToken(userUUID, userID string) (string, error) {
	claims := ExportTokenClaims{
		UserUUID: userUUID,
		UserID:   userID,
		Purpose:  "export_data_download",
		Exp:      time.Now().Add(ExportTokenTTL).Unix(),
	}
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	payload := base64.RawURLEncoding.EncodeToString(claimsJSON)
	sig := hmacSHA256([]byte(payload), deriveKey("export_token"))
	sigB64 := base64.RawURLEncoding.EncodeToString(sig)
	return payload + "." + sigB64, nil
}

// VerifyExportToken verifies an export download token and returns its claims
func VerifyExportToken(tokenStr string) (*ExportTokenClaims, error) {
	parts := strings.Split(tokenStr, ".")
	if len(parts) != 2 {
		return nil, errors.New("invalid export token format")
	}
	payload, sigB64 := parts[0], parts[1]
	expectedSig := hmacSHA256([]byte(payload), deriveKey("export_token"))
	actualSig, err := base64.RawURLEncoding.DecodeString(sigB64)
	if err != nil {
		return nil, errors.New("invalid signature encoding")
	}
	if !hmac.Equal(expectedSig, actualSig) {
		return nil, errors.New("invalid export token signature")
	}
	claimsJSON, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return nil, errors.New("invalid payload encoding")
	}
	var claims ExportTokenClaims
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		return nil, errors.New("invalid claims json")
	}
	if claims.Purpose != "export_data_download" {
		return nil, errors.New("invalid export token purpose")
	}
	if time.Now().Unix() > claims.Exp {
		return nil, errors.New("export token has expired. Please verify your email again")
	}
	return &claims, nil
}
