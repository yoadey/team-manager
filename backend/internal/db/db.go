// Package db provides helpers for connecting to PostgreSQL and running schema
// migrations via goose.
package db

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

// Connect parses dsn, configures a pgxpool.Pool with sensible connection-pool
// settings, pings the database to confirm connectivity, and returns the pool.
//
// Pool configuration:
//   - MaxConns: 25
//   - MinConns: 2
//   - MaxConnLifetime: 1 h
//   - MaxConnIdleTime: 30 min
//
// The caller is responsible for closing the pool (defer pool.Close()).
func Connect(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	if dsn == "" {
		return nil, errors.New("db: dsn must not be empty")
	}

	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("db: parse dsn: %w", err)
	}

	cfg.MaxConns = 25
	cfg.MinConns = 2
	cfg.MaxConnLifetime = time.Hour
	cfg.MaxConnIdleTime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("db: create pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("db: ping database: %w", err)
	}

	return pool, nil
}

// RunMigrations runs all pending goose Up migrations from migrationsDir against
// the database represented by pool. Each applied migration is logged via slog.
//
// It uses pgx/v5's stdlib adapter so that goose can work with a *database/sql.DB
// while the rest of the application continues to use *pgxpool.Pool.
func RunMigrations(ctx context.Context, pool *pgxpool.Pool, migrationsDir string) error {
	if pool == nil {
		return errors.New("db: pool must not be nil")
	}
	if migrationsDir == "" {
		return errors.New("db: migrationsDir must not be empty")
	}

	sqlDB := stdlib.OpenDBFromPool(pool)
	defer func() { _ = sqlDB.Close() }()

	fsys := os.DirFS(migrationsDir)

	provider, err := goose.NewProvider(goose.DialectPostgres, sqlDB, fsys)
	if err != nil {
		return fmt.Errorf("db: create goose provider: %w", err)
	}

	results, err := provider.Up(ctx)
	for _, r := range results {
		if r.Error != nil {
			slog.ErrorContext(
				ctx, "migration failed",
				slog.String("file", r.Source.Path),
				slog.Any("error", r.Error),
			)
			continue
		}
		slog.InfoContext(
			ctx, "migration applied",
			slog.String("file", r.Source.Path),
			slog.Duration("duration", r.Duration),
		)
	}
	if err != nil {
		return fmt.Errorf("db: run migrations: %w", err)
	}

	return nil
}
