// Package db owns the pgx connection pool and migration runner.
//
// Every query in this codebase goes through pgx with bind parameters. There is
// no string-concatenated SQL anywhere in the tree, and no query builder that
// could reintroduce it.
package db

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Migrations is embedded so a built binary carries its own schema and a deploy
// cannot drift from the migration files that were checked in with it.
//
//go:embed all:migrations
var Migrations embed.FS

// Pool is the shared connection pool type used across the app.
type Pool = pgxpool.Pool

// Connect opens the pool and verifies connectivity before returning, so a bad
// DATABASE_URL surfaces at startup instead of on the first request.
func Connect(ctx context.Context, databaseURL string) (*Pool, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("db: parsing DATABASE_URL: %w", err)
	}

	cfg.MaxConns = 10
	cfg.MinConns = 2
	cfg.MaxConnLifetime = time.Hour
	cfg.MaxConnIdleTime = 30 * time.Minute
	cfg.HealthCheckPeriod = time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("db: creating pool: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("db: pinging database: %w", err)
	}

	return pool, nil
}

// ErrNotFound is the package-level sentinel handlers translate into a 404 (or,
// for patient-scoped resources, a deliberately indistinguishable 403).
var ErrNotFound = errors.New("resource not found")
