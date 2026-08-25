package httpapi

import "net/http"

func NewHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", health)

	return mux
}

func health(writer http.ResponseWriter, _ *http.Request) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write([]byte(`{"status":"ok"}`))
}
