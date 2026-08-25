package task

import (
	"context"
	"encoding/json"
	"time"

	"github.com/virtuxa/tz_task_manager/internal/team"
)

type Repository interface {
	CreateWithHistory(context.Context, Task, int64, json.RawMessage) (Task, error)
	FindByID(context.Context, int64) (Task, error)
	List(context.Context, Filter) ([]Task, error)
	UpdateWithHistory(context.Context, Task, int64, json.RawMessage) (Task, error)
	SoftDeleteWithHistory(context.Context, int64, int64, int64, time.Time, json.RawMessage) error
	ListHistory(context.Context, int64) ([]History, error)
}

type ListCache interface {
	Get(context.Context, Filter) ([]Task, string, bool, error)
	Set(context.Context, Filter, string, []Task) error
	InvalidateTeam(context.Context, int64) error
}

type MembershipReader interface {
	MemberRole(context.Context, int64, int64) (team.Role, error)
}
