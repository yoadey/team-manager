package jobs_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	promtestutil "github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
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
	// perms, keyed by userID, controls GetPermissions' return value per
	// subscriber; a userID with no entry gets the zero value of
	// teams.PermissionsJSON (every module ""), which
	// notifications.HasReadAccess treats as no access -- fail-closed default
	// for a member this mock wasn't told about, mirroring
	// mockSubscriptionLister.prefs' keyed-by-userID shape.
	perms   map[uuid.UUID]teams.PermissionsJSON
	err     error
	calledN int
}

func (m *mockPermsChecker) GetPermissions(_ context.Context, _ uuid.UUID, userID uuid.UUID) (teams.PermissionsJSON, error) {
	m.calledN++
	if m.err != nil {
		return teams.PermissionsJSON{}, m.err
	}
	return m.perms[userID], nil
}

type mockSubscriptionLister struct {
	subs    []push.SubscriptionForUser
	err     error
	calledN int

	// prefs, keyed by userID, controls GetPreferences' return value per
	// subscriber; a userID with no entry gets DefaultCategoryPreferences()
	// (everything enabled), matching a member who never saved preferences.
	prefs      map[uuid.UUID]push.CategoryPreferences
	prefsErr   error
	prefsCalls int
}

func (m *mockSubscriptionLister) ListForTeamExcludingUser(context.Context, uuid.UUID, uuid.UUID) ([]push.SubscriptionForUser, error) {
	m.calledN++
	return m.subs, m.err
}

func (m *mockSubscriptionLister) GetPreferences(_ context.Context, _, userID uuid.UUID) (push.CategoryPreferences, error) {
	m.prefsCalls++
	if m.prefsErr != nil {
		return push.CategoryPreferences{}, m.prefsErr
	}
	if p, ok := m.prefs[userID]; ok {
		return p, nil
	}
	return push.DefaultCategoryPreferences(), nil
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

// TestNotificationWorker_InsertsAttendanceStatus regression-tests that
// NotificationArgs.Status actually reaches the notifications.status column
// -- the field existed on the DB table and was already read back by
// notifications.Repository.ListByTeamAndUser, but nothing on the write path
// (NotificationArgs / this INSERT) carried it until events.Service.SetAttendance
// started enqueuing "attendance" notifications.
func TestNotificationWorker_InsertsAttendanceStatus(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	ctx := context.Background()

	teamID := uuid.New()
	actorID := uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Attendance Notify Team')`, teamID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Responder', 'responder@example.com', '#00ff00')`,
		actorID)
	require.NoError(t, err)

	status := "yes"
	eventTitle := "Training"
	worker := jobs.NewNotificationWorker(pool)
	job := &river.Job[jobs.NotificationArgs]{
		JobRow: &rivertype.JobRow{ID: 2},
		Args: jobs.NotificationArgs{
			TeamID:     teamID,
			Type:       "attendance",
			ActorID:    actorID,
			EventTitle: &eventTitle,
			Status:     &status,
		},
	}

	require.NoError(t, worker.Work(ctx, job))

	var gotType string
	var gotStatus *string
	require.NoError(t, pool.QueryRow(
		ctx,
		`SELECT type, status FROM notifications WHERE team_id = $1`, teamID,
	).Scan(&gotType, &gotStatus))
	assert.Equal(t, "attendance", gotType)
	require.NotNil(t, gotStatus)
	assert.Equal(t, "yes", *gotStatus)
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

	subscriberID := uuid.New()
	perms := &mockPermsChecker{perms: map[uuid.UUID]teams.PermissionsJSON{subscriberID: {News: "read"}}}
	lister := &mockSubscriptionLister{subs: []push.SubscriptionForUser{
		{Id: uuid.New(), UserId: subscriberID, Subscription: push.Subscription{Endpoint: "https://push.example/x", P256dh: "p", AuthKey: "a"}},
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

	subscriberID := uuid.New()
	perms := &mockPermsChecker{perms: map[uuid.UUID]teams.PermissionsJSON{subscriberID: {News: "read"}}}
	lister := &mockSubscriptionLister{subs: []push.SubscriptionForUser{
		{Id: uuid.New(), UserId: subscriberID, Subscription: push.Subscription{Endpoint: "https://push.example/y", P256dh: "p", AuthKey: "a"}},
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

// capturingPushDeliveryWorker stands in for the real PushDeliveryWorker in
// end-to-end preference-gate tests: it only records that a push-delivery job
// was actually enqueued and processed, via a buffered channel, without
// touching a real push service.
type capturingPushDeliveryWorker struct {
	river.WorkerDefaults[jobs.PushDeliveryArgs]
	fired chan jobs.PushDeliveryArgs
}

func (w *capturingPushDeliveryWorker) Work(_ context.Context, job *river.Job[jobs.PushDeliveryArgs]) error {
	w.fired <- job.Args
	return nil
}

// TestNotificationWorker_Work_PushDelivery_PreferenceGate is an end-to-end
// regression test (via a real river.Client, not a direct Work() call) for
// the preference gate added alongside per-team push-category preferences:
// enqueuePushDeliveries must skip a subscriber who has disabled the
// notification's category for that team, and still deliver to one who
// hasn't, even though both pass the same module-permission check.
func TestNotificationWorker_Work_PushDelivery_PreferenceGate(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	ctx := context.Background()
	require.NoError(t, jobs.MigrateRiver(ctx, pool))

	teamID := uuid.New()
	actorID := uuid.New()
	optedOutUser := uuid.New()
	optedInUser := uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Preference Gate Team')`, teamID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Actor6', 'actor6@example.com', '#00ff00')`,
		actorID)
	require.NoError(t, err)

	// Both subscribers pass the permission check identically here -- this test
	// isolates the preference gate, not the permission gate (see
	// TestNotificationWorker_Work_PushDelivery_PermissionGate for that).
	perms := &mockPermsChecker{perms: map[uuid.UUID]teams.PermissionsJSON{
		optedOutUser: {News: "read"},
		optedInUser:  {News: "read"},
	}}
	lister := &mockSubscriptionLister{
		subs: []push.SubscriptionForUser{
			{Id: uuid.New(), UserId: optedOutUser, Subscription: push.Subscription{Endpoint: "https://push.example/opted-out", P256dh: "p", AuthKey: "a"}},
			{Id: uuid.New(), UserId: optedInUser, Subscription: push.Subscription{Endpoint: "https://push.example/opted-in", P256dh: "p", AuthKey: "a"}},
		},
		prefs: map[uuid.UUID]push.CategoryPreferences{
			optedOutUser: {Attendance: true, Events: true, News: false, Polls: true, Absence: true},
			// optedInUser has no entry -> DefaultCategoryPreferences() (news enabled).
		},
	}

	notifWorker := jobs.NewNotificationWorker(pool).WithPushDelivery(perms, lister)
	fired := make(chan jobs.PushDeliveryArgs, 4)
	captureWorker := &capturingPushDeliveryWorker{fired: fired}

	workers := river.NewWorkers()
	river.AddWorker(workers, notifWorker)
	river.AddWorker(workers, captureWorker)

	rc, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		Queues:  map[string]river.QueueConfig{river.QueueDefault: {MaxWorkers: 2}},
		Workers: workers,
	})
	require.NoError(t, err)
	require.NoError(t, rc.Start(ctx))
	t.Cleanup(func() { _ = rc.Stop(context.Background()) })

	title := "Preference gate test"
	_, err = rc.Insert(ctx, jobs.NotificationArgs{TeamID: teamID, Type: "news", ActorID: actorID, Title: &title}, nil)
	require.NoError(t, err)

	// Exactly one delivery (the opted-in user's) must fire; the opted-out
	// user's must not, even though both pass the identical permission check.
	select {
	case got := <-fired:
		assert.Equal(t, "https://push.example/opted-in", got.Endpoint, "only the subscriber who hasn't disabled 'news' must receive a push")
	case <-time.After(10 * time.Second):
		t.Fatal("expected one push delivery for the opted-in subscriber, got none")
	}
	select {
	case got := <-fired:
		t.Fatalf("unexpected second push delivery to %q; the opted-out subscriber's 'news' preference must have suppressed it", got.Endpoint)
	case <-time.After(500 * time.Millisecond):
		// Expected: no further delivery.
	}
}

// TestNotificationWorker_Work_PushDelivery_PermissionGate is an end-to-end
// regression test (via a real river.Client, not a direct Work() call) for the
// module-permission gate itself: this is the property the package doc
// comments call "must not open a side channel around the same
// module-permission check the in-app feed already enforces" -- push delivery
// for one subscriber with events:read must fire while a second subscriber
// with events:none (the mockPermsChecker zero value for an unlisted user)
// must not, even though both hold an identical, allowing push-category
// preference. Before mockPermsChecker was keyed by userID, this scenario
// could not be expressed at all: every subscriber shared one perms value.
func TestNotificationWorker_Work_PushDelivery_PermissionGate(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	ctx := context.Background()
	require.NoError(t, jobs.MigrateRiver(ctx, pool))

	teamID := uuid.New()
	actorID := uuid.New()
	deniedUser := uuid.New()
	allowedUser := uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Permission Gate Team')`, teamID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Actor7', 'actor7@example.com', '#00ff00')`,
		actorID)
	require.NoError(t, err)

	// deniedUser has no entry -> teams.PermissionsJSON{} zero value -> every
	// module "" -> notifications.HasReadAccess denies it (fail-closed).
	perms := &mockPermsChecker{perms: map[uuid.UUID]teams.PermissionsJSON{
		allowedUser: {Events: "read"},
	}}
	lister := &mockSubscriptionLister{
		subs: []push.SubscriptionForUser{
			{Id: uuid.New(), UserId: deniedUser, Subscription: push.Subscription{Endpoint: "https://push.example/denied", P256dh: "p", AuthKey: "a"}},
			{Id: uuid.New(), UserId: allowedUser, Subscription: push.Subscription{Endpoint: "https://push.example/allowed", P256dh: "p", AuthKey: "a"}},
		},
		// No prefs entries for either user -> both get
		// DefaultCategoryPreferences() (events enabled), so the preference
		// gate can't be what's suppressing deniedUser's delivery.
	}

	notifWorker := jobs.NewNotificationWorker(pool).WithPushDelivery(perms, lister)
	fired := make(chan jobs.PushDeliveryArgs, 4)
	captureWorker := &capturingPushDeliveryWorker{fired: fired}

	workers := river.NewWorkers()
	river.AddWorker(workers, notifWorker)
	river.AddWorker(workers, captureWorker)

	rc, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		Queues:  map[string]river.QueueConfig{river.QueueDefault: {MaxWorkers: 2}},
		Workers: workers,
	})
	require.NoError(t, err)
	require.NoError(t, rc.Start(ctx))
	t.Cleanup(func() { _ = rc.Stop(context.Background()) })

	eventTitle := "Permission gate test"
	_, err = rc.Insert(ctx, jobs.NotificationArgs{TeamID: teamID, Type: "event_created", ActorID: actorID, EventTitle: &eventTitle}, nil)
	require.NoError(t, err)

	// Exactly one delivery (the allowed user's) must fire; the denied user's
	// must not, even though both share an identical, allowing push preference.
	select {
	case got := <-fired:
		assert.Equal(t, "https://push.example/allowed", got.Endpoint, "only the subscriber with events:read must receive a push")
	case <-time.After(10 * time.Second):
		t.Fatal("expected one push delivery for the allowed subscriber, got none")
	}
	select {
	case got := <-fired:
		t.Fatalf("unexpected second push delivery to %q; the denied subscriber's missing events permission must have suppressed it", got.Endpoint)
	case <-time.After(500 * time.Millisecond):
		// Expected: no further delivery.
	}
}
