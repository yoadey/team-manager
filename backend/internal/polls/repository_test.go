package polls_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/polls"
	"github.com/yoadey/team-manager/backend/internal/testutil"
)

func TestPollRepository_CreateAndList(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := polls.NewRepository(pool)
	ctx := context.Background()

	uid := uuid.New().String()
	tid := uuid.New().String()

	_, err := pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Poll User', 'poll@example.com', '#123456')`,
		uid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO teams (id, name) VALUES ($1, 'Poll Team')`,
		tid)
	require.NoError(t, err)

	teamID := uuid.MustParse(tid)
	creatorID := uuid.MustParse(uid)

	pollID, err := repo.Create(ctx, teamID, creatorID, "Best player?", false, false, []string{"Alice", "Bob", "Charlie"})
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, pollID)

	pr, err := repo.FindByID(ctx, pollID)
	require.NoError(t, err)
	require.NotNil(t, pr)
	assert.Equal(t, "Best player?", pr.Question)
	assert.False(t, pr.Multiple)
	assert.False(t, pr.Anonymous)

	opts, err := repo.ListOptions(ctx, pollID)
	require.NoError(t, err)
	require.Len(t, opts, 3)
	assert.Equal(t, "Alice", opts[0].Text)
	assert.Equal(t, "Bob", opts[1].Text)
	assert.Equal(t, "Charlie", opts[2].Text)

	list, err := repo.ListByTeam(ctx, teamID, 50, nil)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, pollID, list[0].Id)
}

func TestPollRepository_Vote(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := polls.NewRepository(pool)
	ctx := context.Background()

	uid := uuid.New().String()
	tid := uuid.New().String()

	_, err := pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Voter', 'voter@example.com', '#aabbcc')`,
		uid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO teams (id, name) VALUES ($1, 'Vote Team')`,
		tid)
	require.NoError(t, err)

	teamID := uuid.MustParse(tid)
	userID := uuid.MustParse(uid)

	pollID, err := repo.Create(ctx, teamID, userID, "Vote?", false, false, []string{"Yes", "No"})
	require.NoError(t, err)

	opts, err := repo.ListOptions(ctx, pollID)
	require.NoError(t, err)
	require.Len(t, opts, 2)

	yesID := opts[0].Id

	err = repo.ReplaceVotes(ctx, pollID, userID, []uuid.UUID{yesID}, false)
	require.NoError(t, err)

	votes, err := repo.ListVotes(ctx, pollID)
	require.NoError(t, err)
	require.Len(t, votes, 1)
	assert.Equal(t, yesID, votes[0].OptionId)
	assert.Equal(t, userID, votes[0].UserId)

	// Replace with the other option.
	noID := opts[1].Id
	err = repo.ReplaceVotes(ctx, pollID, userID, []uuid.UUID{noID}, false)
	require.NoError(t, err)

	votes, err = repo.ListVotes(ctx, pollID)
	require.NoError(t, err)
	require.Len(t, votes, 1)
	assert.Equal(t, noID, votes[0].OptionId)
}

func TestPollRepository_Delete(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := polls.NewRepository(pool)
	ctx := context.Background()

	uid := uuid.New().String()
	tid := uuid.New().String()

	_, err := pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Del Voter', 'del-poll@example.com', '#ffffff')`,
		uid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO teams (id, name) VALUES ($1, 'Del Poll Team')`,
		tid)
	require.NoError(t, err)

	teamID := uuid.MustParse(tid)
	creatorID := uuid.MustParse(uid)

	pollID, err := repo.Create(ctx, teamID, creatorID, "To Delete?", false, false, []string{"Yes"})
	require.NoError(t, err)

	require.NoError(t, repo.Delete(ctx, pollID))

	list, err := repo.ListByTeam(ctx, teamID, 50, nil)
	require.NoError(t, err)
	assert.Empty(t, list)
}
