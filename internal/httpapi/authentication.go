package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/virtuxa/tz_task_manager/internal/user"
)

const maxJSONBodySize = 1 << 20

type authenticationHandler struct {
	service AuthenticationService
}

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type userResponse struct {
	ID    int64  `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

type loginResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
}

func newAuthenticationHandler(service AuthenticationService) *authenticationHandler {
	return &authenticationHandler{service: service}
}

func (handler *authenticationHandler) register(writer http.ResponseWriter, request *http.Request) {
	var input registerRequest
	if err := decodeJSON(request, &input); err != nil {
		writeError(writer, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	registeredUser, err := handler.service.Register(request.Context(), user.RegisterInput{
		Email:    input.Email,
		Password: input.Password,
		Name:     input.Name,
	})
	if err != nil {
		switch {
		case errors.Is(err, user.ErrInvalidInput):
			writeError(writer, http.StatusBadRequest, "invalid_request", "invalid request")
		case errors.Is(err, user.ErrEmailAlreadyExists):
			writeError(writer, http.StatusConflict, "email_already_exists", "email is already registered")
		default:
			writeError(writer, http.StatusInternalServerError, "internal_error", "internal server error")
		}
		return
	}

	writeJSON(writer, http.StatusCreated, userResponse{
		ID:    registeredUser.ID,
		Email: registeredUser.Email,
		Name:  registeredUser.Name,
	})
}

func (handler *authenticationHandler) login(writer http.ResponseWriter, request *http.Request) {
	var input loginRequest
	if err := decodeJSON(request, &input); err != nil {
		writeError(writer, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	result, err := handler.service.Login(request.Context(), user.LoginInput{
		Email:    input.Email,
		Password: input.Password,
	})
	if err != nil {
		switch {
		case errors.Is(err, user.ErrInvalidInput):
			writeError(writer, http.StatusBadRequest, "invalid_request", "invalid request")
		case errors.Is(err, user.ErrInvalidCredentials):
			writeError(writer, http.StatusUnauthorized, "invalid_credentials", "invalid email or password")
		default:
			writeError(writer, http.StatusInternalServerError, "internal_error", "internal server error")
		}
		return
	}

	writeJSON(writer, http.StatusOK, loginResponse{
		AccessToken: result.AccessToken,
		TokenType:   "Bearer",
	})
}

func decodeJSON(request *http.Request, destination any) error {
	decoder := json.NewDecoder(io.LimitReader(request.Body, maxJSONBodySize))
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(destination); err != nil {
		return err
	}

	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("request body must contain one JSON value")
	}

	return nil
}
