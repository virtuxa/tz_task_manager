package httpapi

import (
	"log/slog"
	"net/http"
	"time"
)

func withRequestLogging(logger *slog.Logger, next http.Handler) http.Handler {
	// Записывает итог каждого HTTP-запроса в структурированном виде
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		startedAt := time.Now()
		recorder := &responseRecorder{ResponseWriter: writer}

		next.ServeHTTP(recorder, request)

		status := recorder.status
		if status == 0 {
			status = http.StatusOK
		}

		attributes := []any{
			"method", request.Method,
			"path", request.URL.Path,
			"status", status,
			"duration_ms", time.Since(startedAt).Milliseconds(),
			"response_bytes", recorder.bytes,
		}
		if requestID := request.Header.Get("X-Request-ID"); requestID != "" {
			attributes = append(attributes, "request_id", requestID)
		}

		logger.InfoContext(request.Context(), "http request completed", attributes...)
	})
}

// responseRecorder сохраняет данные ответа для записи после обработчика
type responseRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (recorder *responseRecorder) WriteHeader(status int) {
	if recorder.status != 0 {
		return
	}

	recorder.status = status
	recorder.ResponseWriter.WriteHeader(status)
}

func (recorder *responseRecorder) Write(data []byte) (int, error) {
	if recorder.status == 0 {
		recorder.WriteHeader(http.StatusOK)
	}

	written, err := recorder.ResponseWriter.Write(data)
	recorder.bytes += written

	return written, err
}

func (recorder *responseRecorder) Unwrap() http.ResponseWriter {
	return recorder.ResponseWriter
}
