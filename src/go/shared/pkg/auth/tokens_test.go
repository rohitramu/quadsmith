package auth

import (
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"
)

func TestUUIDv7(t *testing.T) {
	id1 := NewUUIDv7()
	id2 := NewUUIDv7()

	if len(id1) != 36 || len(id2) != 36 {
		t.Fatalf("expected 36-char UUID, got %s and %s", id1, id2)
	}

	if id1 == id2 {
		t.Fatalf("UUIDv7 collision: %s == %s", id1, id2)
	}

	// Verify UUID v7 version nibble (7)
	if id1[14] != '7' || id2[14] != '7' {
		t.Fatalf("expected version 7 nibble, got %c and %c", id1[14], id2[14])
	}
}

func TestAccessTokenSigningAndVerification(t *testing.T) {
	userUUID := NewUUIDv7()
	userID := "pilot_apex"
	email := "alex@example.com"
	role := "USER"
	sessionUUID := NewUUIDv7()

	token, err := SignAccessToken(userUUID, userID, email, role, sessionUUID)
	if err != nil {
		t.Fatalf("SignAccessToken failed: %v", err)
	}

	claims, err := VerifyAccessToken(token)
	if err != nil {
		t.Fatalf("VerifyAccessToken failed: %v", err)
	}

	if claims.UserUUID != userUUID {
		t.Errorf("expected UserUUID %s, got %s", userUUID, claims.UserUUID)
	}
	if claims.UserID != userID {
		t.Errorf("expected UserID %s, got %s", userID, claims.UserID)
	}
	if claims.Email != email {
		t.Errorf("expected Email %s, got %s", email, claims.Email)
	}
	if claims.Role != role {
		t.Errorf("expected Role %s, got %s", role, claims.Role)
	}
	if claims.SessionUUID != sessionUUID {
		t.Errorf("expected SessionUUID %s, got %s", sessionUUID, claims.SessionUUID)
	}

	// Tampered token test
	tampered := token[:len(token)-5] + "aaaaa"
	_, err = VerifyAccessToken(tampered)
	if err == nil {
		t.Fatalf("expected tampered token verification to fail")
	}
}

func TestMagicLinkTokenSigningAndVerification(t *testing.T) {
	otpUUID := NewUUIDv7()
	email := "alex@example.com"
	exp := time.Now().Add(15 * time.Minute)

	token, err := SignMagicLinkToken(otpUUID, email, "login", "", exp)
	if err != nil {
		t.Fatalf("SignMagicLinkToken failed: %v", err)
	}

	claims, err := VerifyMagicLinkToken(token)
	if err != nil {
		t.Fatalf("VerifyMagicLinkToken failed: %v", err)
	}
	if claims.OtpUUID != otpUUID {
		t.Errorf("expected otpUUID %s, got %s", otpUUID, claims.OtpUUID)
	}
	if claims.Email != email {
		t.Errorf("expected email %s, got %s", email, claims.Email)
	}
	if claims.Purpose != "login" {
		t.Errorf("expected purpose login, got %s", claims.Purpose)
	}

	// Expired token test
	expiredToken, _ := SignMagicLinkToken(otpUUID, email, "login", "", time.Now().Add(-1*time.Minute))
	_, err = VerifyMagicLinkToken(expiredToken)
	if err == nil {
		t.Fatalf("expected expired magic link verification to fail")
	}
}

func TestRefreshTokenGenerationAndHashing(t *testing.T) {
	token, hash, err := GenerateRefreshToken()
	if err != nil {
		t.Fatalf("GenerateRefreshToken failed: %v", err)
	}
	if len(token) != 64 { // 32 bytes hex
		t.Fatalf("expected 64 hex chars, got %d", len(token))
	}
	if HashToken(token) != hash {
		t.Fatalf("HashToken mismatch")
	}
}

func TestChallengeTokenGeneration(t *testing.T) {
	token, hash, err := GenerateChallengeToken()
	if err != nil {
		t.Fatalf("GenerateChallengeToken failed: %v", err)
	}
	if len(token) != 64 {
		t.Fatalf("expected 64 hex chars, got %d", len(token))
	}
	if HashToken(token) != hash {
		t.Fatalf("HashToken mismatch for challenge token")
	}
}

func TestMergeTicketSigningAndVerification(t *testing.T) {
	sourceUserUUID := NewUUIDv7()
	targetUserUUID := NewUUIDv7()
	email := "alex_second@example.com"
	exp := time.Now().Add(15 * time.Minute)

	ticket, err := SignMergeTicket(sourceUserUUID, targetUserUUID, email, exp)
	if err != nil {
		t.Fatalf("SignMergeTicket failed: %v", err)
	}

	claims, err := VerifyMergeTicket(ticket)
	if err != nil {
		t.Fatalf("VerifyMergeTicket failed: %v", err)
	}
	if claims.SourceUserUUID != sourceUserUUID {
		t.Errorf("expected source %s, got %s", sourceUserUUID, claims.SourceUserUUID)
	}
	if claims.TargetUserUUID != targetUserUUID {
		t.Errorf("expected target %s, got %s", targetUserUUID, claims.TargetUserUUID)
	}
	if claims.Email != email {
		t.Errorf("expected email %s, got %s", email, claims.Email)
	}

	// Tampered ticket
	tampered := ticket[:len(ticket)-5] + "xxxxx"
	_, err = VerifyMergeTicket(tampered)
	if err == nil {
		t.Fatalf("expected tampered merge ticket verification to fail")
	}
}

func TestSetupTokenSigningAndVerification(t *testing.T) {
	email := "newpilot@example.com"
	token, err := SignSetupToken(email)
	if err != nil {
		t.Fatalf("SignSetupToken failed: %v", err)
	}

	gotEmail, err := VerifySetupToken(token)
	if err != nil {
		t.Fatalf("VerifySetupToken failed: %v", err)
	}
	if gotEmail != email {
		t.Errorf("expected email %s, got %s", email, gotEmail)
	}

	// Tampered setup token
	tampered := token[:len(token)-5] + "bbbbb"
	_, err = VerifySetupToken(tampered)
	if err == nil {
		t.Fatalf("expected tampered setup token verification to fail")
	}

	// Expired setup token
	claims := SetupTokenClaims{
		Email:   email,
		Exp:     time.Now().Add(-1 * time.Minute).Unix(),
		Purpose: "account_setup",
	}
	claimsJSON, _ := json.Marshal(claims)
	payload := base64.RawURLEncoding.EncodeToString(claimsJSON)
	sig := hmacSHA256([]byte(payload), deriveKey("setup_token"))
	sigB64 := base64.RawURLEncoding.EncodeToString(sig)
	expiredToken := payload + "." + sigB64

	_, err = VerifySetupToken(expiredToken)
	if err == nil {
		t.Fatalf("expected expired setup token verification to fail")
	}
}

func TestExportTokenSigningAndVerification(t *testing.T) {
	userUUID := NewUUIDv7()
	userID := "pilot_fpv"

	token, err := SignExportToken(userUUID, userID)
	if err != nil {
		t.Fatalf("SignExportToken failed: %v", err)
	}

	claims, err := VerifyExportToken(token)
	if err != nil {
		t.Fatalf("VerifyExportToken failed: %v", err)
	}

	if claims.UserUUID != userUUID {
		t.Errorf("expected UserUUID %s, got %s", userUUID, claims.UserUUID)
	}
	if claims.UserID != userID {
		t.Errorf("expected UserID %s, got %s", userID, claims.UserID)
	}
	if claims.Purpose != "export_data_download" {
		t.Errorf("expected Purpose 'export_data_download', got %s", claims.Purpose)
	}

	// Tampered export token
	tampered := token[:len(token)-5] + "eeeee"
	_, err = VerifyExportToken(tampered)
	if err == nil {
		t.Fatalf("expected tampered export token verification to fail")
	}

	// Expired export token
	expClaims := ExportTokenClaims{
		UserUUID: userUUID,
		UserID:   userID,
		Purpose:  "export_data_download",
		Exp:      time.Now().Add(-1 * time.Minute).Unix(),
	}
	claimsJSON, _ := json.Marshal(expClaims)
	payload := base64.RawURLEncoding.EncodeToString(claimsJSON)
	sig := hmacSHA256([]byte(payload), deriveKey("export_token"))
	sigB64 := base64.RawURLEncoding.EncodeToString(sig)
	expiredToken := payload + "." + sigB64

	_, err = VerifyExportToken(expiredToken)
	if err == nil {
		t.Fatalf("expected expired export token verification to fail")
	}
}

func TestPKCEGeneration(t *testing.T) {
	verifier, challenge, err := GeneratePKCE()
	if err != nil {
		t.Fatalf("GeneratePKCE failed: %v", err)
	}
	if len(verifier) < 43 {
		t.Fatalf("expected verifier length >= 43, got %d", len(verifier))
	}
	if len(challenge) == 0 {
		t.Fatalf("expected non-empty challenge")
	}
}

func TestOAuthStateTokenSigningAndVerification(t *testing.T) {
	provider := "github"
	purpose := "cli_login"
	codeVerifier := "some-pkce-verifier-string"
	redirectURI := "http://localhost:5173/login"
	requesterUserUUID := NewUUIDv7()
	challengeIDHash := "hash123"

	state, err := SignOAuthState(provider, purpose, codeVerifier, redirectURI, requesterUserUUID, challengeIDHash)
	if err != nil {
		t.Fatalf("SignOAuthState failed: %v", err)
	}

	claims, err := VerifyOAuthState(state)
	if err != nil {
		t.Fatalf("VerifyOAuthState failed: %v", err)
	}
	if claims.Provider != provider {
		t.Errorf("expected provider %s, got %s", provider, claims.Provider)
	}
	if claims.Purpose != purpose {
		t.Errorf("expected purpose %s, got %s", purpose, claims.Purpose)
	}
	if claims.CodeVerifier != codeVerifier {
		t.Errorf("expected code_verifier %s, got %s", codeVerifier, claims.CodeVerifier)
	}
	if claims.RedirectURI != redirectURI {
		t.Errorf("expected redirect_uri %s, got %s", redirectURI, claims.RedirectURI)
	}
	if claims.RequesterUserUUID != requesterUserUUID {
		t.Errorf("expected requesterUserUUID %s, got %s", requesterUserUUID, claims.RequesterUserUUID)
	}
	if claims.ChallengeIDHash != challengeIDHash {
		t.Errorf("expected challengeIDHash %s, got %s", challengeIDHash, claims.ChallengeIDHash)
	}

	// Tampered state
	tampered := state[:len(state)-5] + "xxxxx"
	_, err = VerifyOAuthState(tampered)
	if err == nil {
		t.Fatalf("expected tampered oauth state verification to fail")
	}

	// Expired state
	expClaims := OAuthStateClaims{
		Provider:     provider,
		Purpose:      purpose,
		CodeVerifier: codeVerifier,
		RedirectURI:  redirectURI,
		Exp:          time.Now().Add(-1 * time.Minute).Unix(),
	}
	claimsJSON, _ := json.Marshal(expClaims)
	payload := base64.RawURLEncoding.EncodeToString(claimsJSON)
	sig := hmacSHA256([]byte(payload), deriveKey("oauth_state"))
	sigB64 := base64.RawURLEncoding.EncodeToString(sig)
	expiredState := payload + "." + sigB64

	_, err = VerifyOAuthState(expiredState)
	if err == nil {
		t.Fatalf("expected expired oauth state verification to fail")
	}
}

func TestOAuthSetupTokenSigningAndVerification(t *testing.T) {
	primaryEmail := "pilot@example.com"
	verifiedEmails := []string{"pilot@example.com", "alt@example.com"}
	provider := "github"
	providerAccountID := "github_12345"
	avatarURL := "https://example.com/avatar.png"

	token, err := SignOAuthSetupToken(primaryEmail, verifiedEmails, provider, providerAccountID, avatarURL, "")
	if err != nil {
		t.Fatalf("SignOAuthSetupToken failed: %v", err)
	}

	claims, err := VerifySetupTokenClaims(token)
	if err != nil {
		t.Fatalf("VerifySetupTokenClaims failed: %v", err)
	}
	if claims.Email != primaryEmail {
		t.Errorf("expected Email %s, got %s", primaryEmail, claims.Email)
	}
	if len(claims.VerifiedEmails) != 2 || claims.VerifiedEmails[1] != "alt@example.com" {
		t.Errorf("expected VerifiedEmails %v, got %v", verifiedEmails, claims.VerifiedEmails)
	}
	if claims.Provider != provider {
		t.Errorf("expected Provider %s, got %s", provider, claims.Provider)
	}
	if claims.ProviderAccountID != providerAccountID {
		t.Errorf("expected ProviderAccountID %s, got %s", providerAccountID, claims.ProviderAccountID)
	}
	if claims.AvatarURL != avatarURL {
		t.Errorf("expected AvatarURL %s, got %s", avatarURL, claims.AvatarURL)
	}

	// Test backward-compat VerifySetupToken
	email, err := VerifySetupToken(token)
	if err != nil {
		t.Fatalf("VerifySetupToken failed: %v", err)
	}
	if email != primaryEmail {
		t.Errorf("expected email %s, got %s", primaryEmail, email)
	}
}
