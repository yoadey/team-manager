package calendarfeed_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/calendarfeed"
	"github.com/yoadey/team-manager/backend/internal/testutil"
)

func TestRepository_IssueTokenRotatesAndFind(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := calendarfeed.NewRepository(pool)
	ctx := context.Background()

	userID := uuid.New()
	teamID := uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Feed User', 'feed@example.com', '#aaaaaa')`, userID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Feed Team')`, teamID)
	require.NoError(t, err)

	tok1, err := repo.IssueToken(ctx, userID, teamID)
	require.NoError(t, err)
	require.NotEmpty(t, tok1)

	row, err := repo.FindActiveByToken(ctx, tok1)
	require.NoError(t, err)
	assert.Equal(t, userID, row.UserId)
	assert.Equal(t, teamID, row.TeamId)

	// Re-issuing rotates: the old token stops resolving.
	tok2, err := repo.IssueToken(ctx, userID, teamID)
	require.NoError(t, err)
	assert.NotEqual(t, tok1, tok2)

	_, err = repo.FindActiveByToken(ctx, tok1)
	require.Error(t, err)
	assert.ErrorIs(t, err, pgx.ErrNoRows, "the previous token must stop resolving once a new one is issued")

	row2, err := repo.FindActiveByToken(ctx, tok2)
	require.NoError(t, err)
	assert.Equal(t, userID, row2.UserId)
}

func TestRepository_Revoke(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := calendarfeed.NewRepository(pool)
	ctx := context.Background()

	userID := uuid.New()
	teamID := uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Feed User 2', 'feed2@example.com', '#aaaaaa')`, userID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Feed Team 2')`, teamID)
	require.NoError(t, err)

	tok, err := repo.IssueToken(ctx, userID, teamID)
	require.NoError(t, err)

	require.NoError(t, repo.Revoke(ctx, userID, teamID))

	_, err = repo.FindActiveByToken(ctx, tok)
	require.True(t, errors.Is(err, pgx.ErrNoRows))
}

func TestRepository_FindActiveByToken_UnknownToken(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := calendarfeed.NewRepository(pool)

	_, err := repo.FindActiveByToken(context.Background(), "does-not-exist")
	require.Error(t, err)
	assert.ErrorIs(t, err, pgx.ErrNoRows)
}

func TestRepository_IssueToken_DefaultsToAllTypesAndBirthdaysOn(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := calendarfeed.NewRepository(pool)
	ctx := context.Background()

	userID := uuid.New()
	teamID := uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Feed User 3', 'feed3@example.com', '#aaaaaa')`, userID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Feed Team 3')`, teamID)
	require.NoError(t, err)

	tok, err := repo.IssueToken(ctx, userID, teamID)
	require.NoError(t, err)

	row, err := repo.FindActiveByToken(ctx, tok)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"training", "auftritt", "event"}, row.Types)
	assert.True(t, row.IncludeBirthdays)
}

func TestRepository_IssueToken_CarriesForwardExistingSelectionOnRotate(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := calendarfeed.NewRepository(pool)
	ctx := context.Background()

	userID := uuid.New()
	teamID := uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Feed User 4', 'feed4@example.com', '#aaaaaa')`, userID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Feed Team 4')`, teamID)
	require.NoError(t, err)

	tok1, err := repo.IssueToken(ctx, userID, teamID)
	require.NoError(t, err)
	require.NoError(t, repo.UpdateSettings(ctx, userID, teamID, []string{"auftritt"}, false))

	// Rotating (re-issuing) must not silently reset a customized selection
	// back to the default -- that would defeat "regenerate link" as a
	// routine security action.
	tok2, err := repo.IssueToken(ctx, userID, teamID)
	require.NoError(t, err)
	assert.NotEqual(t, tok1, tok2)

	row, err := repo.FindActiveByToken(ctx, tok2)
	require.NoError(t, err)
	assert.Equal(t, []string{"auftritt"}, row.Types)
	assert.False(t, row.IncludeBirthdays)
}

func TestRepository_GetSettings_UnknownReturnsNoRows(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := calendarfeed.NewRepository(pool)

	_, _, err := repo.GetSettings(context.Background(), uuid.New(), uuid.New())
	require.Error(t, err)
	assert.ErrorIs(t, err, pgx.ErrNoRows)
}

func TestRepository_UpdateSettings_AppliesToActiveToken(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := calendarfeed.NewRepository(pool)
	ctx := context.Background()

	userID := uuid.New()
	teamID := uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Feed User 5', 'feed5@example.com', '#aaaaaa')`, userID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Feed Team 5')`, teamID)
	require.NoError(t, err)

	_, err = repo.IssueToken(ctx, userID, teamID)
	require.NoError(t, err)

	require.NoError(t, repo.UpdateSettings(ctx, userID, teamID, []string{"training"}, false))

	types, includeBirthdays, err := repo.GetSettings(ctx, userID, teamID)
	require.NoError(t, err)
	assert.Equal(t, []string{"training"}, types)
	assert.False(t, includeBirthdays)
}

func TestRepository_UpdateSettings_NoActiveTokenReturnsNoRows(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := calendarfeed.NewRepository(pool)

	err := repo.UpdateSettings(context.Background(), uuid.New(), uuid.New(), []string{"training"}, true)
	require.Error(t, err)
	assert.ErrorIs(t, err, pgx.ErrNoRows)
}
