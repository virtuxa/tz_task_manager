package user

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
)

type Service struct {
	repository Repository
	passwords  PasswordManager
	tokens     TokenIssuer
}

func NewService(repository Repository, passwords PasswordManager, tokens TokenIssuer) (*Service, error) {
	if repository == nil {
		return nil, errors.New("user repository is required")
	}

	if passwords == nil {
		return nil, errors.New("password manager is required")
	}

	if tokens == nil {
		return nil, errors.New("token issuer is required")
	}

	return &Service{
		repository: repository,
		passwords:  passwords,
		tokens:     tokens,
	}, nil
}

func (service *Service) Register(ctx context.Context, input RegisterInput) (User, error) {
	email, name, err := validateRegistration(input)
	if err != nil {
		return User{}, err
	}

	passwordHash, err := service.passwords.Hash(input.Password)
	if err != nil {
		return User{}, fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}

	createdUser, err := service.repository.Create(ctx, User{
		Email:        email,
		PasswordHash: passwordHash,
		Name:         name,
	})
	if err != nil {
		return User{}, err
	}

	return createdUser, nil
}

func (service *Service) Login(ctx context.Context, input LoginInput) (LoginResult, error) {
	email, err := validateLogin(input)
	if err != nil {
		return LoginResult{}, err
	}

	registeredUser, err := service.repository.FindByEmail(ctx, email)
	if errors.Is(err, ErrNotFound) {
		return LoginResult{}, ErrInvalidCredentials
	}
	if err != nil {
		return LoginResult{}, err
	}

	if err := service.passwords.Compare(registeredUser.PasswordHash, input.Password); err != nil {
		return LoginResult{}, ErrInvalidCredentials
	}

	token, err := service.tokens.Issue(registeredUser.ID)
	if err != nil {
		return LoginResult{}, fmt.Errorf("issue access token: %w", err)
	}

	return LoginResult{AccessToken: token}, nil
}

func validateRegistration(input RegisterInput) (string, string, error) {
	email, err := normalizeEmail(input.Email)
	if err != nil {
		return "", "", err
	}

	name := strings.TrimSpace(input.Name)
	if name == "" || len(name) > maxNameLength {
		return "", "", ErrInvalidInput
	}

	return email, name, nil
}

func validateLogin(input LoginInput) (string, error) {
	email, err := normalizeEmail(input.Email)
	if err != nil {
		return "", err
	}

	if input.Password == "" {
		return "", ErrInvalidInput
	}

	return email, nil
}

func normalizeEmail(value string) (string, error) {
	email := strings.ToLower(strings.TrimSpace(value))
	if email == "" || len(email) > maxEmailLength {
		return "", ErrInvalidInput
	}

	parsed, err := mail.ParseAddress(email)
	if err != nil || parsed.Address != email {
		return "", ErrInvalidInput
	}

	return email, nil
}
