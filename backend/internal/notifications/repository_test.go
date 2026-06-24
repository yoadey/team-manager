package notifications_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/notifications"
	"github.com/yoadey/team-manager/backend/internal/testutil"
)

func TestNotificationsRepository_ListAndMarkSeen(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := notifications.NewRepository(pool)
	ctx := context.Background()

	uid := uuid.New().String()
	tid := uuid.New().String()

	_, err := pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Notif User', 'notif@example.com', '#aaaaaa')`, uid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO teams (id, name) VALUES ($1, 'Notif Team')`, tid)
	require.NoError(t, err)

	teamID := uuid.MustParse(tid)
	userID := uuid.MustParse(uid)

	// Seed a notification.
	_, err = pool.Exec(ctx,
		`INSERT INTO notifications (team_id, type, actor_id, status, title)
		 VALUES ($1, 'news', $2, 'info', 'New article posted')`,
		tid, uid)
	require.NoError(t, err)

	// Before marking seen, notification should be unread.
	list, err := repo.ListByTeamAndUser(ctx, teamID, userID)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.True(t, list[0].Unread)
	assert.Equal(t, "New article posted", *list[0].Title)

	// Mark seen.
	require.NoError(t, repo.MarkSeen(ctx, teamID, userID))

	// After marking seen, notification should be read.
	list, err = repo.ListByTeamAndUser(ctx, teamID, userID)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.False(t, list[0].Unread)

	// Calling MarkSeen again (upsert) should not error.
	require.NoError(t, repo.MarkSeen(ctx, teamID, userID))
}

func TestNotificationsRepository_EmptyForNewTeam(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := notifications.NewRepository(pool)
	ctx := context.Background()

	uid := uuid.New().String()
	tid := uuid.New().String()

	_, err := pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Empty User', 'empty@example.com', '#cccccc')`, uid)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO teams (id, name) VALUES ($1, 'Empty Notif Team')`, tid)
	require.NoError(t, err)

	list, err := repo.ListByTeamAndUser(ctx, uuid.MustParse(tid), uuid.MustParse(uid))
	require.NoError(t, err)
	assert.Empty(t, list)
}
