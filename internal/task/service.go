package task

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/virtuxa/tz_task_manager/internal/team"
)

const (
	defaultListLimit   = 20
	maxListLimit       = 100
	maxTitleLength     = 255
	maxDescriptionSize = 5000
)

var (
	ErrInvalidInput    = errors.New("invalid task input")
	ErrForbidden       = errors.New("task action is forbidden")
	ErrNotFound        = errors.New("task not found")
	ErrVersionConflict = errors.New("task version conflict")
)

type Status string

const (
	StatusTodo       Status = "todo"
	StatusInProgress Status = "in_progress"
	StatusDone       Status = "done"
)

type Task struct {
	ID          int64
	TeamID      int64
	Title       string
	Description string
	Status      Status
	CreatedBy   int64
	AssigneeID  *int64
	CreatedAt   time.Time
	UpdatedAt   time.Time
	ClosedAt    *time.Time
	Version     int64
}

type History struct {
	ID        int64
	TaskID    int64
	ChangedBy int64
	Changes   json.RawMessage
	CreatedAt time.Time
}

type CreateInput struct {
	TeamID      int64
	Title       string
	Description string
	Status      Status
	AssigneeID  *int64
}

type UpdateInput struct {
	Version         int64
	Title           *string
	Description     *string
	Status          *Status
	AssigneeID      *int64
	AssigneeIDIsSet bool
}

type Filter struct {
	TeamID     int64
	Status     *Status
	AssigneeID *int64
	Limit      int
	Offset     int
}

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

type Service struct {
	repository  Repository
	memberships MembershipReader
	cache       ListCache
	now         func() time.Time
}

func NewService(repository Repository, memberships MembershipReader, cache ListCache) (*Service, error) {
	if repository == nil {
		return nil, errors.New("task repository is required")
	}

	if memberships == nil {
		return nil, errors.New("team membership reader is required")
	}
	if cache == nil {
		return nil, errors.New("task list cache is required")
	}

	return &Service{
		repository:  repository,
		memberships: memberships,
		cache:       cache,
		now:         time.Now,
	}, nil
}

func (service *Service) Create(ctx context.Context, actorID int64, input CreateInput) (Task, error) {
	if actorID <= 0 || input.TeamID <= 0 {
		return Task{}, ErrInvalidInput
	}

	if _, err := service.memberRole(ctx, input.TeamID, actorID); err != nil {
		return Task{}, err
	}

	task, err := service.newTask(actorID, input)
	if err != nil {
		return Task{}, err
	}

	if err := service.validateAssignee(ctx, task.TeamID, task.AssigneeID); err != nil {
		return Task{}, err
	}

	changes, err := marshalChanges(map[string]FieldChange{
		"event": {
			Old: nil,
			New: "created",
		},
		"task": {
			Old: nil,
			New: snapshot(task),
		},
	})
	if err != nil {
		return Task{}, err
	}

	createdTask, err := service.repository.CreateWithHistory(ctx, task, actorID, changes)
	if err != nil {
		return Task{}, err
	}

	_ = service.cache.InvalidateTeam(ctx, createdTask.TeamID)
	return createdTask, nil
}

func (service *Service) List(ctx context.Context, actorID int64, filter Filter) ([]Task, error) {
	if actorID <= 0 || filter.TeamID <= 0 {
		return nil, ErrInvalidInput
	}

	if _, err := service.memberRole(ctx, filter.TeamID, actorID); err != nil {
		return nil, err
	}

	if filter.Status != nil && !isValidStatus(*filter.Status) {
		return nil, ErrInvalidInput
	}
	if filter.AssigneeID != nil && *filter.AssigneeID <= 0 {
		return nil, ErrInvalidInput
	}

	if filter.Limit == 0 {
		filter.Limit = defaultListLimit
	}
	if filter.Limit < 1 || filter.Limit > maxListLimit || filter.Offset < 0 {
		return nil, ErrInvalidInput
	}

	cachedTasks, cacheVersion, found, err := service.cache.Get(ctx, filter)
	if err == nil && found {
		return cachedTasks, nil
	}
	cacheAvailable := err == nil

	tasks, err := service.repository.List(ctx, filter)
	if err != nil {
		return nil, err
	}

	if cacheAvailable {
		_ = service.cache.Set(ctx, filter, cacheVersion, tasks)
	}

	return tasks, nil
}

func (service *Service) Update(ctx context.Context, actorID int64, taskID int64, input UpdateInput) (Task, error) {
	if actorID <= 0 || taskID <= 0 || input.Version <= 0 {
		return Task{}, ErrInvalidInput
	}

	current, err := service.repository.FindByID(ctx, taskID)
	if err != nil {
		return Task{}, err
	}

	role, err := service.memberRole(ctx, current.TeamID, actorID)
	if err != nil {
		return Task{}, err
	}

	if current.Version != input.Version {
		return Task{}, ErrVersionConflict
	}

	if !canUpdate(role, current, actorID, input) {
		return Task{}, ErrForbidden
	}

	updated, changes, err := service.applyUpdate(current, input)
	if err != nil {
		return Task{}, err
	}
	if len(changes) == 0 {
		return current, nil
	}

	if err := service.validateAssignee(ctx, updated.TeamID, updated.AssigneeID); err != nil {
		return Task{}, err
	}

	encodedChanges, err := marshalChanges(changes)
	if err != nil {
		return Task{}, err
	}

	updatedTask, err := service.repository.UpdateWithHistory(ctx, updated, actorID, encodedChanges)
	if err != nil {
		return Task{}, err
	}

	_ = service.cache.InvalidateTeam(ctx, updatedTask.TeamID)
	return updatedTask, nil
}

func (service *Service) Delete(ctx context.Context, actorID int64, taskID int64) error {
	if actorID <= 0 || taskID <= 0 {
		return ErrInvalidInput
	}

	current, err := service.repository.FindByID(ctx, taskID)
	if err != nil {
		return err
	}

	role, err := service.memberRole(ctx, current.TeamID, actorID)
	if err != nil {
		return err
	}

	if role != team.RoleOwner && role != team.RoleAdmin && current.CreatedBy != actorID {
		return ErrForbidden
	}

	deletedAt := service.now().UTC()
	changes, err := marshalChanges(map[string]FieldChange{
		"deleted_at": {
			Old: nil,
			New: deletedAt,
		},
		"event": {
			Old: nil,
			New: "deleted",
		},
	})
	if err != nil {
		return err
	}

	if err := service.repository.SoftDeleteWithHistory(ctx, current.ID, current.Version, actorID, deletedAt, changes); err != nil {
		return err
	}

	_ = service.cache.InvalidateTeam(ctx, current.TeamID)
	return nil
}

func (service *Service) History(ctx context.Context, actorID int64, taskID int64) ([]History, error) {
	if actorID <= 0 || taskID <= 0 {
		return nil, ErrInvalidInput
	}

	current, err := service.repository.FindByID(ctx, taskID)
	if err != nil {
		return nil, err
	}

	if _, err := service.memberRole(ctx, current.TeamID, actorID); err != nil {
		return nil, err
	}

	return service.repository.ListHistory(ctx, taskID)
}

func (service *Service) newTask(actorID int64, input CreateInput) (Task, error) {
	title := strings.TrimSpace(input.Title)
	if title == "" || len(title) > maxTitleLength || len(input.Description) > maxDescriptionSize {
		return Task{}, ErrInvalidInput
	}

	status := input.Status
	if status == "" {
		status = StatusTodo
	}
	if !isValidStatus(status) {
		return Task{}, ErrInvalidInput
	}
	if input.AssigneeID != nil && *input.AssigneeID <= 0 {
		return Task{}, ErrInvalidInput
	}

	now := service.now().UTC()
	task := Task{
		TeamID:      input.TeamID,
		Title:       title,
		Description: input.Description,
		Status:      status,
		CreatedBy:   actorID,
		AssigneeID:  input.AssigneeID,
		CreatedAt:   now,
		UpdatedAt:   now,
		Version:     1,
	}
	if status == StatusDone {
		task.ClosedAt = &now
	}

	return task, nil
}

func (service *Service) applyUpdate(current Task, input UpdateInput) (Task, map[string]FieldChange, error) {
	updated := current
	changes := make(map[string]FieldChange)

	if input.Title != nil {
		title := strings.TrimSpace(*input.Title)
		if title == "" || len(title) > maxTitleLength {
			return Task{}, nil, ErrInvalidInput
		}
		if title != current.Title {
			changes["title"] = FieldChange{Old: current.Title, New: title}
			updated.Title = title
		}
	}

	if input.Description != nil {
		if len(*input.Description) > maxDescriptionSize {
			return Task{}, nil, ErrInvalidInput
		}
		if *input.Description != current.Description {
			changes["description"] = FieldChange{Old: current.Description, New: *input.Description}
			updated.Description = *input.Description
		}
	}

	if input.Status != nil {
		if !isValidStatus(*input.Status) {
			return Task{}, nil, ErrInvalidInput
		}
		if *input.Status != current.Status {
			changes["status"] = FieldChange{Old: current.Status, New: *input.Status}
			updated.Status = *input.Status
		}
	}

	if input.AssigneeIDIsSet {
		if input.AssigneeID != nil && *input.AssigneeID <= 0 {
			return Task{}, nil, ErrInvalidInput
		}
		if !sameOptionalInt64(input.AssigneeID, current.AssigneeID) {
			changes["assignee_id"] = FieldChange{Old: current.AssigneeID, New: input.AssigneeID}
			updated.AssigneeID = input.AssigneeID
		}
	}

	if updated.Status == StatusDone && current.Status != StatusDone {
		closedAt := service.now().UTC()
		changes["closed_at"] = FieldChange{Old: current.ClosedAt, New: closedAt}
		updated.ClosedAt = &closedAt
	}
	if updated.Status != StatusDone && current.Status == StatusDone {
		changes["closed_at"] = FieldChange{Old: current.ClosedAt, New: nil}
		updated.ClosedAt = nil
	}

	if len(changes) > 0 {
		updated.UpdatedAt = service.now().UTC()
	}

	return updated, changes, nil
}

func (service *Service) validateAssignee(ctx context.Context, teamID int64, assigneeID *int64) error {
	if assigneeID == nil {
		return nil
	}

	if _, err := service.memberRole(ctx, teamID, *assigneeID); err != nil {
		if errors.Is(err, ErrForbidden) {
			return ErrInvalidInput
		}

		return err
	}

	return nil
}

func (service *Service) memberRole(ctx context.Context, teamID int64, userID int64) (team.Role, error) {
	role, err := service.memberships.MemberRole(ctx, teamID, userID)
	if errors.Is(err, team.ErrMemberNotFound) {
		return "", ErrForbidden
	}
	if err != nil {
		return "", err
	}

	return role, nil
}

func canUpdate(role team.Role, current Task, actorID int64, input UpdateInput) bool {
	if role == team.RoleOwner || role == team.RoleAdmin || current.CreatedBy == actorID {
		return true
	}

	if current.AssigneeID == nil || *current.AssigneeID != actorID {
		return false
	}

	return input.Status != nil && input.Title == nil && input.Description == nil && !input.AssigneeIDIsSet
}

func isValidStatus(status Status) bool {
	return status == StatusTodo || status == StatusInProgress || status == StatusDone
}

func marshalChanges(changes map[string]FieldChange) (json.RawMessage, error) {
	encodedChanges, err := json.Marshal(changes)
	if err != nil {
		return nil, fmt.Errorf("marshal task changes: %w", err)
	}

	return encodedChanges, nil
}

func sameOptionalInt64(left *int64, right *int64) bool {
	if left == nil || right == nil {
		return left == right
	}

	return *left == *right
}

type FieldChange struct {
	Old any `json:"old"`
	New any `json:"new"`
}

type taskSnapshot struct {
	TeamID      int64  `json:"team_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      Status `json:"status"`
	CreatedBy   int64  `json:"created_by"`
	AssigneeID  *int64 `json:"assignee_id"`
}

func snapshot(task Task) taskSnapshot {
	return taskSnapshot{
		TeamID:      task.TeamID,
		Title:       task.Title,
		Description: task.Description,
		Status:      task.Status,
		CreatedBy:   task.CreatedBy,
		AssigneeID:  task.AssigneeID,
	}
}
