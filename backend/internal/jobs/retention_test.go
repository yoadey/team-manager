package jobs_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/jobs"
	"github.com/yoadey/team-manager/backend/internal/testutil"
)

// TestRetentionWorker_DeletesInBatches seeds more rows than a single
// retention batch (1000) to verify deleteBatched loops until the table is
// fully exhausted, rather than only removing the first batch.
func TestRetentionWorker_DeletesInBatches(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	ctx := context.Background()

	teamID := uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Retention Team')`, teamID)
	require.NoError(t, err)

	const oldRowCount = 1500
	oldCutoff := time.Now().Add(-100 * 24 * time.Hour)
	_, err = pool.Exec(ctx, `
		INSERT INTO notifications (team_id, type, created_at)
		SELECT $1, 'news', $2
		FROM generate_series(1, $3)
	`, teamID, oldCutoff, oldRowCount)
	require.NoError(t, err)

	// One recent notification must survive retention.
	_, err = pool.Exec(ctx,
		`INSERT INTO notifications (team_id, type, created_at) VALUES ($1, 'news', now())`, teamID)
	require.NoError(t, err)

	worker := jobs.NewRetentionWorker(pool, 90, 30, 365)
	require.NoError(t, worker.Work(ctx, &river.Job[jobs.RetentionArgs]{}))

	var remaining int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM notifications WHERE team_id = $1`, teamID).Scan(&remaining))
	assert.Equal(t, 1, remaining, "only the recent notification should survive retention")
}

// TestRetentionWorker_KeepsStillValidLongLivedSession is a regression test
// for a bug where session retention deleted rows based on created_at instead
// of expires_at: a session created long ago but with a long TTL (still
// valid, expires_at in the future) must survive retention even though its
// created_at is older than the retention window. Only sessions that have
// actually expired (and stayed expired past the retention grace period)
// should be purged.
func TestRetentionWorker_KeepsStillValidLongLivedSession(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	ctx := context.Background()

	userID := uuid.New()
	_, err := pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Retention User', 'retention@example.com', '#123456')`,
		userID)
	require.NoError(t, err)

	// Created 60 days ago (older than the 30-day retention window) but still
	// valid for another 30 days — must NOT be deleted.
	_, err = pool.Exec(ctx, `
		INSERT INTO sessions (user_id, token_hash, created_at, expires_at)
		VALUES ($1, 'still-valid-long-lived', now() - interval '60 days', now() + interval '30 days')
	`, userID)
	require.NoError(t, err)

	// Expired 100 days ago — must be deleted.
	_, err = pool.Exec(ctx, `
		INSERT INTO sessions (user_id, token_hash, created_at, expires_at)
		VALUES ($1, 'long-expired', now() - interval '130 days', now() - interval '100 days')
	`, userID)
	require.NoError(t, err)

	worker := jobs.NewRetentionWorker(pool, 90, 30, 365)
	require.NoError(t, worker.Work(ctx, &river.Job[jobs.RetentionArgs]{}))

	var tokens []string
	rows, err := pool.Query(ctx, `SELECT token_hash FROM sessions WHERE user_id = $1`, userID)
	require.NoError(t, err)
	for rows.Next() {
		var tok string
		require.NoError(t, rows.Scan(&tok))
		tokens = append(tokens, tok)
	}
	require.NoError(t, rows.Err())

	assert.Equal(t, []string{"still-valid-long-lived"}, tokens, "only the still-valid session should survive retention")
}
