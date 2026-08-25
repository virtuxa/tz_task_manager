package stats

import (
	"context"
	"errors"
	"testing"

	"github.com/virtuxa/tz_task_manager/internal/team"
)

func TestServiceGetAllowsAdmin(t *testing.T) {
	service := newTestService(t, membershipReaderStub{role: team.RoleAdmin})

	result, err := service.Get(context.Background(), 1, 10)
	if err != nil {
		t.Fatalf("get stats: %v", err)
	}

	if result.TeamID != 10 {
		t.Fatalf("team ID = %d, want 10", result.TeamID)
	}
}

func TestServiceGetRejectsMember(t *testing.T) {
	service := newTestService(t, membershipReaderStub{role: team.RoleMember})

	_, err := service.Get(context.Background(), 1, 10)
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("get stats error = %v, want %v", err, ErrForbidden)
	}
}

func newTestService(t *testing.T, memberships MembershipReader) *Service {
	t.Helper()

	service, err := NewService(repositoryStub{}, memberships)
	if err != nil {
		t.Fatalf("create stats service: %v", err)
	}

	return service
}

type repositoryStub struct{}

func (repositoryStub) Get(_ context.Context, teamID int64) (Stats, error) {
	return Stats{TeamID: teamID}, nil
}

type membershipReaderStub struct {
	role team.Role
}

func (stub membershipReaderStub) MemberRole(_ context.Context, _ int64, _ int64) (team.Role, error) {
	return stub.role, nil
}
