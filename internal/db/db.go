// Package db opens the Postgres pool and runs the embedded migrations.
package db

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5" // registers pgx5://
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/brennanMKE/WoodDust/migrations"
)

// Open connects a pool and pings it.
func Open(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse DATABASE_URL: %w", err)
	}
	cfg.MaxConns = 10
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	pctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(pctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return pool, nil
}

func newMigrate(databaseURL string) (*migrate.Migrate, error) {
	src, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return nil, err
	}
	u, err := url.Parse(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse DATABASE_URL: %w", err)
	}
	switch strings.ToLower(u.Scheme) {
	case "postgres", "postgresql":
		u.Scheme = "pgx5"
	case "pgx5":
	default:
		return nil, fmt.Errorf("DATABASE_URL must be a postgres:// URL")
	}
	return migrate.NewWithSourceInstance("iofs", src, u.String())
}

// MigrateUp applies every pending migration.
func MigrateUp(databaseURL string) error {
	m, err := newMigrate(databaseURL)
	if err != nil {
		return err
	}
	defer m.Close()
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}

// MigrateDown rolls back steps migrations (all of them when steps <= 0).
func MigrateDown(databaseURL string, steps int) error {
	m, err := newMigrate(databaseURL)
	if err != nil {
		return err
	}
	defer m.Close()
	if steps <= 0 {
		err = m.Down()
	} else {
		err = m.Steps(-steps)
	}
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}

// Version reports the current migration version.
func Version(databaseURL string) (uint, bool, error) {
	m, err := newMigrate(databaseURL)
	if err != nil {
		return 0, false, err
	}
	defer m.Close()
	v, dirty, err := m.Version()
	if errors.Is(err, migrate.ErrNilVersion) {
		return 0, false, nil
	}
	return v, dirty, err
}
