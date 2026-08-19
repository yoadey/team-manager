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
	"github.com/pressly/goose/v3/lock"
)

// Sentinel errors for the db package.
var (
	ErrDSNEmpty           = errors.New("db: dsn must not be empty")
	ErrPoolNil            = errors.New("db: pool must not be nil")
	ErrMigrationsDirEmpty = errors.New("db: migrationsDir must not be empty")
)

// PoolConfig holds the pgxpool connection-pool sizing knobs. It is normally
// populated from config.Config's DBPoolMaxConns / DBPoolMinConns /
// DBPoolMaxConnLifetime / DBPoolMaxConnIdleTime fields (see
// config.loadDBPoolConfig), which in turn come from the DB_POOL_MAX_CONNS /
// DB_POOL_MIN_CONNS / DB_POOL_MAX_CONN_LIFETIME_MINUTES /
// DB_POOL_MAX_CONN_IDLE_TIME_MINUTES env vars -- letting an operator tune
// per-replica pool size against a fixed DB max_connections budget without a
// code change and redeploy.
type PoolConfig struct {
	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
}

// Connect parses dsn, configures a pgxpool.Pool using poolCfg, pings the
// database to confirm connectivity, and returns the pool.
//
// Pool configuration is caller-supplied via poolCfg (see PoolConfig);
// SlowQueryTracer (logs queries exceeding 1 s at WARN level) is always
// installed regardless of poolCfg.
//
// The caller is responsible for closing the pool (defer pool.Close()).
func Connect(ctx context.Context, dsn string, poolCfg PoolConfig) (*pgxpool.Pool, error) {
	if dsn == "" {
		return nil, ErrDSNEmpty
	}

	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("db: parse dsn: %w", err)
	}

	cfg.MaxConns = poolCfg.MaxConns
	cfg.MinConns = poolCfg.MinConns
	cfg.MaxConnLifetime = poolCfg.MaxConnLifetime
	cfg.MaxConnIdleTime = poolCfg.MaxConnIdleTime
	cfg.ConnConfig.Tracer = NewSlowQueryTracer(slog.Default())

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
		return ErrPoolNil
	}
	if migrationsDir == "" {
		return ErrMigrationsDirEmpty
	}

	sqlDB := stdlib.OpenDBFromPool(pool)
	defer func() { _ = sqlDB.Close() }()

	fsys := os.DirFS(migrationsDir)

	// A Postgres session-level advisory lock serializes concurrent Up() calls
	// across processes — e.g. the Helm chart's per-replica migrate
	// initContainer running on several pods during a rolling update or HPA
	// scale-out — so they apply migrations one at a time instead of racing
	// on the same DDL. Without WithSessionLocker, goose disables locking
	// entirely.
	locker, err := lock.NewPostgresSessionLocker()
	if err != nil {
		return fmt.Errorf("db: create session locker: %w", err)
	}
	provider, err := goose.NewProvider(goose.DialectPostgres, sqlDB, fsys, goose.WithSessionLocker(locker))
	if err != nil {
		return fmt.Errorf("db: create goose provider: %w", err)
	}

	results, err := provider.Up(ctx)
	var migrationErrs []error
	for _, r := range results {
		if r.Error != nil {
			slog.ErrorContext(
				ctx, "migration failed",
				slog.String("file", r.Source.Path),
				slog.Any("error", r.Error),
			)
			migrationErrs = append(migrationErrs, fmt.Errorf("migration %s: %w", r.Source.Path, r.Error))
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
	if len(migrationErrs) > 0 {
		return fmt.Errorf("db: %d migration(s) failed: %w", len(migrationErrs), errors.Join(migrationErrs...))
	}

	return nil
}
