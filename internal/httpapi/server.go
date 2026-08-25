package httpapi

import (
	"context"
	"net/http"

	"github.com/virtuxa/tz_task_manager/internal/user"
)

type AuthenticationService interface {
	Register(context.Context, user.RegisterInput) (user.User, error)
	Login(context.Context, user.LoginInput) (user.LoginResult, error)
}

func NewHandler(authentication AuthenticationService) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", health)

	authenticationHandler := newAuthenticationHandler(authentication)
	mux.HandleFunc("POST /api/v1/register", authenticationHandler.register)
	mux.HandleFunc("POST /api/v1/login", authenticationHandler.login)

	return mux
}

func health(writer http.ResponseWriter, _ *http.Request) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write([]byte(`{"status":"ok"}`))
}
