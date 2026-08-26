package stats

import (
	"context"
	"errors"

	"github.com/virtuxa/tz_task_manager/internal/team"
)

type Service struct {
	repository  Repository
	memberships MembershipReader
}

func NewService(repository Repository, memberships MembershipReader) (*Service, error) {
	if repository == nil {
		return nil, errors.New("stats repository is required")
	}
	if memberships == nil {
		return nil, errors.New("team membership reader is required")
	}

	return &Service{repository: repository, memberships: memberships}, nil
}

func (service *Service) Get(ctx context.Context, actorID int64, teamID int64) (Stats, error) {
	// Открывает отчет владельцу или администратору команды
	if actorID <= 0 || teamID <= 0 {
		return Stats{}, ErrInvalidInput
	}

	role, err := service.memberships.MemberRole(ctx, teamID, actorID)
	if errors.Is(err, team.ErrMemberNotFound) {
		return Stats{}, ErrForbidden
	}
	if err != nil {
		return Stats{}, err
	}

	if role != team.RoleOwner && role != team.RoleAdmin {
		return Stats{}, ErrForbidden
	}

	return service.repository.Get(ctx, teamID)
}
