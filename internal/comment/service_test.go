package comment

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/virtuxa/tz_task_manager/internal/task"
	"github.com/virtuxa/tz_task_manager/internal/team"
)

func TestServiceCreateTrimsContent(t *testing.T) {
	repository := &repositoryStub{}
	service := newTestService(t, repository, membershipReaderStub{role: team.RoleMember})
	service.now = fixedClock

	created, err := service.Create(context.Background(), 1, 10, " Комментарий ")
	if err != nil {
		t.Fatalf("create comment: %v", err)
	}

	if created.Content != "Комментарий" {
		t.Fatalf("content = %q, want %q", created.Content, "Комментарий")
	}

	if !created.CreatedAt.Equal(fixedClock()) {
		t.Fatalf("created at = %s, want %s", created.CreatedAt, fixedClock())
	}
}

func TestServiceListRejectsNonMember(t *testing.T) {
	service := newTestService(t, &repositoryStub{}, membershipReaderStub{error: team.ErrMemberNotFound})

	_, err := service.List(context.Background(), 2, 10)
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("list comments error = %v, want %v", err, ErrForbidden)
	}
}

func newTestService(t *testing.T, repository Repository, memberships MembershipReader) *Service {
	t.Helper()

	service, err := NewService(repository, taskReaderStub{}, memberships)
	if err != nil {
		t.Fatalf("create comment service: %v", err)
	}

	return service
}

func fixedClock() time.Time {
	return time.Date(2026, time.August, 25, 12, 0, 0, 0, time.UTC)
}

type repositoryStub struct{}

func (repositoryStub) Create(_ context.Context, comment Comment) (Comment, error) {
	comment.ID = 1
	return comment, nil
}

func (repositoryStub) List(_ context.Context, _ int64) ([]Comment, error) {
	return []Comment{}, nil
}

type taskReaderStub struct{}

func (taskReaderStub) FindByID(_ context.Context, taskID int64) (task.Task, error) {
	return task.Task{ID: taskID, TeamID: 1}, nil
}

type membershipReaderStub struct {
	role  team.Role
	error error
}

func (stub membershipReaderStub) MemberRole(_ context.Context, _ int64, _ int64) (team.Role, error) {
	return stub.role, stub.error
}
