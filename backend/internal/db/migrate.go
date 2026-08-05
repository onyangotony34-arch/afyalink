package db

import (
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	// Registers the "pgx5" database driver with golang-migrate.
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

// Migrate applies all pending up migrations from the embedded migration set.
//
// It is safe to call on every boot: golang-migrate takes an advisory lock, so
// concurrent instances starting at once will not race, and an already-current
// database is a no-op.
func Migrate(databaseURL string) error {
	source, err := iofs.New(Migrations, "migrations")
	if err != nil {
		return fmt.Errorf("db: opening embedded migrations: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", source, normalizeForMigrate(databaseURL))
	if err != nil {
		return fmt.Errorf("db: initialising migrator: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("db: applying migrations: %w", err)
	}
	return nil
}

// normalizeForMigrate rewrites a postgres:// URL to the pgx/v5 driver scheme
// that golang-migrate registers, so callers can use one DATABASE_URL for both
// the pool and the migrator.
func normalizeForMigrate(databaseURL string) string {
	for _, prefix := range []string{"postgres://", "postgresql://"} {
		if len(databaseURL) >= len(prefix) && databaseURL[:len(prefix)] == prefix {
			return "pgx5://" + databaseURL[len(prefix):]
		}
	}
	return databaseURL
}
