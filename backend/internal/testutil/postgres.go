package testutil

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	dockerclient "github.com/moby/moby/client"
	"github.com/pressly/goose/v3"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// migrationsDir returns the absolute path to the goose migrations directory.
// It derives the backend root from the location of this source file.
func migrationsDir() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("testutil: runtime.Caller failed")
	}
	// filename is .../backend/internal/testutil/postgres.go
	// go up three directories: testutil -> internal -> backend
	backendRoot := filepath.Join(filepath.Dir(filename), "..", "..")
	return filepath.Join(backendRoot, "internal", "db", "migrations")
}

// RequireDocker skips the test if a Docker daemon is not reachable.
// Call at the top of any test that uses testcontainers.
func RequireDocker(t *testing.T) {
	t.Helper()
	client, err := testcontainers.NewDockerClientWithOpts(context.Background())
	if err != nil {
		t.Skipf("Docker not available (%v) — skipping integration test", err)
	}
	defer client.Close()
	if _, err := client.Ping(context.Background(), dockerclient.PingOptions{}); err != nil {
		t.Skipf("Docker daemon not reachable (%v) — skipping integration test", err)
	}
}

// NewTestDB spins up a postgres:17 testcontainer, runs all goose migrations, and
// returns a connected *pgxpool.Pool. The container is terminated via t.Cleanup.
func NewTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	RequireDocker(t)

	ctx := context.Background()

	ctr, err := tcpostgres.Run(
		ctx,
		"postgres:17",
		tcpostgres.WithDatabase("testdb"),
		tcpostgres.WithUsername("testuser"),
		tcpostgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2),
		),
	)
	if err != nil {
		t.Fatalf("testutil: start postgres container: %v", err)
	}

	t.Cleanup(func() {
		if err := ctr.Terminate(context.Background()); err != nil {
			t.Logf("testutil: terminate postgres container: %v", err)
		}
	})

	dsn, err := ctr.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("testutil: get connection string: %v", err)
	}

	if err := runMigrations(ctx, dsn); err != nil {
		t.Fatalf("testutil: run migrations: %v", err)
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("testutil: create pgx pool: %v", err)
	}

	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("testutil: ping postgres: %v", err)
	}

	t.Cleanup(pool.Close)

	return pool
}

// runMigrations opens a *sql.DB from the DSN and runs goose up migrations.
func runMigrations(ctx context.Context, dsn string) error {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return fmt.Errorf("parse dsn: %w", err)
	}

	sqlDB := stdlib.OpenDB(*cfg.ConnConfig)
	defer sqlDB.Close() //nolint:errcheck

	migrDir := migrationsDir()
	if _, err := os.Stat(migrDir); err != nil {
		return fmt.Errorf("migrations dir %q not found: %w", migrDir, err)
	}

	provider, err := goose.NewProvider(goose.DialectPostgres, sqlDB, os.DirFS(migrDir))
	if err != nil {
		return fmt.Errorf("create goose provider: %w", err)
	}

	results, err := provider.Up(ctx)
	if err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	for _, r := range results {
		if r.Error != nil {
			return fmt.Errorf("migration %s failed: %w", r.Source.Path, r.Error)
		}
	}

	return nil
}

// TruncateTables truncates the given tables with CASCADE in a single transaction.
// Useful for resetting state between sub-tests.
func TruncateTables(t *testing.T, pool *pgxpool.Pool, tables ...string) {
	t.Helper()

	if len(tables) == 0 {
		return
	}

	ctx := context.Background()

	conn, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("testutil: acquire connection: %v", err)
	}
	defer conn.Release()

	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatalf("testutil: begin transaction: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Quote each table name with PostgreSQL double-quote identifier syntax.
	quoted := make([]string, len(tables))
	for i, tbl := range tables {
		// Escape any embedded double-quotes by doubling them (SQL standard).
		safe := strings.ReplaceAll(tbl, `"`, `""`)
		quoted[i] = `"` + safe + `"`
	}

	stmt := fmt.Sprintf("TRUNCATE TABLE %s CASCADE", strings.Join(quoted, ", "))
	if _, err := tx.Exec(ctx, stmt); err != nil {
		t.Fatalf("testutil: truncate tables %v: %v", tables, err)
	}

	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("testutil: commit truncate: %v", err)
	}
}
