package task

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/virtuxa/tz_task_manager/internal/team"
)

func TestServiceCreateWritesHistory(t *testing.T) {
	repository := &repositoryStub{}
	service := newTestService(t, repository, map[int64]team.Role{1: team.RoleMember})
	service.now = fixedClock

	created, err := service.Create(context.Background(), 1, CreateInput{
		TeamID:      10,
		Title:       " Задача ",
		Description: "Описание",
	})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}

	if created.Title != "Задача" || created.Status != StatusTodo || created.Version != 1 {
		t.Fatalf("created task = %+v", created)
	}

	var changes map[string]json.RawMessage
	if err := json.Unmarshal(repository.createdChanges, &changes); err != nil {
		t.Fatalf("decode creation history: %v", err)
	}
	if _, exists := changes["task"]; !exists {
		t.Fatal("creation history does not contain task snapshot")
	}
}

func TestServiceUpdateRejectsExecutorChangingTitle(t *testing.T) {
	assigneeID := int64(2)
	repository := &repositoryStub{task: Task{
		ID: 1, TeamID: 10, Title: "Исходная", Description: "Описание", Status: StatusTodo,
		CreatedBy: 1, AssigneeID: &assigneeID, Version: 1,
	}}
	service := newTestService(t, repository, map[int64]team.Role{2: team.RoleMember})
	title := "Новая"

	_, err := service.Update(context.Background(), 2, 1, UpdateInput{Version: 1, Title: &title})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("update error = %v, want %v", err, ErrForbidden)
	}
}

func TestServiceUpdateSetsClosedAtAndHistory(t *testing.T) {
	repository := &repositoryStub{task: Task{
		ID: 1, TeamID: 10, Title: "Задача", Description: "Описание", Status: StatusInProgress,
		CreatedBy: 1, Version: 3,
	}}
	service := newTestService(t, repository, map[int64]team.Role{1: team.RoleMember})
	service.now = fixedClock
	status := StatusDone

	updated, err := service.Update(context.Background(), 1, 1, UpdateInput{Version: 3, Status: &status})
	if err != nil {
		t.Fatalf("update task: %v", err)
	}

	if updated.Status != StatusDone || updated.ClosedAt == nil || !updated.ClosedAt.Equal(fixedClock()) {
		t.Fatalf("updated task = %+v", updated)
	}

	var changes map[string]json.RawMessage
	if err := json.Unmarshal(repository.updatedChanges, &changes); err != nil {
		t.Fatalf("decode update history: %v", err)
	}
	if _, exists := changes["closed_at"]; !exists {
		t.Fatal("update history does not contain closed_at")
	}
}

func TestServiceUpdateRejectsStaleVersion(t *testing.T) {
	repository := &repositoryStub{task: Task{ID: 1, TeamID: 10, CreatedBy: 1, Version: 2}}
	service := newTestService(t, repository, map[int64]team.Role{1: team.RoleMember})
	status := StatusDone

	_, err := service.Update(context.Background(), 1, 1, UpdateInput{Version: 1, Status: &status})
	if !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("update error = %v, want %v", err, ErrVersionConflict)
	}
}

func TestServiceDeleteAllowsCreator(t *testing.T) {
	repository := &repositoryStub{task: Task{ID: 1, TeamID: 10, CreatedBy: 1, Version: 2}}
	service := newTestService(t, repository, map[int64]team.Role{1: team.RoleMember})
	service.now = fixedClock

	if err := service.Delete(context.Background(), 1, 1); err != nil {
		t.Fatalf("delete task: %v", err)
	}

	if !repository.deletedAt.Equal(fixedClock()) {
		t.Fatalf("deleted at = %s, want %s", repository.deletedAt, fixedClock())
	}
}

func newTestService(t *testing.T, repository Repository, roles map[int64]team.Role) *Service {
	t.Helper()

	service, err := NewService(repository, membershipReaderStub{roles: roles}, listCacheStub{})
	if err != nil {
		t.Fatalf("create task service: %v", err)
	}

	return service
}

func fixedClock() time.Time {
	return time.Date(2026, time.August, 25, 12, 0, 0, 0, time.UTC)
}

type repositoryStub struct {
	task           Task
	createdChanges json.RawMessage
	updatedChanges json.RawMessage
	deletedAt      time.Time
}

func (stub *repositoryStub) CreateWithHistory(_ context.Context, task Task, _ int64, changes json.RawMessage) (Task, error) {
	stub.createdChanges = changes
	task.ID = 1
	return task, nil
}

func (stub *repositoryStub) FindByID(_ context.Context, _ int64) (Task, error) {
	return stub.task, nil
}

func (stub *repositoryStub) List(_ context.Context, _ Filter) ([]Task, error) {
	return nil, nil
}

func (stub *repositoryStub) UpdateWithHistory(_ context.Context, task Task, _ int64, changes json.RawMessage) (Task, error) {
	stub.updatedChanges = changes
	task.Version++
	return task, nil
}

func (stub *repositoryStub) SoftDeleteWithHistory(_ context.Context, _ int64, _ int64, _ int64, deletedAt time.Time, _ json.RawMessage) error {
	stub.deletedAt = deletedAt
	return nil
}

func (stub *repositoryStub) ListHistory(_ context.Context, _ int64) ([]History, error) {
	return nil, nil
}

type membershipReaderStub struct {
	roles map[int64]team.Role
}

type listCacheStub struct{}

func (listCacheStub) Get(_ context.Context, _ Filter) ([]Task, string, bool, error) {
	return nil, "0", false, nil
}

func (listCacheStub) Set(_ context.Context, _ Filter, _ string, _ []Task) error {
	return nil
}

func (listCacheStub) InvalidateTeam(_ context.Context, _ int64) error {
	return nil
}

func (stub membershipReaderStub) MemberRole(_ context.Context, _ int64, userID int64) (team.Role, error) {
	role, exists := stub.roles[userID]
	if !exists {
		return "", team.ErrMemberNotFound
	}

	return role, nil
}
