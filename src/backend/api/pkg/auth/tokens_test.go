package auth

import (
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

	token, err := SignMagicLinkToken(otpUUID, email, exp)
	if err != nil {
		t.Fatalf("SignMagicLinkToken failed: %v", err)
	}

	gotUUID, gotEmail, err := VerifyMagicLinkToken(token)
	if err != nil {
		t.Fatalf("VerifyMagicLinkToken failed: %v", err)
	}
	if gotUUID != otpUUID {
		t.Errorf("expected otpUUID %s, got %s", otpUUID, gotUUID)
	}
	if gotEmail != email {
		t.Errorf("expected email %s, got %s", email, gotEmail)
	}

	// Expired token test
	expiredToken, _ := SignMagicLinkToken(otpUUID, email, time.Now().Add(-1*time.Minute))
	_, _, err = VerifyMagicLinkToken(expiredToken)
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
