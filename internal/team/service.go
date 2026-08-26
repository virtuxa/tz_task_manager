package team

import (
	"context"
	"errors"
	"strings"

	"github.com/virtuxa/tz_task_manager/internal/user"
)

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
	// Создает команду вместе с участником-владельцем
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
	// Добавляет зарегистрированного пользователя с учетом роли инициатора
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
	// Меняет роль участника, если действие выполняет владелец
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
	// Не удаляет владельца и участника с незакрытыми назначенными задачами
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
	// Удаляет команду только по запросу владельца
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
