package jobs_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	promtestutil "github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/jobs"
	"github.com/yoadey/team-manager/backend/internal/metrics"
	"github.com/yoadey/team-manager/backend/internal/push"
	"github.com/yoadey/team-manager/backend/internal/teams"
	"github.com/yoadey/team-manager/backend/internal/testutil"
)

// ─── push-delivery gating mocks ────────────────────────────────────────────

type mockPermsChecker struct {
	perms   teams.PermissionsJSON
	err     error
	calledN int
}

func (m *mockPermsChecker) GetPermissions(context.Context, uuid.UUID, uuid.UUID) (teams.PermissionsJSON, error) {
	m.calledN++
	return m.perms, m.err
}

type mockSubscriptionLister struct {
	subs    []push.SubscriptionForUser
	err     error
	calledN int
}

func (m *mockSubscriptionLister) ListForTeamExcludingUser(context.Context, uuid.UUID, uuid.UUID) ([]push.SubscriptionForUser, error) {
	m.calledN++
	return m.subs, m.err
}

// Regression test: a persistent NotificationWorker.Work failure (River
// retries with backoff and eventually discards the job) used to be
// completely invisible to Prometheus -- the triggering request (e.g.
// creating a poll/news item) still shows as a successful domain event via
// TeamEvents, so a dashboard would look entirely healthy while users
// silently stopped receiving notifications. Forces a real failure (a
// team_id with no matching row in teams, violating the NOT NULL FK) and
// asserts metrics.NotificationJobFailures increments.
func TestNotificationWorker_Work_IncrementsFailureMetricOnError(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	ctx := context.Background()

	before := promtestutil.ToFloat64(metrics.NotificationJobFailures)

	worker := jobs.NewNotificationWorker(pool)
	job := &river.Job[jobs.NotificationArgs]{
		JobRow: &rivertype.JobRow{ID: 999999},
		Args: jobs.NotificationArgs{
			TeamID:  uuid.New(), // no such team -- violates notifications.team_id's FK
			Type:    "news",
			ActorID: uuid.New(),
		},
	}

	err := worker.Work(ctx, job)
	require.Error(t, err)

	after := promtestutil.ToFloat64(metrics.NotificationJobFailures)
	assert.Equal(t, before+1, after, "a Work() failure must increment metrics.NotificationJobFailures")
}

// TestNotificationWorker_InsertsNotificationRow verifies that Work() persists
// a notification row with the fields carried on the job args.
func TestNotificationWorker_InsertsNotificationRow(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	ctx := context.Background()

	teamID := uuid.New()
	actorID := uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Notify Team')`, teamID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Actor', 'actor@example.com', '#00ff00')`,
		actorID)
	require.NoError(t, err)

	title := "Neues Training"
	worker := jobs.NewNotificationWorker(pool)
	job := &river.Job[jobs.NotificationArgs]{
		JobRow: &rivertype.JobRow{ID: 1},
		Args: jobs.NotificationArgs{
			TeamID:  teamID,
			Type:    "news",
			ActorID: actorID,
			Title:   &title,
		},
	}

	require.NoError(t, worker.Work(ctx, job))

	var count int
	var gotTitle *string
	require.NoError(t, pool.QueryRow(
		ctx,
		`SELECT COUNT(*), MAX(title) FROM notifications WHERE team_id = $1`, teamID,
	).Scan(&count, &gotTitle))
	assert.Equal(t, 1, count)
	require.NotNil(t, gotTitle)
	assert.Equal(t, title, *gotTitle)
}

// TestNotificationWorker_Work_IsIdempotentOnRetry is a regression test for a
// worker whose doc comment claimed the insert was idempotent via River's
// job-state tracking, when in fact a bare INSERT with no unique key would
// create a duplicate row if River retried the job (its at-least-once
// delivery guarantee) after a crash between commit and job-completion ack.
func TestNotificationWorker_Work_IsIdempotentOnRetry(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	ctx := context.Background()

	teamID := uuid.New()
	actorID := uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Idempotent Notify Team')`, teamID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Actor2', 'actor2@example.com', '#00ff00')`,
		actorID)
	require.NoError(t, err)

	title := "Retried Notification"
	worker := jobs.NewNotificationWorker(pool)
	job := &river.Job[jobs.NotificationArgs]{
		JobRow: &rivertype.JobRow{ID: 424242},
		Args: jobs.NotificationArgs{
			TeamID:  teamID,
			Type:    "news",
			ActorID: actorID,
			Title:   &title,
		},
	}

	// Simulate River retrying the same job after a crash: Work() runs twice
	// with the same job.ID.
	require.NoError(t, worker.Work(ctx, job))
	require.NoError(t, worker.Work(ctx, job))

	var count int
	require.NoError(t, pool.QueryRow(
		ctx,
		`SELECT COUNT(*) FROM notifications WHERE team_id = $1`, teamID,
	).Scan(&count))
	assert.Equal(t, 1, count, "a retried job with the same ID must not create a duplicate notification row")
}

// TestNotificationWorker_Work_PushDelivery_GatesOnNewInsertOnly is a
// regression test: push delivery must only be attempted for a genuinely new
// notification row, not a retry that hit the ON CONFLICT DO NOTHING
// dedup path -- otherwise a retried job would risk pushing the same
// notification twice.
func TestNotificationWorker_Work_PushDelivery_GatesOnNewInsertOnly(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	ctx := context.Background()

	teamID := uuid.New()
	actorID := uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Push Gate Team')`, teamID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Actor3', 'actor3@example.com', '#00ff00')`,
		actorID)
	require.NoError(t, err)

	perms := &mockPermsChecker{perms: teams.PermissionsJSON{News: "read"}}
	lister := &mockSubscriptionLister{subs: []push.SubscriptionForUser{
		{Id: uuid.New(), UserId: uuid.New(), Subscription: push.Subscription{Endpoint: "https://push.example/x", P256dh: "p", AuthKey: "a"}},
	}}

	worker := jobs.NewNotificationWorker(pool).WithPushDelivery(perms, lister)
	job := &river.Job[jobs.NotificationArgs]{
		JobRow: &rivertype.JobRow{ID: 555},
		Args:   jobs.NotificationArgs{TeamID: teamID, Type: "news", ActorID: actorID},
	}

	// First attempt: a genuinely new row -- the gating logic must run (it
	// looks up subscriptions regardless of whether a river client is
	// present in ctx to actually enqueue with).
	require.NoError(t, worker.Work(ctx, job))
	assert.Equal(t, 1, lister.calledN, "a new notification must trigger a push-subscription lookup")

	// Second attempt with the same job ID: ON CONFLICT DO NOTHING means no
	// new row, so the gate must not run again.
	require.NoError(t, worker.Work(ctx, job))
	assert.Equal(t, 1, lister.calledN, "a retried (deduped) notification must not re-trigger push delivery")
}

// TestNotificationWorker_Work_PushDelivery_DisabledByDefault verifies that
// NewNotificationWorker (without WithPushDelivery) never touches
// perms/pushRepo -- the zero-value nil fields must be checked before use.
func TestNotificationWorker_Work_PushDelivery_DisabledByDefault(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	ctx := context.Background()

	teamID := uuid.New()
	actorID := uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'No Push Team')`, teamID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Actor4', 'actor4@example.com', '#00ff00')`,
		actorID)
	require.NoError(t, err)

	worker := jobs.NewNotificationWorker(pool)
	job := &river.Job[jobs.NotificationArgs]{
		JobRow: &rivertype.JobRow{ID: 556},
		Args:   jobs.NotificationArgs{TeamID: teamID, Type: "news", ActorID: actorID},
	}

	require.NoError(t, worker.Work(ctx, job), "push delivery being disabled must not affect the notification insert")
}

// TestNotificationWorker_Work_PushDelivery_SkipsWithoutRiverClientInContext
// verifies that calling Work() directly (as every other test in this file
// does, and as production code never does outside a real River-processed
// job) doesn't panic or error when push delivery is enabled but no
// river.Client is reachable via context -- ClientFromContextSafely must be
// used, not ClientFromContext (which panics).
func TestNotificationWorker_Work_PushDelivery_SkipsWithoutRiverClientInContext(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	ctx := context.Background()

	teamID := uuid.New()
	actorID := uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'No RC Team')`, teamID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Actor5', 'actor5@example.com', '#00ff00')`,
		actorID)
	require.NoError(t, err)

	perms := &mockPermsChecker{perms: teams.PermissionsJSON{News: "read"}}
	lister := &mockSubscriptionLister{subs: []push.SubscriptionForUser{
		{Id: uuid.New(), UserId: uuid.New(), Subscription: push.Subscription{Endpoint: "https://push.example/y", P256dh: "p", AuthKey: "a"}},
	}}

	worker := jobs.NewNotificationWorker(pool).WithPushDelivery(perms, lister)
	job := &river.Job[jobs.NotificationArgs]{
		JobRow: &rivertype.JobRow{ID: 557},
		Args:   jobs.NotificationArgs{TeamID: teamID, Type: "news", ActorID: actorID},
	}

	require.NotPanics(t, func() {
		require.NoError(t, worker.Work(ctx, job))
	})
}
