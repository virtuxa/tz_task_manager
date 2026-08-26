package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/go-sql-driver/mysql"
	"github.com/virtuxa/tz_task_manager/internal/team"
)

type MySQLTeamRepository struct {
	database *sql.DB
}

func NewMySQLTeamRepository(database *sql.DB) *MySQLTeamRepository {
	return &MySQLTeamRepository{database: database}
}

func (repository *MySQLTeamRepository) CreateWithOwner(ctx context.Context, name string, ownerID int64) (team.Team, error) {
	// Создает команду и первого участника-владельца одной транзакцией
	transaction, err := repository.database.BeginTx(ctx, nil)
	if err != nil {
		return team.Team{}, fmt.Errorf("begin create team: %w", err)
	}

	result, err := transaction.ExecContext(
		ctx,
		`INSERT INTO teams (name, created_by) VALUES (?, ?)`,
		name,
		ownerID,
	)
	if err != nil {
		transaction.Rollback()
		return team.Team{}, fmt.Errorf("insert team: %w", err)
	}

	teamID, err := result.LastInsertId()
	if err != nil {
		transaction.Rollback()
		return team.Team{}, fmt.Errorf("read inserted team ID: %w", err)
	}

	if _, err := transaction.ExecContext(
		ctx,
		`INSERT INTO team_members (team_id, user_id, role) VALUES (?, ?, ?)`,
		teamID,
		ownerID,
		team.RoleOwner,
	); err != nil {
		transaction.Rollback()
		return team.Team{}, fmt.Errorf("add team owner: %w", err)
	}

	if err := transaction.Commit(); err != nil {
		return team.Team{}, fmt.Errorf("commit create team: %w", err)
	}

	return team.Team{
		ID:        teamID,
		Name:      name,
		CreatedBy: ownerID,
		Role:      team.RoleOwner,
	}, nil
}

func (repository *MySQLTeamRepository) ListByUser(ctx context.Context, userID int64) ([]team.Team, error) {
	rows, err := repository.database.QueryContext(
		ctx,
		`SELECT t.id, t.name, t.created_by, t.created_at, tm.role
		 FROM teams t
		 JOIN team_members tm ON tm.team_id = t.id
		 WHERE tm.user_id = ?
		 ORDER BY t.created_at ASC, t.id ASC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list teams: %w", err)
	}
	defer rows.Close()

	teams := make([]team.Team, 0)
	for rows.Next() {
		item := team.Team{}
		if err := rows.Scan(&item.ID, &item.Name, &item.CreatedBy, &item.CreatedAt, &item.Role); err != nil {
			return nil, fmt.Errorf("scan team: %w", err)
		}

		teams = append(teams, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate teams: %w", err)
	}

	return teams, nil
}

func (repository *MySQLTeamRepository) MemberRole(ctx context.Context, teamID int64, userID int64) (team.Role, error) {
	var role team.Role
	err := repository.database.QueryRowContext(
		ctx,
		`SELECT role FROM team_members WHERE team_id = ? AND user_id = ?`,
		teamID,
		userID,
	).Scan(&role)
	if errors.Is(err, sql.ErrNoRows) {
		return "", team.ErrMemberNotFound
	}
	if err != nil {
		return "", fmt.Errorf("find member role: %w", err)
	}

	return role, nil
}

func (repository *MySQLTeamRepository) AddMember(ctx context.Context, teamID int64, userID int64, role team.Role) error {
	_, err := repository.database.ExecContext(
		ctx,
		`INSERT INTO team_members (team_id, user_id, role) VALUES (?, ?, ?)`,
		teamID,
		userID,
		role,
	)
	if err != nil {
		var mysqlError *mysql.MySQLError
		if errors.As(err, &mysqlError) && mysqlError.Number == 1062 {
			return team.ErrAlreadyMember
		}

		return fmt.Errorf("add team member: %w", err)
	}

	return nil
}

func (repository *MySQLTeamRepository) ChangeMemberRole(ctx context.Context, teamID int64, userID int64, role team.Role) error {
	result, err := repository.database.ExecContext(
		ctx,
		`UPDATE team_members SET role = ? WHERE team_id = ? AND user_id = ? AND role <> ?`,
		role,
		teamID,
		userID,
		team.RoleOwner,
	)
	if err != nil {
		return fmt.Errorf("change member role: %w", err)
	}

	updatedRows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read changed member rows: %w", err)
	}
	if updatedRows == 0 {
		return team.ErrMemberNotFound
	}

	return nil
}

func (repository *MySQLTeamRepository) RemoveMember(ctx context.Context, teamID int64, userID int64) error {
	result, err := repository.database.ExecContext(
		ctx,
		`DELETE FROM team_members WHERE team_id = ? AND user_id = ? AND role <> ?`,
		teamID,
		userID,
		team.RoleOwner,
	)
	if err != nil {
		return fmt.Errorf("remove team member: %w", err)
	}

	deletedRows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read removed member rows: %w", err)
	}
	if deletedRows == 0 {
		return team.ErrMemberNotFound
	}

	return nil
}

func (repository *MySQLTeamRepository) HasOpenAssignedTasks(ctx context.Context, teamID int64, userID int64) (bool, error) {
	var exists bool
	if err := repository.database.QueryRowContext(
		ctx,
		`SELECT EXISTS(
			SELECT 1 FROM tasks
			WHERE team_id = ? AND assignee_id = ? AND status <> 'done' AND deleted_at IS NULL
		)`,
		teamID,
		userID,
	).Scan(&exists); err != nil {
		return false, fmt.Errorf("check member open tasks: %w", err)
	}

	return exists, nil
}

func (repository *MySQLTeamRepository) Delete(ctx context.Context, teamID int64) error {
	result, err := repository.database.ExecContext(ctx, `DELETE FROM teams WHERE id = ?`, teamID)
	if err != nil {
		return fmt.Errorf("delete team: %w", err)
	}

	deletedRows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read deleted team rows: %w", err)
	}
	if deletedRows == 0 {
		return team.ErrNotFound
	}

	return nil
}
