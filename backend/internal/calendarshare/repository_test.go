package calendarshare_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/calendarshare"
	"github.com/yoadey/team-manager/backend/internal/testutil"
)

func TestRepository_Grant_CreatesShareAndIsIdempotent(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := calendarshare.NewRepository(pool)
	ctx := context.Background()

	ownerID := uuid.New()
	viewerID := uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Owner Team')`, ownerID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Viewer Team')`, viewerID)
	require.NoError(t, err)

	row, err := repo.Grant(ctx, ownerID, viewerID)
	require.NoError(t, err)
	assert.Equal(t, viewerID, row.TeamId)
	assert.Equal(t, "Viewer Team", row.TeamName)

	// Granting an already-active share is a no-op that returns the
	// existing row, not an error.
	row2, err := repo.Grant(ctx, ownerID, viewerID)
	require.NoError(t, err)
	assert.Equal(t, row.CreatedAt.Unix(), row2.CreatedAt.Unix())
}

func TestRepository_Grant_UnknownViewerTeam(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := calendarshare.NewRepository(pool)
	ctx := context.Background()

	ownerID := uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Owner Team 2')`, ownerID)
	require.NoError(t, err)

	_, err = repo.Grant(ctx, ownerID, uuid.New())
	require.Error(t, err)
	assert.ErrorIs(t, err, calendarshare.ErrTeamNotFound)
}

func TestRepository_Revoke_RemovesGrantImmediately(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := calendarshare.NewRepository(pool)
	ctx := context.Background()

	ownerID := uuid.New()
	viewerID := uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Owner Team 3')`, ownerID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Viewer Team 3')`, viewerID)
	require.NoError(t, err)

	_, err = repo.Grant(ctx, ownerID, viewerID)
	require.NoError(t, err)

	has, err := repo.HasGrant(ctx, ownerID, viewerID)
	require.NoError(t, err)
	assert.True(t, has)

	require.NoError(t, repo.Revoke(ctx, ownerID, viewerID))

	has, err = repo.HasGrant(ctx, ownerID, viewerID)
	require.NoError(t, err)
	assert.False(t, has, "a revoked grant must deny access immediately")

	// Revoking a non-existent grant is a no-op, not an error.
	require.NoError(t, repo.Revoke(ctx, ownerID, viewerID))
}

func TestRepository_ListGrantedByOwnerAndListGrantedToViewer(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := calendarshare.NewRepository(pool)
	ctx := context.Background()

	ownerID := uuid.New()
	viewerA := uuid.New()
	viewerB := uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Owner Team 4')`, ownerID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Viewer A')`, viewerA)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Viewer B')`, viewerB)
	require.NoError(t, err)

	_, err = repo.Grant(ctx, ownerID, viewerA)
	require.NoError(t, err)
	_, err = repo.Grant(ctx, ownerID, viewerB)
	require.NoError(t, err)

	granted, err := repo.ListGrantedByOwner(ctx, ownerID)
	require.NoError(t, err)
	require.Len(t, granted, 2)

	sources, err := repo.ListGrantedToViewer(ctx, viewerA)
	require.NoError(t, err)
	require.Len(t, sources, 1)
	assert.Equal(t, ownerID, sources[0].TeamId)

	// A team with no grants sees an empty list, not an error.
	none, err := repo.ListGrantedToViewer(ctx, uuid.New())
	require.NoError(t, err)
	assert.Empty(t, none)
}

func TestRepository_ListRedactedEvents_ExcludesCancelledAndRespectsDateRange(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := calendarshare.NewRepository(pool)
	ctx := context.Background()

	teamID := uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Schedule Team')`, teamID)
	require.NoError(t, err)

	inRange := uuid.New()
	outOfRange := uuid.New()
	cancelled := uuid.New()
	_, err = pool.Exec(ctx, `
		INSERT INTO events (id, team_id, type, title, date, location, note, start_time, end_time, status)
		VALUES
			($1, $4, 'training', 'In Range', '2026-06-15', 'Halle 1', 'secret note', '18:00', '20:00', 'active'),
			($2, $4, 'training', 'Out Of Range', '2026-01-01', 'Halle 2', 'secret note', NULL, NULL, 'active'),
			($3, $4, 'training', 'Cancelled', '2026-06-16', 'Halle 3', 'secret note', NULL, NULL, 'cancelled')
	`, inRange, outOfRange, cancelled, teamID)
	require.NoError(t, err)

	from := mustParseDate(t, "2026-06-01")
	to := mustParseDate(t, "2026-06-30")
	rows, err := repo.ListRedactedEvents(ctx, teamID, &from, &to)
	require.NoError(t, err)
	require.Len(t, rows, 1, "must exclude both the out-of-range and the cancelled event")
	assert.Equal(t, inRange, rows[0].Id)
	assert.Equal(t, "In Range", rows[0].Title)
	assert.Equal(t, "Halle 1", *rows[0].Location)
	require.NotNil(t, rows[0].StartTime)
	assert.Equal(t, "18:00", *rows[0].StartTime)

	// No date bounds: still excludes the cancelled event, includes both active ones.
	all, err := repo.ListRedactedEvents(ctx, teamID, nil, nil)
	require.NoError(t, err)
	assert.Len(t, all, 2)
}

func mustParseDate(t *testing.T, s string) time.Time {
	t.Helper()
	tm, err := time.Parse("2006-01-02", s)
	require.NoError(t, err)
	return tm
}

func TestRepository_HasGrant_FalseWhenNoGrantExists(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := calendarshare.NewRepository(pool)

	has, err := repo.HasGrant(context.Background(), uuid.New(), uuid.New())
	require.NoError(t, err)
	assert.False(t, has)
}

// Regression guard: pgx.ErrNoRows must never leak unwrapped from Grant for a
// missing viewer team -- callers check errors.Is(err, calendarshare.ErrTeamNotFound).
func TestRepository_Grant_UnknownViewerTeamDoesNotLeakPgxErrNoRows(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := calendarshare.NewRepository(pool)
	ctx := context.Background()

	ownerID := uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Owner Team 5')`, ownerID)
	require.NoError(t, err)

	_, err = repo.Grant(ctx, ownerID, uuid.New())
	require.Error(t, err)
	assert.NotErrorIs(t, err, pgx.ErrNoRows)
}
