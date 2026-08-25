package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/virtuxa/tz_task_manager/internal/stats"
)

func TestGetStats(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/v1/teams/10/stats", nil)
	request.Header.Set("Authorization", "Bearer valid-token")
	response := httptest.NewRecorder()

	newTestHandler().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", response.Code, http.StatusOK)
	}

	want := `{"team_id":10,"tasks_by_status":[{"status":"todo","count":2}],"top_assignees":[{"user_id":7,"name":"Анна","closed_tasks_count":3}],"average_close_duration_seconds":3600,"comments_count":4}` + "\n"
	if body := response.Body.String(); body != want {
		t.Fatalf("body = %q, want %q", body, want)
	}
}

func TestGetStatsRejectsMember(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/v1/teams/10/stats", nil)
	request.Header.Set("Authorization", "Bearer valid-token")
	response := httptest.NewRecorder()

	NewHandler(
		authenticationServiceStub{},
		teamServiceStub{},
		taskServiceStub{},
		commentServiceStub{},
		statsServiceStub{getError: stats.ErrForbidden},
		tokenParserStub{},
	).ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("status code = %d, want %d", response.Code, http.StatusForbidden)
	}
}

type statsServiceStub struct {
	getError error
}

func (stub statsServiceStub) Get(_ context.Context, userID int64, teamID int64) (stats.Stats, error) {
	if stub.getError != nil {
		return stats.Stats{}, stub.getError
	}
	if userID != 5 || teamID != 10 {
		return stats.Stats{}, errors.New("unexpected stats input")
	}

	return stats.Stats{
		TeamID: 10,
		TasksByStatus: []stats.StatusCount{
			{Status: "todo", Count: 2},
		},
		TopAssignees: []stats.AssigneeStat{
			{UserID: 7, Name: "Анна", ClosedTasksCount: 3},
		},
		AverageCloseDurationSeconds: 3600,
		CommentsCount:               4,
	}, nil
}
