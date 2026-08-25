package auth

import (
	"errors"
	"fmt"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"
)

const (
	minPasswordLength = 8
	maxPasswordLength = 72
)

var (
	ErrInvalidPassword    = errors.New("invalid password")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

type PasswordManager struct {
	cost int
}

func NewPasswordManager(cost int) (PasswordManager, error) {
	if cost < bcrypt.MinCost || cost > bcrypt.MaxCost {
		return PasswordManager{}, fmt.Errorf("bcrypt cost must be between %d and %d", bcrypt.MinCost, bcrypt.MaxCost)
	}

	return PasswordManager{cost: cost}, nil
}

func (manager PasswordManager) Hash(password string) (string, error) {
	if err := validatePassword(password); err != nil {
		return "", err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), manager.cost)
	if err != nil {
		return "", fmt.Errorf("generate password hash: %w", err)
	}

	return string(hash), nil
}

func (manager PasswordManager) Compare(hash string, password string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err != nil {
		return ErrInvalidCredentials
	}

	return nil
}

func validatePassword(password string) error {
	if utf8.RuneCountInString(password) < minPasswordLength || len(password) > maxPasswordLength {
		return ErrInvalidPassword
	}

	return nil
}
