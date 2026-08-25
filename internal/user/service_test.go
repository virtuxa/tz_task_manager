package user

import (
	"context"
	"errors"
	"strconv"
	"testing"
)

func TestServiceRegister(t *testing.T) {
	repository := &repositoryStub{}
	service := newTestService(t, repository)

	registeredUser, err := service.Register(context.Background(), RegisterInput{
		Email:    "  USER@Example.com ",
		Password: "password1",
		Name:     " Анна ",
	})
	if err != nil {
		t.Fatalf("register user: %v", err)
	}

	if registeredUser.Email != "user@example.com" {
		t.Fatalf("email = %q, want %q", registeredUser.Email, "user@example.com")
	}

	if registeredUser.Name != "Анна" {
		t.Fatalf("name = %q, want %q", registeredUser.Name, "Анна")
	}

	if repository.created.PasswordHash == "password1" {
		t.Fatal("repository received plain password")
	}
}

func TestServiceLoginReturnsGenericErrorForUnknownEmail(t *testing.T) {
	service := newTestService(t, &repositoryStub{findError: ErrNotFound})

	_, err := service.Login(context.Background(), LoginInput{
		Email:    "unknown@example.com",
		Password: "password1",
	})
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("login error = %v, want %v", err, ErrInvalidCredentials)
	}
}

func TestServiceLoginIssuesToken(t *testing.T) {
	repository := &repositoryStub{
		found: User{
			ID:           5,
			Email:        "user@example.com",
			PasswordHash: "stored-password",
		},
	}
	service := newTestService(t, repository)

	result, err := service.Login(context.Background(), LoginInput{
		Email:    "user@example.com",
		Password: "password1",
	})
	if err != nil {
		t.Fatalf("login user: %v", err)
	}

	if result.AccessToken != "token-for-5" {
		t.Fatalf("access token = %q", result.AccessToken)
	}
}

func newTestService(t *testing.T, repository Repository) *Service {
	t.Helper()

	service, err := NewService(repository, passwordManagerStub{}, tokenIssuerStub{})
	if err != nil {
		t.Fatalf("create user service: %v", err)
	}

	return service
}

type repositoryStub struct {
	created   User
	createErr error
	found     User
	findError error
}

func (stub *repositoryStub) Create(_ context.Context, user User) (User, error) {
	if stub.createErr != nil {
		return User{}, stub.createErr
	}

	stub.created = user
	user.ID = 5
	return user, nil
}

func (stub *repositoryStub) FindByEmail(_ context.Context, _ string) (User, error) {
	return stub.found, stub.findError
}

type passwordManagerStub struct{}

func (passwordManagerStub) Hash(password string) (string, error) {
	if password == "" {
		return "", errors.New("password is empty")
	}

	return "hashed-" + password, nil
}

func (passwordManagerStub) Compare(hash string, password string) error {
	if hash != "stored-password" || password != "password1" {
		return errors.New("password does not match")
	}

	return nil
}

type tokenIssuerStub struct{}

func (tokenIssuerStub) Issue(userID int64) (string, error) {
	return "token-for-" + strconv.FormatInt(userID, 10), nil
}
