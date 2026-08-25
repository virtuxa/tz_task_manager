package team

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/virtuxa/tz_task_manager/internal/user"
)

const maxNameLength = 100

var (
	ErrInvalidInput          = errors.New("invalid team input")
	ErrForbidden             = errors.New("team action is forbidden")
	ErrNotFound              = errors.New("team not found")
	ErrMemberNotFound        = errors.New("team member not found")
	ErrAlreadyMember         = errors.New("user is already a team member")
	ErrOpenAssignedTasks     = errors.New("member has open assigned tasks")
	ErrOwnerMembershipChange = errors.New("owner membership cannot be changed")
)

type Role string

const (
	RoleOwner  Role = "owner"
	RoleAdmin  Role = "admin"
	RoleMember Role = "member"
)

type Team struct {
	ID        int64
	Name      string
	CreatedBy int64
	CreatedAt time.Time
	Role      Role
}

type Member struct {
	UserID int64
	Role   Role
}

type Repository interface {
	CreateWithOwner(context.Context, string, int64) (Team, error)
	ListByUser(context.Context, int64) ([]Team, error)
	MemberRole(context.Context, int64, int64) (Role, error)
	AddMember(context.Context, int64, int64, Role) error
	ChangeMemberRole(context.Context, int64, int64, Role) error
	RemoveMember(context.Context, int64, int64) error
	HasOpenAssignedTasks(context.Context, int64, int64) (bool, error)
	Delete(context.Context, int64) error
}

type UserFinder interface {
	FindByEmail(context.Context, string) (user.User, error)
}

type Service struct {
	repository Repository
	users      UserFinder
}

func NewService(repository Repository, users UserFinder) (*Service, error) {
	if repository == nil {
		return nil, errors.New("team repository is required")
	}

	if users == nil {
		return nil, errors.New("user finder is required")
	}

	return &Service{repository: repository, users: users}, nil
}

func (service *Service) Create(ctx context.Context, ownerID int64, name string) (Team, error) {
	if ownerID <= 0 {
		return Team{}, ErrInvalidInput
	}

	name = strings.TrimSpace(name)
	if name == "" || len(name) > maxNameLength {
		return Team{}, ErrInvalidInput
	}

	return service.repository.CreateWithOwner(ctx, name, ownerID)
}

func (service *Service) List(ctx context.Context, userID int64) ([]Team, error) {
	if userID <= 0 {
		return nil, ErrInvalidInput
	}

	return service.repository.ListByUser(ctx, userID)
}

func (service *Service) Invite(ctx context.Context, actorID int64, teamID int64, email string, role Role) error {
	actorRole, err := service.actorRole(ctx, teamID, actorID)
	if err != nil {
		return err
	}

	if role != RoleAdmin && role != RoleMember {
		return ErrInvalidInput
	}

	if !canInvite(actorRole, role) {
		return ErrForbidden
	}

	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return ErrInvalidInput
	}

	invitedUser, err := service.users.FindByEmail(ctx, email)
	if errors.Is(err, user.ErrNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}

	if err := service.repository.AddMember(ctx, teamID, invitedUser.ID, role); err != nil {
		return err
	}

	return nil
}

func (service *Service) ChangeRole(ctx context.Context, actorID int64, teamID int64, memberID int64, role Role) error {
	if memberID <= 0 {
		return ErrInvalidInput
	}

	actorRole, err := service.actorRole(ctx, teamID, actorID)
	if err != nil {
		return err
	}

	if actorRole != RoleOwner {
		return ErrForbidden
	}

	if role != RoleAdmin && role != RoleMember {
		return ErrInvalidInput
	}

	memberRole, err := service.repository.MemberRole(ctx, teamID, memberID)
	if errors.Is(err, ErrMemberNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}

	if memberRole == RoleOwner {
		return ErrOwnerMembershipChange
	}
	if memberRole == role {
		return nil
	}

	return service.repository.ChangeMemberRole(ctx, teamID, memberID, role)
}

func (service *Service) RemoveMember(ctx context.Context, actorID int64, teamID int64, memberID int64) error {
	if memberID <= 0 {
		return ErrInvalidInput
	}

	actorRole, err := service.actorRole(ctx, teamID, actorID)
	if err != nil {
		return err
	}

	memberRole, err := service.repository.MemberRole(ctx, teamID, memberID)
	if errors.Is(err, ErrMemberNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}

	if memberRole == RoleOwner {
		return ErrOwnerMembershipChange
	}

	if !canRemoveMember(actorRole, actorID, memberID, memberRole) {
		return ErrForbidden
	}

	hasOpenTasks, err := service.repository.HasOpenAssignedTasks(ctx, teamID, memberID)
	if err != nil {
		return err
	}
	if hasOpenTasks {
		return ErrOpenAssignedTasks
	}

	return service.repository.RemoveMember(ctx, teamID, memberID)
}

func (service *Service) Delete(ctx context.Context, actorID int64, teamID int64) error {
	actorRole, err := service.actorRole(ctx, teamID, actorID)
	if err != nil {
		return err
	}

	if actorRole != RoleOwner {
		return ErrForbidden
	}

	return service.repository.Delete(ctx, teamID)
}

func (service *Service) actorRole(ctx context.Context, teamID int64, userID int64) (Role, error) {
	if teamID <= 0 || userID <= 0 {
		return "", ErrInvalidInput
	}

	role, err := service.repository.MemberRole(ctx, teamID, userID)
	if errors.Is(err, ErrMemberNotFound) {
		return "", ErrForbidden
	}
	if err != nil {
		return "", err
	}

	return role, nil
}

func canInvite(actorRole Role, invitedRole Role) bool {
	if actorRole == RoleOwner {
		return invitedRole == RoleAdmin || invitedRole == RoleMember
	}

	return actorRole == RoleAdmin && invitedRole == RoleMember
}

func canRemoveMember(actorRole Role, actorID int64, memberID int64, memberRole Role) bool {
	if actorID == memberID {
		return actorRole == RoleAdmin || actorRole == RoleMember
	}

	if actorRole == RoleOwner {
		return memberRole == RoleAdmin || memberRole == RoleMember
	}

	return actorRole == RoleAdmin && memberRole == RoleMember
}
