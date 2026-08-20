package jobs_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivertest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yoadey/team-manager/backend/internal/jobs"
	"github.com/yoadey/team-manager/backend/internal/push"
	"github.com/yoadey/team-manager/backend/internal/teams"
	"github.com/yoadey/team-manager/backend/internal/testutil"
)

// ─── mocks ──────────────────────────────────────────────────────────────────

type mockEventsReminderLister struct {
	candidates []jobs.ReminderCandidate
	err        error
	calledN    int
}

func (m *mockEventsReminderLister) ListUpcomingForReminders(context.Context, time.Time, time.Time) ([]jobs.ReminderCandidate, error) {
	m.calledN++
	return m.candidates, m.err
}

type mockReminderSubscriptionLister struct {
	subs []push.SubscriptionForUser
	err  error

	// prefs, keyed by userID, controls GetPreferences' return value per
	// subscriber; a userID with no entry gets DefaultCategoryPreferences().
	prefs    map[uuid.UUID]push.CategoryPreferences
	prefsErr error
}

func (m *mockReminderSubscriptionLister) ListForTeam(context.Context, uuid.UUID) ([]push.SubscriptionForUser, error) {
	return m.subs, m.err
}

func (m *mockReminderSubscriptionLister) GetPreferences(_ context.Context, _, userID uuid.UUID) (push.CategoryPreferences, error) {
	if m.prefsErr != nil {
		return push.CategoryPreferences{}, m.prefsErr
	}
	if p, ok := m.prefs[userID]; ok {
		return p, nil
	}
	return push.DefaultCategoryPreferences(), nil
}

// ─── harness ────────────────────────────────────────────────────────────────
//
// capturingPushDeliveryWorker and mockPermsChecker are defined in
// notification_worker_test.go (same jobs_test package) and reused here.

// runReminderTick starts a real river.Client backed by pool, registers
// worker alongside a capturing push-delivery worker, enqueues a single
// EventReminderArgs tick, and waits for it to complete before returning --
// so callers can then make a deterministic assertion about what was (or
// wasn't) enqueued, rather than racing a fixed sleep against the worker.
func runReminderTick(t *testing.T, ctx context.Context, pool *pgxpool.Pool, worker *jobs.EventReminderWorker) chan jobs.PushDeliveryArgs {
	t.Helper()
	require.NoError(t, jobs.MigrateRiver(ctx, pool))

	fired := make(chan jobs.PushDeliveryArgs, 4)
	captureWorker := &capturingPushDeliveryWorker{fired: fired}

	workers := river.NewWorkers()
	river.AddWorker(workers, worker)
	river.AddWorker(workers, captureWorker)

	rc, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		Queues:  map[string]river.QueueConfig{river.QueueDefault: {MaxWorkers: 2}},
		Workers: workers,
	})
	require.NoError(t, err)
	require.NoError(t, rc.Start(ctx))
	t.Cleanup(func() { _ = rc.Stop(context.WithoutCancel(ctx)) })

	job, err := rc.Insert(ctx, jobs.EventReminderArgs{}, nil)
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		var state string
		err := pool.QueryRow(ctx, `SELECT state FROM river_job WHERE id = $1`, job.Job.ID).Scan(&state)
		return err == nil && state == "completed"
	}, 10*time.Second, 50*time.Millisecond, "event_reminder tick did not complete")

	return fired
}

func seedTeamUserEvent(t *testing.T, ctx context.Context, pool *pgxpool.Pool, teamID, eventID, userID uuid.UUID, title, email string, start time.Time) {
	t.Helper()
	_, err := pool.Exec(ctx, `INSERT INTO teams (id, name) VALUES ($1, 'Reminder Team')`, teamID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO users (id, name, email, avatar_color) VALUES ($1, 'Reminder User', $2, '#00ff00')`,
		userID, email)
	require.NoError(t, err)
	_, err = pool.Exec(ctx,
		`INSERT INTO events (id, team_id, type, title, date) VALUES ($1, $2, 'training', $3, $4)`,
		eventID, teamID, title, start)
	require.NoError(t, err)
}

// ─── tests ──────────────────────────────────────────────────────────────────

func TestEventReminderWorker_Work_SendsReminderWhenDue(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	ctx := context.Background()

	teamID, eventID, userID := uuid.New(), uuid.New(), uuid.New()
	start := time.Now().Add(5 * time.Hour) // default 6h lead time -> already due
	seedTeamUserEvent(t, ctx, pool, teamID, eventID, userID, "Due Event", "due@example.com", start)

	eventsRepo := &mockEventsReminderLister{candidates: []jobs.ReminderCandidate{
		{EventID: eventID, TeamID: teamID, Title: "Due Event", Start: start},
	}}
	pushRepo := &mockReminderSubscriptionLister{
		subs: []push.SubscriptionForUser{
			{Id: uuid.New(), UserId: userID, Subscription: push.Subscription{Endpoint: "https://push.example/due", P256dh: "p", AuthKey: "a"}},
		},
	}
	perms := &mockPermsChecker{perms: map[uuid.UUID]teams.PermissionsJSON{userID: {Events: "read"}}}
	worker := jobs.NewEventReminderWorker(pool, eventsRepo, pushRepo, perms)

	fired := runReminderTick(t, ctx, pool, worker)

	select {
	case got := <-fired:
		assert.Equal(t, "https://push.example/due", got.Endpoint)
	case <-time.After(10 * time.Second):
		t.Fatal("expected a reminder push, got none")
	}

	var count int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM event_reminders_sent WHERE event_id = $1 AND user_id = $2`, eventID, userID,
	).Scan(&count))
	assert.Equal(t, 1, count)
}

func TestEventReminderWorker_Work_SkipsWhenNotYetDue(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	ctx := context.Background()

	teamID, eventID, userID := uuid.New(), uuid.New(), uuid.New()
	start := time.Now().Add(48 * time.Hour) // default 6h lead time -> nowhere near due
	seedTeamUserEvent(t, ctx, pool, teamID, eventID, userID, "Far Off Event", "notyet@example.com", start)

	eventsRepo := &mockEventsReminderLister{candidates: []jobs.ReminderCandidate{
		{EventID: eventID, TeamID: teamID, Title: "Far Off Event", Start: start},
	}}
	pushRepo := &mockReminderSubscriptionLister{
		subs: []push.SubscriptionForUser{
			{Id: uuid.New(), UserId: userID, Subscription: push.Subscription{Endpoint: "https://push.example/notyet", P256dh: "p", AuthKey: "a"}},
		},
	}
	perms := &mockPermsChecker{perms: map[uuid.UUID]teams.PermissionsJSON{userID: {Events: "read"}}}
	worker := jobs.NewEventReminderWorker(pool, eventsRepo, pushRepo, perms)

	runReminderTick(t, ctx, pool, worker)

	var pushJobs int
	require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM river_job WHERE kind = 'push_delivery'`).Scan(&pushJobs))
	assert.Equal(t, 0, pushJobs, "a reminder not yet due must not enqueue a push")

	var count int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM event_reminders_sent WHERE event_id = $1 AND user_id = $2`, eventID, userID,
	).Scan(&count))
	assert.Equal(t, 0, count, "a reminder not yet due must not be marked sent")
}

func TestEventReminderWorker_Work_SkipsWhenPreferenceDisabled(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	ctx := context.Background()

	teamID, eventID, userID := uuid.New(), uuid.New(), uuid.New()
	start := time.Now().Add(1 * time.Hour) // well within any default lead time
	seedTeamUserEvent(t, ctx, pool, teamID, eventID, userID, "Opted Out Event", "optedout@example.com", start)

	eventsRepo := &mockEventsReminderLister{candidates: []jobs.ReminderCandidate{
		{EventID: eventID, TeamID: teamID, Title: "Opted Out Event", Start: start},
	}}
	pushRepo := &mockReminderSubscriptionLister{
		subs: []push.SubscriptionForUser{
			{Id: uuid.New(), UserId: userID, Subscription: push.Subscription{Endpoint: "https://push.example/optedout", P256dh: "p", AuthKey: "a"}},
		},
		prefs: map[uuid.UUID]push.CategoryPreferences{
			userID: {EventReminderEnabled: false, EventReminderHoursBefore: 6},
		},
	}
	perms := &mockPermsChecker{perms: map[uuid.UUID]teams.PermissionsJSON{userID: {Events: "read"}}}
	worker := jobs.NewEventReminderWorker(pool, eventsRepo, pushRepo, perms)

	runReminderTick(t, ctx, pool, worker)

	var pushJobs int
	require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM river_job WHERE kind = 'push_delivery'`).Scan(&pushJobs))
	assert.Equal(t, 0, pushJobs, "a member who disabled event reminders must not receive a push")

	var count int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM event_reminders_sent WHERE event_id = $1 AND user_id = $2`, eventID, userID,
	).Scan(&count))
	assert.Equal(t, 0, count, "a member who disabled event reminders must not be marked sent")
}

func TestEventReminderWorker_Work_SkipsWhenPermissionDenied(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	ctx := context.Background()

	teamID, eventID, userID := uuid.New(), uuid.New(), uuid.New()
	start := time.Now().Add(1 * time.Hour)
	seedTeamUserEvent(t, ctx, pool, teamID, eventID, userID, "No Access Event", "noaccess@example.com", start)

	eventsRepo := &mockEventsReminderLister{candidates: []jobs.ReminderCandidate{
		{EventID: eventID, TeamID: teamID, Title: "No Access Event", Start: start},
	}}
	pushRepo := &mockReminderSubscriptionLister{
		subs: []push.SubscriptionForUser{
			{Id: uuid.New(), UserId: userID, Subscription: push.Subscription{Endpoint: "https://push.example/noaccess", P256dh: "p", AuthKey: "a"}},
		},
	}
	perms := &mockPermsChecker{perms: map[uuid.UUID]teams.PermissionsJSON{userID: {Events: "none"}}}
	worker := jobs.NewEventReminderWorker(pool, eventsRepo, pushRepo, perms)

	runReminderTick(t, ctx, pool, worker)

	var pushJobs int
	require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM river_job WHERE kind = 'push_delivery'`).Scan(&pushJobs))
	assert.Equal(t, 0, pushJobs, "a member with events:none must not receive a push")

	var count int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM event_reminders_sent WHERE event_id = $1 AND user_id = $2`, eventID, userID,
	).Scan(&count))
	assert.Equal(t, 0, count, "a member with events:none must not be marked sent")
}

// TestEventReminderWorker_Work_NoCandidatesIsNoOp covers an empty candidate
// list (e.g. every upcoming event happens to be cancelled -- filtering that
// out is events.Repository.ListUpcomingForReminders' job, not this
// worker's, so this worker only needs to handle "nothing came back").
func TestEventReminderWorker_Work_NoCandidatesIsNoOp(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	ctx := context.Background()

	eventsRepo := &mockEventsReminderLister{candidates: nil}
	pushRepo := &mockReminderSubscriptionLister{}
	perms := &mockPermsChecker{}
	worker := jobs.NewEventReminderWorker(pool, eventsRepo, pushRepo, perms)

	runReminderTick(t, ctx, pool, worker)

	assert.Equal(t, 1, eventsRepo.calledN)
	var pushJobs int
	require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM river_job WHERE kind = 'push_delivery'`).Scan(&pushJobs))
	assert.Equal(t, 0, pushJobs)
}

func TestEventReminderWorker_Work_IdempotentAcrossRuns(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	ctx := context.Background()

	teamID, eventID, userID := uuid.New(), uuid.New(), uuid.New()
	start := time.Now().Add(1 * time.Hour)
	seedTeamUserEvent(t, ctx, pool, teamID, eventID, userID, "Idempotent Event", "idempotent-reminder@example.com", start)

	eventsRepo := &mockEventsReminderLister{candidates: []jobs.ReminderCandidate{
		{EventID: eventID, TeamID: teamID, Title: "Idempotent Event", Start: start},
	}}
	pushRepo := &mockReminderSubscriptionLister{
		subs: []push.SubscriptionForUser{
			{Id: uuid.New(), UserId: userID, Subscription: push.Subscription{Endpoint: "https://push.example/idempotent", P256dh: "p", AuthKey: "a"}},
		},
	}
	perms := &mockPermsChecker{perms: map[uuid.UUID]teams.PermissionsJSON{userID: {Events: "read"}}}
	worker := jobs.NewEventReminderWorker(pool, eventsRepo, pushRepo, perms)

	require.NoError(t, jobs.MigrateRiver(ctx, pool))
	fired := make(chan jobs.PushDeliveryArgs, 4)
	captureWorker := &capturingPushDeliveryWorker{fired: fired}
	workers := river.NewWorkers()
	river.AddWorker(workers, worker)
	river.AddWorker(workers, captureWorker)
	rc, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		Queues:  map[string]river.QueueConfig{river.QueueDefault: {MaxWorkers: 2}},
		Workers: workers,
	})
	require.NoError(t, err)
	require.NoError(t, rc.Start(ctx))
	t.Cleanup(func() { _ = rc.Stop(context.Background()) })

	// First tick sends the reminder.
	_, err = rc.Insert(ctx, jobs.EventReminderArgs{}, nil)
	require.NoError(t, err)
	select {
	case <-fired:
	case <-time.After(10 * time.Second):
		t.Fatal("expected the first tick to send a reminder")
	}

	// Second tick re-scans the same still-upcoming event; must not resend.
	job2, err := rc.Insert(ctx, jobs.EventReminderArgs{}, nil)
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		var state string
		err := pool.QueryRow(ctx, `SELECT state FROM river_job WHERE id = $1`, job2.Job.ID).Scan(&state)
		return err == nil && state == "completed"
	}, 10*time.Second, 50*time.Millisecond, "second event_reminder tick did not complete")

	select {
	case got := <-fired:
		t.Fatalf("expected no second reminder, got one for %s", got.Endpoint)
	default:
	}

	var count int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM event_reminders_sent WHERE event_id = $1 AND user_id = $2`, eventID, userID,
	).Scan(&count))
	assert.Equal(t, 1, count, "a repeated tick must not duplicate the event_reminders_sent row")
}

// TestEventReminderWorker_Work_MarkerNotOrphanedWhenEnqueueFails simulates a
// crash (or transient error) between the event_reminders_sent marker insert
// and the push-delivery enqueue: it deliberately skips jobs.MigrateRiver, so
// the app-level event_reminders_sent table exists (via testutil.NewTestDB's
// goose migrations) but the river_job table InsertTx needs does not. The
// marker insert therefore succeeds while the enqueue that follows in the
// same transaction fails with a genuine DB error -- exactly the gap the fix
// closes by doing both in one transaction. Before the fix this would leave
// a marker row with no corresponding push job, permanently blocking any
// future retry for this (event, user) pair (ON CONFLICT DO NOTHING skips
// it forever); after the fix, the failed enqueue rolls the marker insert
// back too, so a later retry starts from a clean slate.
func TestEventReminderWorker_Work_MarkerNotOrphanedWhenEnqueueFails(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	ctx := context.Background()

	teamID, eventID, userID := uuid.New(), uuid.New(), uuid.New()
	start := time.Now().Add(1 * time.Hour)
	seedTeamUserEvent(t, ctx, pool, teamID, eventID, userID, "Enqueue Fails Event", "enqueue-fails@example.com", start)

	eventsRepo := &mockEventsReminderLister{candidates: []jobs.ReminderCandidate{
		{EventID: eventID, TeamID: teamID, Title: "Enqueue Fails Event", Start: start},
	}}
	pushRepo := &mockReminderSubscriptionLister{
		subs: []push.SubscriptionForUser{
			{Id: uuid.New(), UserId: userID, Subscription: push.Subscription{Endpoint: "https://push.example/enqueue-fails", P256dh: "p", AuthKey: "a"}},
		},
	}
	perms := &mockPermsChecker{perms: teams.PermissionsJSON{Events: "read"}}
	worker := jobs.NewEventReminderWorker(pool, eventsRepo, pushRepo, perms)

	// Note: jobs.MigrateRiver is intentionally NOT called here -- see the
	// doc comment above for why that's what makes rc.InsertTx fail.
	workers := river.NewWorkers()
	river.AddWorker(workers, worker)
	rc, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		Queues:  map[string]river.QueueConfig{river.QueueDefault: {MaxWorkers: 2}},
		Workers: workers,
	})
	require.NoError(t, err)

	workCtx := rivertest.WorkContext(ctx, rc)
	job := &river.Job[jobs.EventReminderArgs]{Args: jobs.EventReminderArgs{}}

	err = worker.Work(workCtx, job)
	require.Error(t, err, "Work must surface the enqueue failure so River retries the whole tick, instead of swallowing it")

	var count int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM event_reminders_sent WHERE event_id = $1 AND user_id = $2`, eventID, userID,
	).Scan(&count))
	assert.Equal(t, 0, count, "the marker insert must roll back when the push-delivery enqueue fails in the same transaction, never left orphaned without its job")
}

// TestEventReminderWorker_Work_SkipsWithoutRiverClientInContext verifies
// that calling Work() directly (as production code never does outside a
// real River-processed job) doesn't panic when no river.Client is reachable
// via context -- and, since a reminder can only ever be delivered once
// (unlike NotificationWorker's per-notification-row idempotency), that it
// bails out before writing an event_reminders_sent row it could never
// actually deliver on.
func TestEventReminderWorker_Work_SkipsWithoutRiverClientInContext(t *testing.T) {
	t.Parallel()

	pool := testutil.NewTestDB(t)
	ctx := context.Background()

	teamID, eventID, userID := uuid.New(), uuid.New(), uuid.New()
	start := time.Now().Add(1 * time.Hour)
	seedTeamUserEvent(t, ctx, pool, teamID, eventID, userID, "No RC Event", "norc-reminder@example.com", start)

	eventsRepo := &mockEventsReminderLister{candidates: []jobs.ReminderCandidate{
		{EventID: eventID, TeamID: teamID, Title: "No RC Event", Start: start},
	}}
	pushRepo := &mockReminderSubscriptionLister{
		subs: []push.SubscriptionForUser{
			{Id: uuid.New(), UserId: userID, Subscription: push.Subscription{Endpoint: "https://push.example/norc", P256dh: "p", AuthKey: "a"}},
		},
	}
	perms := &mockPermsChecker{perms: map[uuid.UUID]teams.PermissionsJSON{userID: {Events: "read"}}}

	worker := jobs.NewEventReminderWorker(pool, eventsRepo, pushRepo, perms)
	job := &river.Job[jobs.EventReminderArgs]{Args: jobs.EventReminderArgs{}}

	require.NotPanics(t, func() {
		require.NoError(t, worker.Work(ctx, job))
	})

	var count int
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM event_reminders_sent WHERE event_id = $1 AND user_id = $2`, eventID, userID,
	).Scan(&count))
	assert.Equal(t, 0, count)
}
