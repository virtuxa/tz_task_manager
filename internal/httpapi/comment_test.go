package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateComment(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/tasks/1/comments", strings.NewReader(`{"content":"Комментарий"}`))
	request.Header.Set("Authorization", "Bearer valid-token")
	response := httptest.NewRecorder()

	newTestHandler().ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status code = %d, want %d", response.Code, http.StatusCreated)
	}
}
