package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealth(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	response := httptest.NewRecorder()

	NewHandler().ServeHTTP(response, request)

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
