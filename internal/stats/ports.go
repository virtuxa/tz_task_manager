package stats

import (
	"context"

	"github.com/virtuxa/tz_task_manager/internal/team"
)

type Repository interface {
	Get(context.Context, int64) (Stats, error)
}

type MembershipReader interface {
	MemberRole(context.Context, int64, int64) (team.Role, error)
}
