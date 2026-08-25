package stats

import (
	"context"
	"errors"

	"github.com/virtuxa/tz_task_manager/internal/team"
)

var (
	ErrInvalidInput = errors.New("invalid stats input")
	ErrForbidden    = errors.New("stats action is forbidden")
)

type StatusCount struct {
	Status string
	Count  int64
}

type AssigneeStat struct {
	UserID           int64
	Name             string
	ClosedTasksCount int64
}

type Stats struct {
	TeamID                      int64
	TasksByStatus               []StatusCount
	TopAssignees                []AssigneeStat
	AverageCloseDurationSeconds float64
	CommentsCount               int64
}

type Repository interface {
	Get(context.Context, int64) (Stats, error)
}

type MembershipReader interface {
	MemberRole(context.Context, int64, int64) (team.Role, error)
}

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
