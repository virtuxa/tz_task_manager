package team

import (
	"context"
	"errors"
	"testing"

	"github.com/virtuxa/tz_task_manager/internal/user"
)

func TestServiceCreateNormalizesName(t *testing.T) {
	repository := &repositoryStub{}
	service := newTestService(t, repository)

	created, err := service.Create(context.Background(), 1, " Команда ")
	if err != nil {
		t.Fatalf("create team: %v", err)
	}

	if created.Name != "Команда" {
		t.Fatalf("team name = %q, want %q", created.Name, "Команда")
	}
}

func TestServiceInviteRestrictsAdminRole(t *testing.T) {
	repository := &repositoryStub{roles: map[int64]Role{1: RoleAdmin}}
	service := newTestService(t, repository)

	err := service.Invite(context.Background(), 1, 10, "member@example.com", RoleAdmin)
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("invite error = %v, want %v", err, ErrForbidden)
	}
}

func TestServiceRemoveMemberBlocksOpenTasks(t *testing.T) {
	repository := &repositoryStub{
		roles:        map[int64]Role{1: RoleOwner, 2: RoleMember},
		hasOpenTasks: true,
	}
	service := newTestService(t, repository)

	err := service.RemoveMember(context.Background(), 1, 10, 2)
	if !errors.Is(err, ErrOpenAssignedTasks) {
		t.Fatalf("remove member error = %v, want %v", err, ErrOpenAssignedTasks)
	}
}

func TestServiceRemoveMemberAllowsSelfRemoval(t *testing.T) {
	repository := &repositoryStub{roles: map[int64]Role{2: RoleMember}}
	service := newTestService(t, repository)

	if err := service.RemoveMember(context.Background(), 2, 10, 2); err != nil {
		t.Fatalf("remove own membership: %v", err)
	}

	if repository.removedUserID != 2 {
		t.Fatalf("removed user ID = %d, want 2", repository.removedUserID)
	}
}

func TestServiceChangeRoleRejectsOwner(t *testing.T) {
	repository := &repositoryStub{roles: map[int64]Role{1: RoleOwner, 2: RoleOwner}}
	service := newTestService(t, repository)

	err := service.ChangeRole(context.Background(), 1, 10, 2, RoleAdmin)
	if !errors.Is(err, ErrOwnerMembershipChange) {
		t.Fatalf("change role error = %v, want %v", err, ErrOwnerMembershipChange)
	}
}

func newTestService(t *testing.T, repository Repository) *Service {
	t.Helper()

	service, err := NewService(repository, userFinderStub{})
	if err != nil {
		t.Fatalf("create team service: %v", err)
	}

	return service
}

type repositoryStub struct {
	roles         map[int64]Role
	hasOpenTasks  bool
	removedUserID int64
}

func (stub *repositoryStub) CreateWithOwner(_ context.Context, name string, ownerID int64) (Team, error) {
	return Team{ID: 10, Name: name, CreatedBy: ownerID, Role: RoleOwner}, nil
}

func (stub *repositoryStub) ListByUser(_ context.Context, _ int64) ([]Team, error) {
	return nil, nil
}

func (stub *repositoryStub) MemberRole(_ context.Context, _ int64, userID int64) (Role, error) {
	role, exists := stub.roles[userID]
	if !exists {
		return "", ErrMemberNotFound
	}

	return role, nil
}

func (stub *repositoryStub) AddMember(_ context.Context, _ int64, _ int64, _ Role) error {
	return nil
}

func (stub *repositoryStub) ChangeMemberRole(_ context.Context, _ int64, _ int64, _ Role) error {
	return nil
}

func (stub *repositoryStub) RemoveMember(_ context.Context, _ int64, userID int64) error {
	stub.removedUserID = userID
	return nil
}

func (stub *repositoryStub) HasOpenAssignedTasks(_ context.Context, _ int64, _ int64) (bool, error) {
	return stub.hasOpenTasks, nil
}

func (stub *repositoryStub) Delete(_ context.Context, _ int64) error {
	return nil
}

type userFinderStub struct{}

func (userFinderStub) FindByEmail(_ context.Context, email string) (user.User, error) {
	if email == "member@example.com" {
		return user.User{ID: 2, Email: email}, nil
	}

	return user.User{}, user.ErrNotFound
}
