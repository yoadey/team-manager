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

// HasPhoto is computed via a LEFT JOIN on users (actor_id is nullable for
// system notifications), so a notification with no actor at all must not
// error the scan and must report HasPhoto=false rather than leaking a NULL
// into a non-pointer bool.
func TestNotificationsRepository_ListByTeamAndUser_HasPhoto(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := notifications.NewRepository(pool)
	ctx := context.Background()

	uidWithPhoto := uuid.New().String()
	uidNoPhoto := uuid.New().String()
	tid := uuid.New().String()

	_, err := pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color, photo_data, photo_mime) VALUES ($1, 'Has Photo', 'hasphoto@example.com', '#111111', $2, 'image/png')`,
		uidWithPhoto, []byte{0xff, 0xd8})
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'No Photo', 'nophoto@example.com', '#222222')`, uidNoPhoto)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'HasPhoto Team')`, tid)
	require.NoError(t, err)

	_, err = pool.Exec(ctx,
		`INSERT INTO notifications (team_id, type, actor_id, status, title) VALUES ($1, 'news', $2, 'info', 'From photo user')`,
		tid, uidWithPhoto)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO notifications (team_id, type, actor_id, status, title) VALUES ($1, 'news', $2, 'info', 'From no-photo user')`,
		tid, uidNoPhoto)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO notifications (team_id, type, actor_id, status, title) VALUES ($1, 'system', NULL, 'info', 'System notification')`,
		tid)
	require.NoError(t, err)

	list, err := repo.ListByTeamAndUser(ctx, uuid.MustParse(tid), uuid.New())
	require.NoError(t, err)
	require.Len(t, list, 3)

	byTitle := map[string]bool{}
	for _, n := range list {
		byTitle[*n.Title] = n.HasPhoto
	}
	assert.True(t, byTitle["From photo user"])
	assert.False(t, byTitle["From no-photo user"])
	assert.False(t, byTitle["System notification"], "notification with no actor must not error and must report HasPhoto=false")
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
