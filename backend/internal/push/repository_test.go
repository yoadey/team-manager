package push_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/push"
	"github.com/yoadey/team-manager/backend/internal/testutil"
)

func seedUser(t *testing.T, ctx context.Context, pool *pgxpool.Pool, name, email string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, $2, $3, '#aaaaaa')`, id, name, email)
	require.NoError(t, err)
	return id
}

func TestRepository_UpsertAndDelete(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := push.NewRepository(pool)
	ctx := context.Background()

	userID := seedUser(t, ctx, pool, "Push User", "push@example.com")
	sub := push.Subscription{Endpoint: "https://push.example/abc", P256dh: "p256dh-1", AuthKey: "auth-1"}

	require.NoError(t, repo.Upsert(ctx, userID, sub))

	var count int
	require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM push_subscriptions WHERE user_id = $1`, userID).Scan(&count))
	assert.Equal(t, 1, count)

	// Re-subscribing the same endpoint with new keys updates in place, not duplicates.
	sub.P256dh = "p256dh-2"
	require.NoError(t, repo.Upsert(ctx, userID, sub))
	require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM push_subscriptions WHERE user_id = $1`, userID).Scan(&count))
	assert.Equal(t, 1, count, "re-subscribing the same endpoint must update, not duplicate")

	var gotP256dh string
	require.NoError(t, pool.QueryRow(ctx, `SELECT p256dh FROM push_subscriptions WHERE endpoint = $1`, sub.Endpoint).Scan(&gotP256dh))
	assert.Equal(t, "p256dh-2", gotP256dh)

	require.NoError(t, repo.Delete(ctx, userID, sub.Endpoint))
	require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM push_subscriptions WHERE user_id = $1`, userID).Scan(&count))
	assert.Equal(t, 0, count)
}

func TestRepository_Delete_ScopedToOwner(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := push.NewRepository(pool)
	ctx := context.Background()

	owner := seedUser(t, ctx, pool, "Owner", "owner@example.com")
	attacker := seedUser(t, ctx, pool, "Attacker", "attacker@example.com")
	sub := push.Subscription{Endpoint: "https://push.example/owner", P256dh: "p", AuthKey: "a"}
	require.NoError(t, repo.Upsert(ctx, owner, sub))

	// Attacker naming the owner's endpoint must not delete it.
	require.NoError(t, repo.Delete(ctx, attacker, sub.Endpoint))

	var count int
	require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM push_subscriptions WHERE user_id = $1`, owner).Scan(&count))
	assert.Equal(t, 1, count, "a user must not be able to delete another user's subscription")
}

func TestRepository_ListForTeamExcludingUser(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := push.NewRepository(pool)
	ctx := context.Background()

	teamID := uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Push Team')`, teamID)
	require.NoError(t, err)

	actor := seedUser(t, ctx, pool, "Actor", "actor2@example.com")
	member := seedUser(t, ctx, pool, "Member", "member@example.com")
	outsider := seedUser(t, ctx, pool, "Outsider", "outsider@example.com")

	_, err = pool.Exec(ctx, `INSERT INTO memberships (team_id, user_id) VALUES ($1, $2), ($1, $3)`, teamID, actor, member)
	require.NoError(t, err)

	require.NoError(t, repo.Upsert(ctx, actor, push.Subscription{Endpoint: "https://push.example/actor", P256dh: "p", AuthKey: "a"}))
	require.NoError(t, repo.Upsert(ctx, member, push.Subscription{Endpoint: "https://push.example/member", P256dh: "p", AuthKey: "a"}))
	require.NoError(t, repo.Upsert(ctx, outsider, push.Subscription{Endpoint: "https://push.example/outsider", P256dh: "p", AuthKey: "a"}))

	subs, err := repo.ListForTeamExcludingUser(ctx, teamID, actor)
	require.NoError(t, err)
	require.Len(t, subs, 1, "must include only current team members other than the excluded actor")
	assert.Equal(t, member, subs[0].UserId)
	assert.Equal(t, "https://push.example/member", subs[0].Subscription.Endpoint)
}
