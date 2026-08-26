package httpapi

import (
	"log/slog"
	"net/http"
)

func NewHandler(authentication AuthenticationService, teams TeamService, tasks TaskService, comments CommentService, statistics StatsService, tokens TokenParser) http.Handler {
	// Регистрирует маршруты и защищает приватные операции JWT
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", health)

	authenticationHandler := newAuthenticationHandler(authentication)
	mux.HandleFunc("POST /api/v1/register", authenticationHandler.register)
	mux.HandleFunc("POST /api/v1/login", authenticationHandler.login)

	teamHandler := newTeamHandler(teams)
	protected := requireAuthentication(tokens)
	mux.Handle("POST /api/v1/teams", protected(http.HandlerFunc(teamHandler.create)))
	mux.Handle("GET /api/v1/teams", protected(http.HandlerFunc(teamHandler.list)))
	mux.Handle("POST /api/v1/teams/{teamID}/invite", protected(http.HandlerFunc(teamHandler.invite)))
	mux.Handle("PATCH /api/v1/teams/{teamID}/members/{userID}", protected(http.HandlerFunc(teamHandler.changeRole)))
	mux.Handle("DELETE /api/v1/teams/{teamID}/members/{userID}", protected(http.HandlerFunc(teamHandler.removeMember)))
	mux.Handle("DELETE /api/v1/teams/{teamID}", protected(http.HandlerFunc(teamHandler.delete)))

	statsHandler := newStatsHandler(statistics)
	mux.Handle("GET /api/v1/teams/{teamID}/stats", protected(http.HandlerFunc(statsHandler.get)))

	taskHandler := newTaskHandler(tasks)
	mux.Handle("POST /api/v1/tasks", protected(http.HandlerFunc(taskHandler.create)))
	mux.Handle("GET /api/v1/tasks", protected(http.HandlerFunc(taskHandler.list)))
	mux.Handle("PUT /api/v1/tasks/{taskID}", protected(http.HandlerFunc(taskHandler.update)))
	mux.Handle("DELETE /api/v1/tasks/{taskID}", protected(http.HandlerFunc(taskHandler.delete)))
	mux.Handle("GET /api/v1/tasks/{taskID}/history", protected(http.HandlerFunc(taskHandler.history)))

	commentHandler := newCommentHandler(comments)
	mux.Handle("POST /api/v1/tasks/{taskID}/comments", protected(http.HandlerFunc(commentHandler.create)))
	mux.Handle("GET /api/v1/tasks/{taskID}/comments", protected(http.HandlerFunc(commentHandler.list)))

	return withRequestLogging(slog.Default(), mux)
}

func health(writer http.ResponseWriter, _ *http.Request) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write([]byte(`{"status":"ok"}`))
}
