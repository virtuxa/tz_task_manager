package auth

import (
	"errors"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestPasswordManagerHashAndCompare(t *testing.T) {
	manager, err := NewPasswordManager(bcrypt.MinCost)
	if err != nil {
		t.Fatalf("create password manager: %v", err)
	}

	hash, err := manager.Hash("password1")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	if hash == "password1" {
		t.Fatal("password hash must not contain plain password")
	}

	if err := manager.Compare(hash, "password1"); err != nil {
		t.Fatalf("compare correct password: %v", err)
	}

	if err := manager.Compare(hash, "wrong-password"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("compare wrong password error = %v, want %v", err, ErrInvalidCredentials)
	}
}

func TestPasswordManagerRejectsInvalidPassword(t *testing.T) {
	manager, err := NewPasswordManager(bcrypt.MinCost)
	if err != nil {
		t.Fatalf("create password manager: %v", err)
	}

	if _, err := manager.Hash("short"); !errors.Is(err, ErrInvalidPassword) {
		t.Fatalf("hash short password error = %v, want %v", err, ErrInvalidPassword)
	}
}
