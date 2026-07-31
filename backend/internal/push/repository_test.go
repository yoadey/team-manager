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

// TestRepository_GetPreferences_DefaultsWhenNoRow verifies that a member who
// has never saved preferences gets DefaultCategoryPreferences() (everything
// enabled) rather than an error or a zero-value (everything disabled) --
// existing subscribers predate this table and must keep receiving push
// exactly as before.
func TestRepository_GetPreferences_DefaultsWhenNoRow(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := push.NewRepository(pool)
	ctx := context.Background()

	teamID := uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Prefs Default Team')`, teamID)
	require.NoError(t, err)
	userID := seedUser(t, ctx, pool, "Prefs User", "prefs@example.com")

	got, err := repo.GetPreferences(ctx, teamID, userID)
	require.NoError(t, err)
	assert.Equal(t, push.DefaultCategoryPreferences(), got)
}

// TestRepository_UpsertPreferences_RoundTrips verifies preferences saved via
// UpsertPreferences are the ones GetPreferences later returns, and that
// saving twice updates the same row rather than duplicating it.
func TestRepository_UpsertPreferences_RoundTrips(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := push.NewRepository(pool)
	ctx := context.Background()

	teamID := uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Prefs Roundtrip Team')`, teamID)
	require.NoError(t, err)
	userID := seedUser(t, ctx, pool, "Prefs User 2", "prefs2@example.com")

	prefs := push.CategoryPreferences{Attendance: true, Events: false, News: true, Polls: false, Absence: true}
	require.NoError(t, repo.UpsertPreferences(ctx, teamID, userID, prefs))

	got, err := repo.GetPreferences(ctx, teamID, userID)
	require.NoError(t, err)
	assert.Equal(t, prefs, got)

	var count int
	require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM push_preferences WHERE team_id = $1 AND user_id = $2`, teamID, userID).Scan(&count))
	assert.Equal(t, 1, count)

	// Saving again updates the same row rather than duplicating it.
	prefs.News = false
	require.NoError(t, repo.UpsertPreferences(ctx, teamID, userID, prefs))
	got, err = repo.GetPreferences(ctx, teamID, userID)
	require.NoError(t, err)
	assert.Equal(t, prefs, got)
	require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM push_preferences WHERE team_id = $1 AND user_id = $2`, teamID, userID).Scan(&count))
	assert.Equal(t, 1, count, "saving preferences again must update, not duplicate")
}

// TestRepository_Preferences_ScopedPerTeam verifies a member's preferences
// in one team don't affect another team they belong to.
func TestRepository_Preferences_ScopedPerTeam(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	repo := push.NewRepository(pool)
	ctx := context.Background()

	teamA := uuid.New()
	teamB := uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Team A'), ($2, 'Team B')`, teamA, teamB)
	require.NoError(t, err)
	userID := seedUser(t, ctx, pool, "Multi Team User", "multiteam@example.com")

	require.NoError(t, repo.UpsertPreferences(ctx, teamA, userID, push.CategoryPreferences{Attendance: true, Events: true, News: false, Polls: true, Absence: true}))

	gotA, err := repo.GetPreferences(ctx, teamA, userID)
	require.NoError(t, err)
	assert.False(t, gotA.News, "team A's news preference must be disabled")

	gotB, err := repo.GetPreferences(ctx, teamB, userID)
	require.NoError(t, err)
	assert.Equal(t, push.DefaultCategoryPreferences(), gotB, "team B must be unaffected by team A's preferences")
}
