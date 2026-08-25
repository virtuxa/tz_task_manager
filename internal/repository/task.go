package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/virtuxa/tz_task_manager/internal/task"
)

type MySQLTaskRepository struct {
	database *sql.DB
}

func NewMySQLTaskRepository(database *sql.DB) *MySQLTaskRepository {
	return &MySQLTaskRepository{database: database}
}

func (repository *MySQLTaskRepository) CreateWithHistory(ctx context.Context, newTask task.Task, changedBy int64, changes json.RawMessage) (task.Task, error) {
	transaction, err := repository.database.BeginTx(ctx, nil)
	if err != nil {
		return task.Task{}, fmt.Errorf("begin create task: %w", err)
	}

	result, err := transaction.ExecContext(
		ctx,
		`INSERT INTO tasks (
			team_id, title, description, status, created_by, assignee_id, created_at, updated_at, closed_at, version
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		newTask.TeamID,
		newTask.Title,
		newTask.Description,
		newTask.Status,
		newTask.CreatedBy,
		newTask.AssigneeID,
		newTask.CreatedAt,
		newTask.UpdatedAt,
		newTask.ClosedAt,
		newTask.Version,
	)
	if err != nil {
		transaction.Rollback()
		return task.Task{}, fmt.Errorf("insert task: %w", err)
	}

	taskID, err := result.LastInsertId()
	if err != nil {
		transaction.Rollback()
		return task.Task{}, fmt.Errorf("read inserted task ID: %w", err)
	}

	if err := insertTaskHistory(ctx, transaction, taskID, changedBy, changes); err != nil {
		transaction.Rollback()
		return task.Task{}, err
	}

	if err := transaction.Commit(); err != nil {
		return task.Task{}, fmt.Errorf("commit create task: %w", err)
	}

	newTask.ID = taskID
	return newTask, nil
}

func (repository *MySQLTaskRepository) FindByID(ctx context.Context, taskID int64) (task.Task, error) {
	row := repository.database.QueryRowContext(
		ctx,
		`SELECT id, team_id, title, description, status, created_by, assignee_id,
			created_at, updated_at, closed_at, version
		 FROM tasks
		 WHERE id = ? AND deleted_at IS NULL`,
		taskID,
	)

	foundTask, err := scanTask(row)
	if errors.Is(err, sql.ErrNoRows) {
		return task.Task{}, task.ErrNotFound
	}
	if err != nil {
		return task.Task{}, fmt.Errorf("find task: %w", err)
	}

	return foundTask, nil
}

func (repository *MySQLTaskRepository) List(ctx context.Context, filter task.Filter) ([]task.Task, error) {
	query := strings.Builder{}
	query.WriteString(`SELECT id, team_id, title, description, status, created_by, assignee_id,
		created_at, updated_at, closed_at, version
		FROM tasks
		WHERE team_id = ? AND deleted_at IS NULL`)

	arguments := []any{filter.TeamID}
	if filter.Status != nil {
		query.WriteString(" AND status = ?")
		arguments = append(arguments, *filter.Status)
	}
	if filter.AssigneeID != nil {
		query.WriteString(" AND assignee_id = ?")
		arguments = append(arguments, *filter.AssigneeID)
	}

	query.WriteString(" ORDER BY updated_at DESC, id DESC LIMIT ? OFFSET ?")
	arguments = append(arguments, filter.Limit, filter.Offset)

	rows, err := repository.database.QueryContext(ctx, query.String(), arguments...)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	defer rows.Close()

	tasks := make([]task.Task, 0)
	for rows.Next() {
		item, err := scanTask(rows)
		if err != nil {
			return nil, fmt.Errorf("scan task: %w", err)
		}

		tasks = append(tasks, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tasks: %w", err)
	}

	return tasks, nil
}

func (repository *MySQLTaskRepository) UpdateWithHistory(ctx context.Context, updatedTask task.Task, changedBy int64, changes json.RawMessage) (task.Task, error) {
	transaction, err := repository.database.BeginTx(ctx, nil)
	if err != nil {
		return task.Task{}, fmt.Errorf("begin update task: %w", err)
	}

	result, err := transaction.ExecContext(
		ctx,
		`UPDATE tasks
		 SET title = ?, description = ?, status = ?, assignee_id = ?, closed_at = ?, updated_at = ?, version = version + 1
		 WHERE id = ? AND version = ? AND deleted_at IS NULL`,
		updatedTask.Title,
		updatedTask.Description,
		updatedTask.Status,
		updatedTask.AssigneeID,
		updatedTask.ClosedAt,
		updatedTask.UpdatedAt,
		updatedTask.ID,
		updatedTask.Version,
	)
	if err != nil {
		transaction.Rollback()
		return task.Task{}, fmt.Errorf("update task: %w", err)
	}

	updatedRows, err := result.RowsAffected()
	if err != nil {
		transaction.Rollback()
		return task.Task{}, fmt.Errorf("read updated task rows: %w", err)
	}
	if updatedRows == 0 {
		transaction.Rollback()
		return task.Task{}, task.ErrVersionConflict
	}

	if err := insertTaskHistory(ctx, transaction, updatedTask.ID, changedBy, changes); err != nil {
		transaction.Rollback()
		return task.Task{}, err
	}

	if err := transaction.Commit(); err != nil {
		return task.Task{}, fmt.Errorf("commit update task: %w", err)
	}

	updatedTask.Version++
	return updatedTask, nil
}

func (repository *MySQLTaskRepository) SoftDeleteWithHistory(ctx context.Context, taskID int64, version int64, changedBy int64, deletedAt time.Time, changes json.RawMessage) error {
	transaction, err := repository.database.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin delete task: %w", err)
	}

	result, err := transaction.ExecContext(
		ctx,
		`UPDATE tasks
		 SET deleted_at = ?, updated_at = ?, version = version + 1
		 WHERE id = ? AND version = ? AND deleted_at IS NULL`,
		deletedAt,
		deletedAt,
		taskID,
		version,
	)
	if err != nil {
		transaction.Rollback()
		return fmt.Errorf("delete task: %w", err)
	}

	updatedRows, err := result.RowsAffected()
	if err != nil {
		transaction.Rollback()
		return fmt.Errorf("read deleted task rows: %w", err)
	}
	if updatedRows == 0 {
		transaction.Rollback()
		return task.ErrVersionConflict
	}

	if err := insertTaskHistory(ctx, transaction, taskID, changedBy, changes); err != nil {
		transaction.Rollback()
		return err
	}

	if err := transaction.Commit(); err != nil {
		return fmt.Errorf("commit delete task: %w", err)
	}

	return nil
}

func (repository *MySQLTaskRepository) ListHistory(ctx context.Context, taskID int64) ([]task.History, error) {
	rows, err := repository.database.QueryContext(
		ctx,
		`SELECT id, task_id, changed_by, changes, created_at
		 FROM task_history
		 WHERE task_id = ?
		 ORDER BY created_at ASC, id ASC`,
		taskID,
	)
	if err != nil {
		return nil, fmt.Errorf("list task history: %w", err)
	}
	defer rows.Close()

	history := make([]task.History, 0)
	for rows.Next() {
		item := task.History{}
		var changes []byte
		if err := rows.Scan(&item.ID, &item.TaskID, &item.ChangedBy, &changes, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan task history: %w", err)
		}

		item.Changes = changes
		history = append(history, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate task history: %w", err)
	}

	return history, nil
}

func insertTaskHistory(ctx context.Context, transaction *sql.Tx, taskID int64, changedBy int64, changes json.RawMessage) error {
	if _, err := transaction.ExecContext(
		ctx,
		`INSERT INTO task_history (task_id, changed_by, changes) VALUES (?, ?, ?)`,
		taskID,
		changedBy,
		changes,
	); err != nil {
		return fmt.Errorf("insert task history: %w", err)
	}

	return nil
}

type rowScanner interface {
	Scan(...any) error
}

func scanTask(scanner rowScanner) (task.Task, error) {
	foundTask := task.Task{}
	var assigneeID sql.NullInt64
	var closedAt sql.NullTime

	err := scanner.Scan(
		&foundTask.ID,
		&foundTask.TeamID,
		&foundTask.Title,
		&foundTask.Description,
		&foundTask.Status,
		&foundTask.CreatedBy,
		&assigneeID,
		&foundTask.CreatedAt,
		&foundTask.UpdatedAt,
		&closedAt,
		&foundTask.Version,
	)
	if err != nil {
		return task.Task{}, err
	}

	if assigneeID.Valid {
		foundTask.AssigneeID = &assigneeID.Int64
	}
	if closedAt.Valid {
		foundTask.ClosedAt = &closedAt.Time
	}

	return foundTask, nil
}
