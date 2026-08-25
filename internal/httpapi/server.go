package httpapi

import (
	"context"
	"net/http"

	"github.com/virtuxa/tz_task_manager/internal/comment"
	"github.com/virtuxa/tz_task_manager/internal/task"
	"github.com/virtuxa/tz_task_manager/internal/team"
	"github.com/virtuxa/tz_task_manager/internal/user"
)

type AuthenticationService interface {
	Register(context.Context, user.RegisterInput) (user.User, error)
	Login(context.Context, user.LoginInput) (user.LoginResult, error)
}

type TeamService interface {
	Create(context.Context, int64, string) (team.Team, error)
	List(context.Context, int64) ([]team.Team, error)
	Invite(context.Context, int64, int64, string, team.Role) error
	ChangeRole(context.Context, int64, int64, int64, team.Role) error
	RemoveMember(context.Context, int64, int64, int64) error
	Delete(context.Context, int64, int64) error
}

type TokenParser interface {
	Parse(string) (int64, error)
}

type TaskService interface {
	Create(context.Context, int64, task.CreateInput) (task.Task, error)
	List(context.Context, int64, task.Filter) ([]task.Task, error)
	Update(context.Context, int64, int64, task.UpdateInput) (task.Task, error)
	Delete(context.Context, int64, int64) error
	History(context.Context, int64, int64) ([]task.History, error)
}

type CommentService interface {
	Create(context.Context, int64, int64, string) (comment.Comment, error)
	List(context.Context, int64, int64) ([]comment.Comment, error)
}

func NewHandler(authentication AuthenticationService, teams TeamService, tasks TaskService, comments CommentService, tokens TokenParser) http.Handler {
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

	taskHandler := newTaskHandler(tasks)
	mux.Handle("POST /api/v1/tasks", protected(http.HandlerFunc(taskHandler.create)))
	mux.Handle("GET /api/v1/tasks", protected(http.HandlerFunc(taskHandler.list)))
	mux.Handle("PUT /api/v1/tasks/{taskID}", protected(http.HandlerFunc(taskHandler.update)))
	mux.Handle("DELETE /api/v1/tasks/{taskID}", protected(http.HandlerFunc(taskHandler.delete)))
	mux.Handle("GET /api/v1/tasks/{taskID}/history", protected(http.HandlerFunc(taskHandler.history)))

	commentHandler := newCommentHandler(comments)
	mux.Handle("POST /api/v1/tasks/{taskID}/comments", protected(http.HandlerFunc(commentHandler.create)))
	mux.Handle("GET /api/v1/tasks/{taskID}/comments", protected(http.HandlerFunc(commentHandler.list)))

	return mux
}

func health(writer http.ResponseWriter, _ *http.Request) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write([]byte(`{"status":"ok"}`))
}
