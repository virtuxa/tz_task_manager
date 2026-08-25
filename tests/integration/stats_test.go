package integration

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/virtuxa/tz_task_manager/internal/database"
	"github.com/virtuxa/tz_task_manager/internal/migration"
	"github.com/virtuxa/tz_task_manager/internal/repository"
)

func TestMySQLStatsRepository(t *testing.T) {
	dsn := os.Getenv("MYSQL_DSN")
	if dsn == "" {
		t.Skip("MYSQL_DSN is not set")
	}

	ctx := context.Background()
	mysqlDB, err := database.OpenMySQL(ctx, dsn)
	if err != nil {
		t.Fatalf("open MySQL: %v", err)
	}
	t.Cleanup(func() {
		if err := mysqlDB.Close(); err != nil {
			t.Errorf("close MySQL: %v", err)
		}
	})

	if err := migration.Apply(ctx, mysqlDB); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	fixture := insertStatsFixture(t, ctx, mysqlDB)
	t.Cleanup(func() {
		if fixture.teamID != 0 {
			if _, err := mysqlDB.ExecContext(ctx, `DELETE FROM teams WHERE id = ?`, fixture.teamID); err != nil {
				t.Errorf("delete test team: %v", err)
			}
		}
		if _, err := mysqlDB.ExecContext(ctx, `DELETE FROM users WHERE id IN (?, ?)`, fixture.ownerID, fixture.memberID); err != nil {
			t.Errorf("delete test users: %v", err)
		}
	})

	stats, err := repository.NewMySQLStatsRepository(mysqlDB).Get(ctx, fixture.teamID)
	if err != nil {
		t.Fatalf("get team stats: %v", err)
	}

	if len(stats.TasksByStatus) != 3 {
		t.Fatalf("status counts length = %d, want 3", len(stats.TasksByStatus))
	}
	if stats.TasksByStatus[0].Status != "todo" || stats.TasksByStatus[0].Count != 1 {
		t.Fatalf("todo count = %+v, want todo:1", stats.TasksByStatus[0])
	}
	if stats.TasksByStatus[1].Status != "in_progress" || stats.TasksByStatus[1].Count != 1 {
		t.Fatalf("in progress count = %+v, want in_progress:1", stats.TasksByStatus[1])
	}
	if stats.TasksByStatus[2].Status != "done" || stats.TasksByStatus[2].Count != 3 {
		t.Fatalf("done count = %+v, want done:3", stats.TasksByStatus[2])
	}

	if len(stats.TopAssignees) != 1 || stats.TopAssignees[0].UserID != fixture.memberID || stats.TopAssignees[0].ClosedTasksCount != 2 {
		t.Fatalf("top assignees = %+v", stats.TopAssignees)
	}
	if stats.AverageCloseDurationSeconds != 8400 {
		t.Fatalf("average close duration = %f, want 8400", stats.AverageCloseDurationSeconds)
	}
	if stats.CommentsCount != 2 {
		t.Fatalf("comments count = %d, want 2", stats.CommentsCount)
	}
}

type statsFixture struct {
	teamID   int64
	ownerID  int64
	memberID int64
}

func insertStatsFixture(t *testing.T, ctx context.Context, database *sql.DB) statsFixture {
	t.Helper()

	uniqueSuffix := time.Now().UTC().UnixNano()
	ownerID := insertUser(t, ctx, database, fmt.Sprintf("stats-owner-%d@example.com", uniqueSuffix), "Owner")
	memberID := insertUser(t, ctx, database, fmt.Sprintf("stats-member-%d@example.com", uniqueSuffix), "Member")
	teamID := insertTeam(t, ctx, database, ownerID)
	insertMember(t, ctx, database, teamID, ownerID, "owner")
	insertMember(t, ctx, database, teamID, memberID, "member")

	now := time.Now().UTC().Truncate(time.Second)
	insertTask(t, ctx, database, teamID, ownerID, nil, "todo", now.Add(-time.Hour), nil)
	insertTask(t, ctx, database, teamID, ownerID, nil, "in_progress", now.Add(-time.Hour), nil)
	doneTaskOne := insertTask(t, ctx, database, teamID, ownerID, &memberID, "done", now.Add(-2*time.Hour), timePointer(now.Add(-time.Hour)))
	doneTaskTwo := insertTask(t, ctx, database, teamID, ownerID, &memberID, "done", now.Add(-4*time.Hour), timePointer(now.Add(-time.Hour)))
	oldClosedAt := now.Add(-31 * 24 * time.Hour)
	insertTask(t, ctx, database, teamID, ownerID, &ownerID, "done", oldClosedAt.Add(-3*time.Hour), timePointer(oldClosedAt))
	insertComment(t, ctx, database, doneTaskOne, memberID)
	insertComment(t, ctx, database, doneTaskTwo, ownerID)

	return statsFixture{teamID: teamID, ownerID: ownerID, memberID: memberID}
}

func insertUser(t *testing.T, ctx context.Context, database *sql.DB, email string, name string) int64 {
	t.Helper()

	result, err := database.ExecContext(ctx, `INSERT INTO users (email, password_hash, name) VALUES (?, ?, ?)`, email, "hash", name)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("read user ID: %v", err)
	}

	return id
}

func insertTeam(t *testing.T, ctx context.Context, database *sql.DB, ownerID int64) int64 {
	t.Helper()

	result, err := database.ExecContext(ctx, `INSERT INTO teams (name, created_by) VALUES (?, ?)`, "Stats Team", ownerID)
	if err != nil {
		t.Fatalf("insert team: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("read team ID: %v", err)
	}

	return id
}

func insertMember(t *testing.T, ctx context.Context, database *sql.DB, teamID int64, userID int64, role string) {
	t.Helper()

	if _, err := database.ExecContext(ctx, `INSERT INTO team_members (team_id, user_id, role) VALUES (?, ?, ?)`, teamID, userID, role); err != nil {
		t.Fatalf("insert team member: %v", err)
	}
}

func insertTask(t *testing.T, ctx context.Context, database *sql.DB, teamID int64, createdBy int64, assigneeID *int64, status string, createdAt time.Time, closedAt *time.Time) int64 {
	t.Helper()

	result, err := database.ExecContext(
		ctx,
		`INSERT INTO tasks (team_id, title, description, status, created_by, assignee_id, created_at, updated_at, closed_at, version)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 1)`,
		teamID,
		"Stats Task",
		"Description",
		status,
		createdBy,
		assigneeID,
		createdAt,
		createdAt,
		closedAt,
	)
	if err != nil {
		t.Fatalf("insert task: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("read task ID: %v", err)
	}

	return id
}

func insertComment(t *testing.T, ctx context.Context, database *sql.DB, taskID int64, userID int64) {
	t.Helper()

	if _, err := database.ExecContext(ctx, `INSERT INTO task_comments (task_id, user_id, content) VALUES (?, ?, ?)`, taskID, userID, "Comment"); err != nil {
		t.Fatalf("insert task comment: %v", err)
	}
}

func timePointer(value time.Time) *time.Time {
	return &value
}
