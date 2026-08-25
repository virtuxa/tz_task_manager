package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/go-sql-driver/mysql"
	"github.com/virtuxa/tz_task_manager/internal/user"
)

type MySQLUserRepository struct {
	database *sql.DB
}

func NewMySQLUserRepository(database *sql.DB) *MySQLUserRepository {
	return &MySQLUserRepository{database: database}
}

func (repository *MySQLUserRepository) Create(ctx context.Context, newUser user.User) (user.User, error) {
	result, err := repository.database.ExecContext(
		ctx,
		`INSERT INTO users (email, password_hash, name) VALUES (?, ?, ?)`,
		newUser.Email,
		newUser.PasswordHash,
		newUser.Name,
	)
	if err != nil {
		var mysqlError *mysql.MySQLError
		if errors.As(err, &mysqlError) && mysqlError.Number == 1062 {
			return user.User{}, user.ErrEmailAlreadyExists
		}

		return user.User{}, fmt.Errorf("insert user: %w", err)
	}

	userID, err := result.LastInsertId()
	if err != nil {
		return user.User{}, fmt.Errorf("read inserted user ID: %w", err)
	}

	newUser.ID = userID
	return newUser, nil
}

func (repository *MySQLUserRepository) FindByEmail(ctx context.Context, email string) (user.User, error) {
	registeredUser := user.User{}
	err := repository.database.QueryRowContext(
		ctx,
		`SELECT id, email, password_hash, name, created_at FROM users WHERE email = ?`,
		email,
	).Scan(
		&registeredUser.ID,
		&registeredUser.Email,
		&registeredUser.PasswordHash,
		&registeredUser.Name,
		&registeredUser.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return user.User{}, user.ErrNotFound
	}
	if err != nil {
		return user.User{}, fmt.Errorf("find user by email: %w", err)
	}

	return registeredUser, nil
}
