package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/virtuxa/tz_task_manager/internal/team"
)

func TestTeamsRequireBearerToken(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/v1/teams", nil)
	response := httptest.NewRecorder()

	newTestHandler().ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status code = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}

func TestCreateTeam(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/teams", strings.NewReader(`{"name":"Команда"}`))
	request.Header.Set("Authorization", "Bearer valid-token")
	response := httptest.NewRecorder()

	newTestHandler().ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status code = %d, want %d", response.Code, http.StatusCreated)
	}

	if body := response.Body.String(); body != `{"id":10,"name":"Команда","created_by":5,"role":"owner"}`+"\n" {
		t.Fatalf("body = %q", body)
	}
}

func TestCreateTeamRejectsInvalidToken(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/teams", strings.NewReader(`{"name":"Команда"}`))
	request.Header.Set("Authorization", "Bearer invalid-token")
	response := httptest.NewRecorder()

	newTestHandler().ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status code = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}

func newTestHandler() http.Handler {
	return NewHandler(authenticationServiceStub{}, teamServiceStub{}, taskServiceStub{}, tokenParserStub{})
}

type teamServiceStub struct{}

func (teamServiceStub) Create(_ context.Context, userID int64, name string) (team.Team, error) {
	if userID != 5 || name != "Команда" {
		return team.Team{}, errors.New("unexpected team input")
	}

	return team.Team{ID: 10, Name: name, CreatedBy: userID, Role: team.RoleOwner}, nil
}

func (teamServiceStub) List(_ context.Context, _ int64) ([]team.Team, error) {
	return []team.Team{}, nil
}

func (teamServiceStub) Invite(_ context.Context, _ int64, _ int64, _ string, _ team.Role) error {
	return nil
}

func (teamServiceStub) ChangeRole(_ context.Context, _ int64, _ int64, _ int64, _ team.Role) error {
	return nil
}

func (teamServiceStub) RemoveMember(_ context.Context, _ int64, _ int64, _ int64) error {
	return nil
}

func (teamServiceStub) Delete(_ context.Context, _ int64, _ int64) error {
	return nil
}

type tokenParserStub struct{}

func (tokenParserStub) Parse(rawToken string) (int64, error) {
	if rawToken != "valid-token" {
		return 0, errors.New("invalid token")
	}

	return 5, nil
}
