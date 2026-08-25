package team

import (
	"context"

	"github.com/virtuxa/tz_task_manager/internal/user"
)

type Repository interface {
	CreateWithOwner(context.Context, string, int64) (Team, error)
	ListByUser(context.Context, int64) ([]Team, error)
	MemberRole(context.Context, int64, int64) (Role, error)
	AddMember(context.Context, int64, int64, Role) error
	ChangeMemberRole(context.Context, int64, int64, Role) error
	RemoveMember(context.Context, int64, int64) error
	HasOpenAssignedTasks(context.Context, int64, int64) (bool, error)
	Delete(context.Context, int64) error
}

type UserFinder interface {
	FindByEmail(context.Context, string) (user.User, error)
}
