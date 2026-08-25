package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/virtuxa/tz_task_manager/internal/user"
)

func TestHealth(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	response := httptest.NewRecorder()

	NewHandler(authenticationServiceStub{}, teamServiceStub{}, taskServiceStub{}, tokenParserStub{}).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", response.Code, http.StatusOK)
	}

	if contentType := response.Header().Get("Content-Type"); contentType != "application/json" {
		t.Fatalf("content type = %q, want %q", contentType, "application/json")
	}

	if body := response.Body.String(); body != `{"status":"ok"}` {
		t.Fatalf("body = %q, want %q", body, `{"status":"ok"}`)
	}
}

func TestRegister(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/register", strings.NewReader(`{"email":"user@example.com","password":"password1","name":"Анна"}`))
	response := httptest.NewRecorder()

	NewHandler(authenticationServiceStub{
		registeredUser: user.User{ID: 5, Email: "user@example.com", Name: "Анна"},
	}, teamServiceStub{}, taskServiceStub{}, tokenParserStub{}).ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status code = %d, want %d", response.Code, http.StatusCreated)
	}

	if body := response.Body.String(); body != `{"id":5,"email":"user@example.com","name":"Анна"}`+"\n" {
		t.Fatalf("body = %q", body)
	}
}

func TestLoginRejectsInvalidCredentials(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/login", strings.NewReader(`{"email":"user@example.com","password":"wrong"}`))
	response := httptest.NewRecorder()

	NewHandler(authenticationServiceStub{loginError: user.ErrInvalidCredentials}, teamServiceStub{}, taskServiceStub{}, tokenParserStub{}).ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status code = %d, want %d", response.Code, http.StatusUnauthorized)
	}

	if body := response.Body.String(); body != `{"error":{"code":"invalid_credentials","message":"invalid email or password"}}`+"\n" {
		t.Fatalf("body = %q", body)
	}
}

type authenticationServiceStub struct {
	registeredUser user.User
	registerError  error
	loginResult    user.LoginResult
	loginError     error
}

func (stub authenticationServiceStub) Register(_ context.Context, _ user.RegisterInput) (user.User, error) {
	return stub.registeredUser, stub.registerError
}

func (stub authenticationServiceStub) Login(_ context.Context, _ user.LoginInput) (user.LoginResult, error) {
	return stub.loginResult, stub.loginError
}
