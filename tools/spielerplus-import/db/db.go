// Package db writes imported data directly into the Teamverwaltung Postgres
// database, bypassing the backend HTTP API/services (see design.md for why).
// It re-implements the invariants those services would otherwise enforce:
// placeholder-but-non-empty password hashes + pre-verified email for new
// users, the attendance status enum, and absence overlap/span checks.
package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Store wraps a Postgres connection pool with the inserts this importer
// needs.
type Store struct {
	Pool *pgxpool.Pool
	// DryRun, when true, makes every Ensure*/Insert*/Upsert* method perform
	// its lookups/validation (SELECTs, constraint checks) but skip the
	// actual write, so a dry run's reported counts reflect what a real run
	// would do.
	DryRun bool
}

// dryRunID is returned by write methods in dry-run mode in place of a real
// generated id, since nothing was actually inserted.
const dryRunID = "00000000-0000-0000-0000-000000000000"

// Open connects to databaseURL.
func Open(ctx context.Context, databaseURL string, dryRun bool) (*Store, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("db: connect: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("db: ping: %w", err)
	}
	return &Store{Pool: pool, DryRun: dryRun}, nil
}

func (s *Store) Close() {
	s.Pool.Close()
}
