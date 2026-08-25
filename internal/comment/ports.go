package comment

import (
	"context"

	"github.com/virtuxa/tz_task_manager/internal/task"
	"github.com/virtuxa/tz_task_manager/internal/team"
)

type Repository interface {
	Create(context.Context, Comment) (Comment, error)
	List(context.Context, int64) ([]Comment, error)
}

type TaskReader interface {
	FindByID(context.Context, int64) (task.Task, error)
}

type MembershipReader interface {
	MemberRole(context.Context, int64, int64) (team.Role, error)
}
