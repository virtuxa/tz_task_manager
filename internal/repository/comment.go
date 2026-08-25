package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/virtuxa/tz_task_manager/internal/comment"
)

type MySQLCommentRepository struct {
	database *sql.DB
}

func NewMySQLCommentRepository(database *sql.DB) *MySQLCommentRepository {
	return &MySQLCommentRepository{database: database}
}

func (repository *MySQLCommentRepository) Create(ctx context.Context, newComment comment.Comment) (comment.Comment, error) {
	result, err := repository.database.ExecContext(
		ctx,
		`INSERT INTO task_comments (task_id, user_id, content, created_at) VALUES (?, ?, ?, ?)`,
		newComment.TaskID,
		newComment.UserID,
		newComment.Content,
		newComment.CreatedAt,
	)
	if err != nil {
		return comment.Comment{}, fmt.Errorf("insert task comment: %w", err)
	}

	commentID, err := result.LastInsertId()
	if err != nil {
		return comment.Comment{}, fmt.Errorf("read inserted comment ID: %w", err)
	}

	newComment.ID = commentID
	return newComment, nil
}

func (repository *MySQLCommentRepository) List(ctx context.Context, taskID int64) ([]comment.Comment, error) {
	rows, err := repository.database.QueryContext(
		ctx,
		`SELECT id, task_id, user_id, content, created_at
		 FROM task_comments
		 WHERE task_id = ?
		 ORDER BY created_at ASC, id ASC`,
		taskID,
	)
	if err != nil {
		return nil, fmt.Errorf("list task comments: %w", err)
	}
	defer rows.Close()

	comments := make([]comment.Comment, 0)
	for rows.Next() {
		item := comment.Comment{}
		if err := rows.Scan(&item.ID, &item.TaskID, &item.UserID, &item.Content, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan task comment: %w", err)
		}

		comments = append(comments, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate task comments: %w", err)
	}

	return comments, nil
}
