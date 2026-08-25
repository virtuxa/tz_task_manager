package auth

import (
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const testSigningKey = "test-signing-key-with-at-least-thirty-two-bytes"

func TestTokenManagerIssueAndParse(t *testing.T) {
	manager, err := NewTokenManager(testSigningKey, time.Hour)
	if err != nil {
		t.Fatalf("create token manager: %v", err)
	}

	issuedAt := time.Date(2026, time.August, 25, 12, 0, 0, 0, time.UTC)
	manager.now = func() time.Time { return issuedAt }

	token, err := manager.Issue(42)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}

	userID, err := manager.Parse(token)
	if err != nil {
		t.Fatalf("parse token: %v", err)
	}

	if userID != 42 {
		t.Fatalf("user ID = %d, want 42", userID)
	}
}

func TestTokenManagerRejectsExpiredToken(t *testing.T) {
	manager, err := NewTokenManager(testSigningKey, time.Hour)
	if err != nil {
		t.Fatalf("create token manager: %v", err)
	}

	issuedAt := time.Date(2026, time.August, 25, 12, 0, 0, 0, time.UTC)
	manager.now = func() time.Time { return issuedAt }

	token, err := manager.Issue(42)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}

	manager.now = func() time.Time { return issuedAt.Add(time.Hour + time.Second) }
	if _, err := manager.Parse(token); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("parse expired token error = %v, want %v", err, ErrInvalidToken)
	}
}

func TestTokenManagerRejectsUnexpectedSigningMethod(t *testing.T) {
	manager, err := NewTokenManager(testSigningKey, time.Hour)
	if err != nil {
		t.Fatalf("create token manager: %v", err)
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS384, claims{UserID: 42})
	rawToken, err := token.SignedString([]byte(testSigningKey))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	if _, err := manager.Parse(rawToken); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("parse token with unexpected signing method error = %v, want %v", err, ErrInvalidToken)
	}
}
