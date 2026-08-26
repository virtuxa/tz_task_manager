package migration

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strconv"
	"strings"
)

//go:embed *.up.sql
var migrationFiles embed.FS

type migration struct {
	version int
	name    string
	content string
}

func Apply(ctx context.Context, database *sql.DB) error {
	// Применяет только миграции, которых нет в schema_migrations
	if _, err := database.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version BIGINT NOT NULL PRIMARY KEY,
			applied_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6)
		) ENGINE=InnoDB
	`); err != nil {
		return fmt.Errorf("create schema migrations table: %w", err)
	}

	migrations, err := loadMigrations()
	if err != nil {
		return err
	}

	for _, migration := range migrations {
		if err := apply(ctx, database, migration); err != nil {
			return err
		}
	}

	return nil
}

func loadMigrations() ([]migration, error) {
	entries, err := fs.ReadDir(migrationFiles, ".")
	if err != nil {
		return nil, fmt.Errorf("read migration files: %w", err)
	}

	migrations := make([]migration, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".up.sql") {
			continue
		}

		migration, err := parseMigration(entry.Name())
		if err != nil {
			return nil, err
		}

		content, err := migrationFiles.ReadFile(entry.Name())
		if err != nil {
			return nil, fmt.Errorf("read migration %s: %w", entry.Name(), err)
		}

		migration.content = string(content)
		migrations = append(migrations, migration)
	}

	sort.Slice(migrations, func(left int, right int) bool {
		return migrations[left].version < migrations[right].version
	})

	return migrations, nil
}

func parseMigration(name string) (migration, error) {
	parts := strings.SplitN(name, "_", 2)
	if len(parts) != 2 {
		return migration{}, fmt.Errorf("migration %s must start with a numeric version", name)
	}

	version, err := strconv.Atoi(parts[0])
	if err != nil || version <= 0 {
		return migration{}, fmt.Errorf("migration %s has invalid version", name)
	}

	return migration{version: version, name: name}, nil
}

func apply(ctx context.Context, database *sql.DB, migration migration) error {
	var exists bool
	if err := database.QueryRowContext(
		ctx,
		"SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = ?)",
		migration.version,
	).Scan(&exists); err != nil {
		return fmt.Errorf("check migration %s: %w", migration.name, err)
	}

	if exists {
		return nil
	}

	transaction, err := database.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin migration %s: %w", migration.name, err)
	}

	if _, err := transaction.ExecContext(ctx, migration.content); err != nil {
		transaction.Rollback()
		return fmt.Errorf("apply migration %s: %w", migration.name, err)
	}

	if _, err := transaction.ExecContext(
		ctx,
		"INSERT INTO schema_migrations (version) VALUES (?)",
		migration.version,
	); err != nil {
		transaction.Rollback()
		return fmt.Errorf("record migration %s: %w", migration.name, err)
	}

	if err := transaction.Commit(); err != nil {
		return fmt.Errorf("commit migration %s: %w", migration.name, err)
	}

	return nil
}
