// Command api serves the AfyaLink HTTP API.
package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/OderoCeasar/afyalink/backend/internal/api"
	"github.com/OderoCeasar/afyalink/backend/internal/config"
	"github.com/OderoCeasar/afyalink/backend/internal/crypto"
	"github.com/OderoCeasar/afyalink/backend/internal/db"
	"github.com/OderoCeasar/afyalink/backend/internal/logging"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	logger := logging.New(os.Stdout, cfg.Env)

	// Applied at boot so a deploy can never serve against a schema older than
	// the binary expects.
	if err := db.Migrate(cfg.DatabaseURL); err != nil {
		return err
	}
	logger.Info("migrations applied")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	cipher, err := crypto.New(cfg.PHIKey)
	if err != nil {
		return err
	}

	engine, err := api.New(api.Deps{
		Config: cfg,
		Pool:   pool,
		Logger: logger,
		Cipher: cipher,
	})
	if err != nil {
		return err
	}

	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: engine,

		// Timeouts are set explicitly because Go's zero value is "no timeout",
		// which lets a slow client hold a connection open indefinitely.
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       2 * time.Minute,
	}

	serverErr := make(chan error, 1)
	go func() {
		logger.Info("api listening", "port", cfg.Port, "env", cfg.Env)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	select {
	case err := <-serverErr:
		return err
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	}

	// Give in-flight requests a chance to finish rather than cutting a
	// clinician off mid-write.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown failed: %w", err)
	}

	logger.Info("api stopped")
	return nil
}
