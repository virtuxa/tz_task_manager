package httpapi

import (
	"context"

	"github.com/virtuxa/tz_task_manager/internal/comment"
	"github.com/virtuxa/tz_task_manager/internal/stats"
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

type StatsService interface {
	Get(context.Context, int64, int64) (stats.Stats, error)
}
