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
