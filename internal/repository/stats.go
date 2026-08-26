package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/virtuxa/tz_task_manager/internal/stats"
)

type MySQLStatsRepository struct {
	database *sql.DB
}

func NewMySQLStatsRepository(database *sql.DB) *MySQLStatsRepository {
	return &MySQLStatsRepository{database: database}
}

func (repository *MySQLStatsRepository) Get(ctx context.Context, teamID int64) (stats.Stats, error) {
	// Один CTE-запрос собирает все метрики только для указанной команды
	rows, err := repository.database.QueryContext(ctx, `
		WITH
		team_tasks AS (
			SELECT id, status, assignee_id, created_at, closed_at
			FROM tasks
			WHERE team_id = ? AND deleted_at IS NULL
		),
		statuses AS (
			SELECT _utf8mb4'todo' COLLATE utf8mb4_0900_ai_ci AS status
			UNION ALL SELECT _utf8mb4'in_progress' COLLATE utf8mb4_0900_ai_ci
			UNION ALL SELECT _utf8mb4'done' COLLATE utf8mb4_0900_ai_ci
		),
		status_counts AS (
			SELECT statuses.status, COUNT(team_tasks.id) AS task_count
			FROM statuses
			LEFT JOIN team_tasks ON team_tasks.status = statuses.status
			GROUP BY statuses.status
		),
		top_assignees AS (
			SELECT team_tasks.assignee_id AS user_id, users.name, COUNT(*) AS closed_tasks_count
			FROM team_tasks
			JOIN users ON users.id = team_tasks.assignee_id
			WHERE team_tasks.status = 'done'
				AND team_tasks.closed_at >= UTC_TIMESTAMP() - INTERVAL 30 DAY
			GROUP BY team_tasks.assignee_id, users.name
			ORDER BY closed_tasks_count DESC, user_id ASC
			LIMIT 3
		),
		close_metrics AS (
			SELECT COALESCE(AVG(TIMESTAMPDIFF(SECOND, created_at, closed_at)), 0) AS average_close_duration_seconds
			FROM team_tasks
			WHERE status = 'done' AND closed_at IS NOT NULL
		),
		comment_metrics AS (
			SELECT COUNT(*) AS comments_count
			FROM task_comments
			JOIN team_tasks ON team_tasks.id = task_comments.task_id
		)
		SELECT
			1 AS row_type,
			status_counts.status,
			status_counts.task_count,
			CAST(NULL AS SIGNED) AS user_id,
			CAST(NULL AS CHAR CHARACTER SET utf8mb4) COLLATE utf8mb4_0900_ai_ci AS user_name,
			CAST(NULL AS SIGNED) AS closed_tasks_count,
			CAST(NULL AS DECIMAL(20, 6)) AS average_close_duration_seconds,
			CAST(NULL AS SIGNED) AS comments_count
		FROM status_counts
		UNION ALL
		SELECT
			2 AS row_type,
			CAST(NULL AS CHAR CHARACTER SET utf8mb4) COLLATE utf8mb4_0900_ai_ci,
			CAST(NULL AS SIGNED),
			top_assignees.user_id,
			top_assignees.name,
			top_assignees.closed_tasks_count,
			CAST(NULL AS DECIMAL(20, 6)),
			CAST(NULL AS SIGNED)
		FROM top_assignees
		UNION ALL
		SELECT
			3 AS row_type,
			CAST(NULL AS CHAR CHARACTER SET utf8mb4) COLLATE utf8mb4_0900_ai_ci,
			CAST(NULL AS SIGNED),
			CAST(NULL AS SIGNED),
			CAST(NULL AS CHAR CHARACTER SET utf8mb4) COLLATE utf8mb4_0900_ai_ci,
			CAST(NULL AS SIGNED),
			close_metrics.average_close_duration_seconds,
			comment_metrics.comments_count
		FROM close_metrics
		CROSS JOIN comment_metrics
	`, teamID)
	if err != nil {
		return stats.Stats{}, fmt.Errorf("query team stats: %w", err)
	}
	defer rows.Close()

	result := stats.Stats{
		TeamID:        teamID,
		TasksByStatus: make([]stats.StatusCount, 0, 3),
		TopAssignees:  make([]stats.AssigneeStat, 0, 3),
	}
	statusCounts := make(map[string]int64, 3)
	for rows.Next() {
		var rowType int
		var status sql.NullString
		var statusCount sql.NullInt64
		var userID sql.NullInt64
		var userName sql.NullString
		var closedTasksCount sql.NullInt64
		var averageCloseDuration sql.NullFloat64
		var commentsCount sql.NullInt64
		if err := rows.Scan(
			&rowType,
			&status,
			&statusCount,
			&userID,
			&userName,
			&closedTasksCount,
			&averageCloseDuration,
			&commentsCount,
		); err != nil {
			return stats.Stats{}, fmt.Errorf("scan team stats: %w", err)
		}

		switch rowType {
		case 1:
			statusCounts[status.String] = statusCount.Int64
		case 2:
			result.TopAssignees = append(result.TopAssignees, stats.AssigneeStat{
				UserID:           userID.Int64,
				Name:             userName.String,
				ClosedTasksCount: closedTasksCount.Int64,
			})
		case 3:
			result.AverageCloseDurationSeconds = averageCloseDuration.Float64
			result.CommentsCount = commentsCount.Int64
		}
	}

	if err := rows.Err(); err != nil {
		return stats.Stats{}, fmt.Errorf("iterate team stats: %w", err)
	}

	for _, status := range []string{"todo", "in_progress", "done"} {
		result.TasksByStatus = append(result.TasksByStatus, stats.StatusCount{
			Status: status,
			Count:  statusCounts[status],
		})
	}

	return result, nil
}
