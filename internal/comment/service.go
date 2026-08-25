package comment

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/virtuxa/tz_task_manager/internal/task"
	"github.com/virtuxa/tz_task_manager/internal/team"
)

const maxContentLength = 5000

var (
	ErrInvalidInput = errors.New("invalid comment input")
	ErrForbidden    = errors.New("comment action is forbidden")
	ErrNotFound     = errors.New("comment task not found")
)

type Comment struct {
	ID        int64
	TaskID    int64
	UserID    int64
	Content   string
	CreatedAt time.Time
}

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

type Service struct {
	repository  Repository
	tasks       TaskReader
	memberships MembershipReader
	now         func() time.Time
}

func NewService(repository Repository, tasks TaskReader, memberships MembershipReader) (*Service, error) {
	if repository == nil {
		return nil, errors.New("comment repository is required")
	}
	if tasks == nil {
		return nil, errors.New("task reader is required")
	}
	if memberships == nil {
		return nil, errors.New("team membership reader is required")
	}

	return &Service{
		repository:  repository,
		tasks:       tasks,
		memberships: memberships,
		now:         time.Now,
	}, nil
}

func (service *Service) Create(ctx context.Context, actorID int64, taskID int64, content string) (Comment, error) {
	if actorID <= 0 || taskID <= 0 {
		return Comment{}, ErrInvalidInput
	}

	content = strings.TrimSpace(content)
	if content == "" || len(content) > maxContentLength {
		return Comment{}, ErrInvalidInput
	}

	storedTask, err := service.accessibleTask(ctx, actorID, taskID)
	if err != nil {
		return Comment{}, err
	}

	return service.repository.Create(ctx, Comment{
		TaskID:    storedTask.ID,
		UserID:    actorID,
		Content:   content,
		CreatedAt: service.now().UTC(),
	})
}

func (service *Service) List(ctx context.Context, actorID int64, taskID int64) ([]Comment, error) {
	if actorID <= 0 || taskID <= 0 {
		return nil, ErrInvalidInput
	}

	if _, err := service.accessibleTask(ctx, actorID, taskID); err != nil {
		return nil, err
	}

	return service.repository.List(ctx, taskID)
}

func (service *Service) accessibleTask(ctx context.Context, actorID int64, taskID int64) (task.Task, error) {
	storedTask, err := service.tasks.FindByID(ctx, taskID)
	if errors.Is(err, task.ErrNotFound) {
		return task.Task{}, ErrNotFound
	}
	if err != nil {
		return task.Task{}, err
	}

	if _, err := service.memberships.MemberRole(ctx, storedTask.TeamID, actorID); errors.Is(err, team.ErrMemberNotFound) {
		return task.Task{}, ErrForbidden
	} else if err != nil {
		return task.Task{}, err
	}

	return storedTask, nil
}
