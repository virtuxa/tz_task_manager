package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/virtuxa/tz_task_manager/internal/auth"
	"github.com/virtuxa/tz_task_manager/internal/config"
	"github.com/virtuxa/tz_task_manager/internal/database"
	"github.com/virtuxa/tz_task_manager/internal/httpapi"
	"github.com/virtuxa/tz_task_manager/internal/migration"
	"github.com/virtuxa/tz_task_manager/internal/repository"
	"github.com/virtuxa/tz_task_manager/internal/user"
)

const (
	shutdownTimeout = 10 * time.Second
	startupTimeout  = 10 * time.Second
)

func main() {
	if err := run(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Printf("api stopped with error: %v", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	startupContext, cancelStartup := context.WithTimeout(context.Background(), startupTimeout)
	defer cancelStartup()

	mysqlDB, err := database.OpenMySQL(startupContext, cfg.MySQLDSN)
	if err != nil {
		return err
	}
	defer mysqlDB.Close()

	if err := migration.Apply(startupContext, mysqlDB); err != nil {
		return err
	}

	passwords, err := auth.NewPasswordManager(bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	tokens, err := auth.NewTokenManager(cfg.JWTSecret, cfg.JWTTTL)
	if err != nil {
		return err
	}

	users, err := user.NewService(repository.NewMySQLUserRepository(mysqlDB), passwords, tokens)
	if err != nil {
		return err
	}

	server := &http.Server{
		Addr:              cfg.HTTPAddress,
		Handler:           httpapi.NewHandler(users),
		ReadHeaderTimeout: 5 * time.Second,
	}

	stopContext, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serverErrors := make(chan error, 1)
	go func() {
		log.Printf("api listening on %s", cfg.HTTPAddress)
		serverErrors <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErrors:
		return err
	case <-stopContext.Done():
		shutdownContext, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		return server.Shutdown(shutdownContext)
	}
}
