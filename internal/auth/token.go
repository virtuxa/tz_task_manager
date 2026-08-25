package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type TokenManager struct {
	signingKey []byte
	ttl        time.Duration
	now        func() time.Time
}

type claims struct {
	UserID int64 `json:"user_id"`
	jwt.RegisteredClaims
}

func NewTokenManager(signingKey string, ttl time.Duration) (*TokenManager, error) {
	if len(signingKey) < minSigningKeyLength {
		return nil, fmt.Errorf("JWT signing key must be at least %d bytes", minSigningKeyLength)
	}

	if ttl <= 0 {
		return nil, errors.New("JWT TTL must be positive")
	}

	return &TokenManager{
		signingKey: []byte(signingKey),
		ttl:        ttl,
		now:        time.Now,
	}, nil
}

func (manager *TokenManager) Issue(userID int64) (string, error) {
	if userID <= 0 {
		return "", errors.New("user ID must be positive")
	}

	issuedAt := manager.now().UTC()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(issuedAt),
			ExpiresAt: jwt.NewNumericDate(issuedAt.Add(manager.ttl)),
		},
	})

	signedToken, err := token.SignedString(manager.signingKey)
	if err != nil {
		return "", fmt.Errorf("sign JWT: %w", err)
	}

	return signedToken, nil
}

func (manager *TokenManager) Parse(rawToken string) (int64, error) {
	parsedClaims := new(claims)
	token, err := jwt.ParseWithClaims(
		rawToken,
		parsedClaims,
		func(token *jwt.Token) (any, error) {
			if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
				return nil, ErrInvalidToken
			}

			return manager.signingKey, nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithTimeFunc(manager.now),
	)
	if err != nil || !token.Valid || parsedClaims.UserID <= 0 {
		return 0, ErrInvalidToken
	}

	return parsedClaims.UserID, nil
}
