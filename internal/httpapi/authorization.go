package httpapi

import (
	"context"
	"net/http"
	"strings"
)

type userIDContextKey struct{}

func requireAuthentication(tokens TokenParser) func(http.Handler) http.Handler {
	// Извлекает пользователя из Bearer-токена и передает его обработчику
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			rawToken, ok := bearerToken(request.Header.Get("Authorization"))
			if !ok {
				writeError(writer, http.StatusUnauthorized, "unauthorized", "authorization token is required")
				return
			}

			userID, err := tokens.Parse(rawToken)
			if err != nil {
				writeError(writer, http.StatusUnauthorized, "unauthorized", "authorization token is invalid")
				return
			}

			requestContext := context.WithValue(request.Context(), userIDContextKey{}, userID)
			next.ServeHTTP(writer, request.WithContext(requestContext))
		})
	}
}

func bearerToken(value string) (string, bool) {
	parts := strings.Fields(value)
	if len(parts) != 2 || parts[0] != "Bearer" || parts[1] == "" {
		return "", false
	}

	return parts[1], true
}

func userIDFromContext(context context.Context) (int64, bool) {
	userID, ok := context.Value(userIDContextKey{}).(int64)
	return userID, ok && userID > 0
}
