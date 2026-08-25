package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/virtuxa/tz_task_manager/internal/task"
)

func TestCreateTask(t *testing.T) {
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/tasks",
		strings.NewReader(`{"team_id":10,"title":"Задача","description":"Описание","assignee_id":2}`),
	)
	request.Header.Set("Authorization", "Bearer valid-token")
	response := httptest.NewRecorder()

	newTestHandler().ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status code = %d, want %d", response.Code, http.StatusCreated)
	}
}

func TestListTasksRequiresTeamID(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/v1/tasks", nil)
	request.Header.Set("Authorization", "Bearer valid-token")
	response := httptest.NewRecorder()

	newTestHandler().ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status code = %d, want %d", response.Code, http.StatusBadRequest)
	}
}

type taskServiceStub struct{}

func (taskServiceStub) Create(_ context.Context, userID int64, input task.CreateInput) (task.Task, error) {
	if userID != 5 || input.TeamID != 10 || input.Title != "Задача" || input.AssigneeID == nil || *input.AssigneeID != 2 {
		return task.Task{}, task.ErrInvalidInput
	}

	now := time.Date(2026, time.August, 25, 12, 0, 0, 0, time.UTC)
	return task.Task{
		ID: 1, TeamID: input.TeamID, Title: input.Title, Description: input.Description,
		Status: task.StatusTodo, CreatedBy: userID, AssigneeID: input.AssigneeID,
		CreatedAt: now, UpdatedAt: now, Version: 1,
	}, nil
}

func (taskServiceStub) List(_ context.Context, _ int64, _ task.Filter) ([]task.Task, error) {
	return []task.Task{}, nil
}

func (taskServiceStub) Update(_ context.Context, _ int64, _ int64, _ task.UpdateInput) (task.Task, error) {
	return task.Task{}, nil
}

func (taskServiceStub) Delete(_ context.Context, _ int64, _ int64) error {
	return nil
}

func (taskServiceStub) History(_ context.Context, _ int64, _ int64) ([]task.History, error) {
	return []task.History{}, nil
}
