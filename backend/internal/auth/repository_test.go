package auth_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/auth"
	"github.com/yoadey/team-manager/backend/internal/testutil"
)

func TestRepository_FindUserByEmail(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := auth.NewRepository(pool)
	ctx := context.Background()

	// Insert a test user directly.
	_, err := pool.Exec(
		ctx,
		`INSERT INTO users (id, name, email, avatar_color, password_hash)
		 VALUES ('11111111-1111-1111-1111-111111111111', 'Alice', 'alice@example.com', '#ff0000', 'hash1')`,
	)
	require.NoError(t, err)

	user, err := repo.FindUserByEmail(ctx, "alice@example.com")
	require.NoError(t, err)
	assert.Equal(t, "Alice", user.Name)
	assert.Equal(t, "alice@example.com", user.Email)
	assert.Equal(t, "#ff0000", user.AvatarColor)
	assert.Equal(t, "hash1", user.PasswordHash)
	assert.Equal(t, "11111111-1111-1111-1111-111111111111", user.Id.String())
}

func TestRepository_FindUserByID(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := auth.NewRepository(pool)
	ctx := context.Background()

	_, err := pool.Exec(
		ctx,
		`INSERT INTO users (id, name, email, avatar_color)
		 VALUES ('22222222-2222-2222-2222-222222222222', 'Bob', 'bob@example.com', '#00ff00')`,
	)
	require.NoError(t, err)

	user, err := repo.FindUserByID(ctx, "22222222-2222-2222-2222-222222222222")
	require.NoError(t, err)
	assert.Equal(t, "Bob", user.Name)
	assert.Equal(t, "bob@example.com", user.Email)
}

func TestRepository_CreateAndFindSession(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := auth.NewRepository(pool)
	ctx := context.Background()

	// Need a user first (FK constraint).
	_, err := pool.Exec(
		ctx,
		`INSERT INTO users (id, name, email, avatar_color)
		 VALUES ('33333333-3333-3333-3333-333333333333', 'Carol', 'carol@example.com', '#0000ff')`,
	)
	require.NoError(t, err)

	expiresAt := time.Now().Add(24 * time.Hour).UTC().Truncate(time.Millisecond)
	sess, err := repo.CreateSession(ctx, "33333333-3333-3333-3333-333333333333", "testhash123", expiresAt)
	require.NoError(t, err)
	assert.NotEmpty(t, sess.Id)
	assert.Equal(t, "testhash123", sess.TokenHash)
	assert.Equal(t, "password", sess.Provider)
	assert.True(t, sess.ExpiresAt.After(time.Now()), "ExpiresAt should be in the future")

	// Find it back.
	found, err := repo.FindSession(ctx, "testhash123")
	require.NoError(t, err)
	assert.Equal(t, sess.Id, found.Id)
	assert.Equal(t, "33333333-3333-3333-3333-333333333333", found.UserId.String())
}

func TestRepository_EraseUser_SoleSettingsAdmin_Blocked(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := auth.NewRepository(pool)
	ctx := context.Background()

	teamID := "aaaaaaaa-1111-1111-1111-111111111111"
	userID := "aaaaaaaa-2222-2222-2222-222222222222"
	_, err := pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Erase Test Team')`, teamID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Sole Admin', 'sole-admin@example.com', '#123456')`, userID)
	require.NoError(t, err)
	var membershipID, roleID string
	require.NoError(t, pool.QueryRow(ctx, `INSERT INTO memberships (team_id, user_id) VALUES ($1, $2) RETURNING id`, teamID, userID).Scan(&membershipID))
	require.NoError(t, pool.QueryRow(ctx, `
		INSERT INTO roles (team_id, name, permissions)
		VALUES ($1, 'Admin', '{"events":"write","members":"write","finances":"write","news":"write","polls":"write","settings":"write"}')
		RETURNING id
	`, teamID).Scan(&roleID))
	_, err = pool.Exec(ctx, `INSERT INTO membership_roles (membership_id, role_id) VALUES ($1, $2)`, membershipID, roleID)
	require.NoError(t, err)

	err = repo.EraseUser(ctx, userID)
	require.ErrorIs(t, err, auth.ErrSoleSettingsAdmin)

	// Must not have been anonymized -- the erasure was fully rejected.
	var deletedAt *time.Time
	require.NoError(t, pool.QueryRow(ctx, `SELECT deleted_at FROM users WHERE id = $1`, userID).Scan(&deletedAt))
	assert.Nil(t, deletedAt)
}

func TestRepository_EraseUser_AnotherSettingsAdminExists_Succeeds(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := auth.NewRepository(pool)
	ctx := context.Background()

	teamID := "bbbbbbbb-1111-1111-1111-111111111111"
	userID := "bbbbbbbb-2222-2222-2222-222222222222"
	otherAdminID := "bbbbbbbb-3333-3333-3333-333333333333"
	_, err := pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Erase Test Team 2')`, teamID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Leaving Admin', 'leaving-admin@example.com', '#123456')`, userID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Other Admin', 'other-admin@example.com', '#654321')`, otherAdminID)
	require.NoError(t, err)

	var roleID string
	require.NoError(t, pool.QueryRow(ctx, `
		INSERT INTO roles (team_id, name, permissions)
		VALUES ($1, 'Admin', '{"events":"write","members":"write","finances":"write","news":"write","polls":"write","settings":"write"}')
		RETURNING id
	`, teamID).Scan(&roleID))

	for _, uid := range []string{userID, otherAdminID} {
		var membershipID string
		require.NoError(t, pool.QueryRow(ctx, `INSERT INTO memberships (team_id, user_id) VALUES ($1, $2) RETURNING id`, teamID, uid).Scan(&membershipID))
		_, err = pool.Exec(ctx, `INSERT INTO membership_roles (membership_id, role_id) VALUES ($1, $2)`, membershipID, roleID)
		require.NoError(t, err)
	}

	require.NoError(t, repo.EraseUser(ctx, userID))

	var deletedAt *time.Time
	require.NoError(t, pool.QueryRow(ctx, `SELECT deleted_at FROM users WHERE id = $1`, userID).Scan(&deletedAt))
	assert.NotNil(t, deletedAt)
}

func TestRepository_DeleteSession(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := auth.NewRepository(pool)
	ctx := context.Background()

	_, err := pool.Exec(
		ctx,
		`INSERT INTO users (id, name, email, avatar_color)
		 VALUES ('44444444-4444-4444-4444-444444444444', 'Dave', 'dave@example.com', '#ffffff')`,
	)
	require.NoError(t, err)

	sess, err := repo.CreateSession(ctx, "44444444-4444-4444-4444-444444444444", "deletehash", time.Now().Add(time.Hour))
	require.NoError(t, err)
	require.NotEmpty(t, sess.Id)

	err = repo.DeleteSession(ctx, "deletehash")
	require.NoError(t, err)

	_, err = repo.FindSession(ctx, "deletehash")
	assert.Error(t, err, "FindSession should return error after deletion")
}
