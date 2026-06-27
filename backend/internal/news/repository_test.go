package news_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/news"
	"github.com/yoadey/team-manager/backend/internal/testutil"
)

func TestNewsRepository_CreateAndList(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := news.NewRepository(pool)
	ctx := context.Background()

	uid := uuid.New().String()
	tid := uuid.New().String()

	_, err := pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'News Author', 'news@example.com', '#336699')`,
		uid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO teams (id, name) VALUES ($1, 'News Team')`,
		tid)
	require.NoError(t, err)

	teamID := uuid.MustParse(tid)
	authorID := uuid.MustParse(uid)

	item, err := repo.Create(ctx, teamID, authorID, "Hello World", "First post", false)
	require.NoError(t, err)
	require.NotNil(t, item)
	assert.Equal(t, "Hello World", item.Title)
	assert.Equal(t, "First post", item.Body)
	assert.False(t, item.Pinned)
	assert.Equal(t, "News Author", *item.AuthorName)

	pinned, err := repo.Create(ctx, teamID, authorID, "Pinned Post", "Important", true)
	require.NoError(t, err)
	require.NotNil(t, pinned)
	assert.True(t, pinned.Pinned)

	list, err := repo.ListByTeam(ctx, teamID, 50, nil)
	require.NoError(t, err)
	require.Len(t, list, 2)
	// Pinned items come first.
	assert.True(t, list[0].Pinned)
	assert.False(t, list[1].Pinned)
}

func TestNewsRepository_Update(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := news.NewRepository(pool)
	ctx := context.Background()

	uid := uuid.New().String()
	tid := uuid.New().String()

	_, err := pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Upd Author', 'upd-news@example.com', '#000000')`,
		uid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO teams (id, name) VALUES ($1, 'Upd News Team')`,
		tid)
	require.NoError(t, err)

	item, err := repo.Create(ctx, uuid.MustParse(tid), uuid.MustParse(uid), "Old Title", "Old body", false)
	require.NoError(t, err)

	newTitle := "New Title"
	pinned := true
	updated, err := repo.Update(ctx, item.Id, &newTitle, nil, &pinned)
	require.NoError(t, err)
	assert.Equal(t, "New Title", updated.Title)
	assert.Equal(t, "Old body", updated.Body)
	assert.True(t, updated.Pinned)
}

func TestNewsRepository_Delete(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := news.NewRepository(pool)
	ctx := context.Background()

	uid := uuid.New().String()
	tid := uuid.New().String()

	_, err := pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Del Author', 'del-news@example.com', '#ffffff')`,
		uid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO teams (id, name) VALUES ($1, 'Del News Team')`,
		tid)
	require.NoError(t, err)

	teamID := uuid.MustParse(tid)
	item, err := repo.Create(ctx, teamID, uuid.MustParse(uid), "To Delete", "body", false)
	require.NoError(t, err)

	require.NoError(t, repo.Delete(ctx, item.Id))

	list, err := repo.ListByTeam(ctx, teamID, 50, nil)
	require.NoError(t, err)
	assert.Empty(t, list)
}
