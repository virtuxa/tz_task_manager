package httpapi

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestLoggingWritesStructuredRecord(t *testing.T) {
	var output bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&output, nil))
	handler := withRequestLogging(logger, http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusCreated)
		_, _ = writer.Write([]byte("created"))
	}))

	request := httptest.NewRequest(http.MethodPost, "/api/v1/tasks?team_id=10", nil)
	request.Header.Set("X-Request-ID", "request-123")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	var record map[string]any
	if err := json.Unmarshal(output.Bytes(), &record); err != nil {
		t.Fatalf("decode log record: %v", err)
	}

	if record["msg"] != "http request completed" || record["method"] != http.MethodPost || record["path"] != "/api/v1/tasks" {
		t.Fatalf("log record = %+v", record)
	}
	if record["status"] != float64(http.StatusCreated) || record["response_bytes"] != float64(len("created")) {
		t.Fatalf("response fields = %+v", record)
	}
	if record["request_id"] != "request-123" {
		t.Fatalf("request ID = %v", record["request_id"])
	}
}
