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

	"github.com/virtuxa/tz_task_manager/internal/config"
	"github.com/virtuxa/tz_task_manager/internal/httpapi"
)

const shutdownTimeout = 10 * time.Second

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

	server := &http.Server{
		Addr:              cfg.HTTPAddress,
		Handler:           httpapi.NewHandler(),
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
