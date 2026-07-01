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
