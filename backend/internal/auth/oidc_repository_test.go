package auth_test

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/auth"
	"github.com/yoadey/team-manager/backend/internal/testutil"
)

const (
	oidcTestUserID  = "aaaaaaaa-1111-1111-1111-111111111111"
	oidcTestUserID2 = "aaaaaaaa-2222-2222-2222-222222222222"
)

func TestRepository_LinkAndFindOIDCAccount(t *testing.T) {
	t.Parallel()
	pool := testutil.NewTestDB(t)
	repo := newTestRepo(t, pool)
	ctx := context.Background()

	_, err := pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'OIDC User', 'oidc@example.com', '#ff0000')`,
		oidcTestUserID)
	require.NoError(t, err)

	// Nothing linked yet.
	_, err = repo.FindUserByOIDCSubject(ctx, "google", "subject-1")
	require.ErrorIs(t, err, pgx.ErrNoRows)

	require.NoError(t, repo.LinkOIDCAccount(ctx, oidcTestUserID, "google", "subject-1"))

	user, err := repo.FindUserByOIDCSubject(ctx, "google", "subject-1")
	require.NoError(t, err)
	assert.Equal(t, oidcTestUserID, user.Id.String())
	assert.Equal(t, "oidc@example.com", user.Email)

	// A different provider with the same subject is a different identity.
	_, err = repo.FindUserByOIDCSubject(ctx, "other", "subject-1")
	assert.ErrorIs(t, err, pgx.ErrNoRows)
}

// Linking is idempotent so a racing second login of the same new user does not
// fail; the end state is the same either way.
func TestRepository_LinkOIDCAccountIsIdempotent(t *testing.T) {
	t.Parallel()
	pool := testutil.NewTestDB(t)
	repo := newTestRepo(t, pool)
	ctx := context.Background()

	_, err := pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'OIDC User', 'idem@example.com', '#ff0000')`,
		oidcTestUserID2)
	require.NoError(t, err)

	require.NoError(t, repo.LinkOIDCAccount(ctx, oidcTestUserID2, "google", "subject-2"))
	require.NoError(t, repo.LinkOIDCAccount(ctx, oidcTestUserID2, "google", "subject-2"))

	var count int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT count(*) FROM oidc_accounts WHERE provider = 'google' AND subject = 'subject-2'`).Scan(&count))
	assert.Equal(t, 1, count)
}

// A soft-deleted account must not be reachable through its external identity
// either -- otherwise erasure would only close the password door.
func TestRepository_FindUserByOIDCSubjectSkipsDeletedUsers(t *testing.T) {
	t.Parallel()
	pool := testutil.NewTestDB(t)
	repo := newTestRepo(t, pool)
	ctx := context.Background()

	const id = "aaaaaaaa-3333-3333-3333-333333333333"
	_, err := pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color, deleted_at)
		 VALUES ($1, 'Gone', 'gone@example.com', '#ff0000', now())`, id)
	require.NoError(t, err)
	require.NoError(t, repo.LinkOIDCAccount(ctx, id, "google", "subject-3"))

	_, err = repo.FindUserByOIDCSubject(ctx, "google", "subject-3")
	assert.ErrorIs(t, err, pgx.ErrNoRows)
}

// sessions.provider used to be hardcoded to 'password' in Go, which made an
// OIDC session indistinguishable from a password one after the fact.
func TestRepository_CreateSessionRecordsProvider(t *testing.T) {
	t.Parallel()
	pool := testutil.NewTestDB(t)
	repo := newTestRepo(t, pool)
	ctx := context.Background()

	const id = "aaaaaaaa-4444-4444-4444-444444444444"
	_, err := pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'P', 'provider@example.com', '#ff0000')`, id)
	require.NoError(t, err)

	sess, err := repo.CreateSession(ctx, id, "hash-oidc", time.Now().Add(time.Hour), "google")
	require.NoError(t, err)
	assert.Equal(t, "google", sess.Provider)

	// An empty provider falls back to password rather than violating NOT NULL.
	fallback, err := repo.CreateSession(ctx, id, "hash-default", time.Now().Add(time.Hour), "")
	require.NoError(t, err)
	assert.Equal(t, auth.SessionProviderPassword, fallback.Provider)
}
