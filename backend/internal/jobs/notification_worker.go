// Package jobs defines River background workers for the team-manager backend.
package jobs

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"

	"github.com/yoadey/team-manager/backend/internal/gen"
	"github.com/yoadey/team-manager/backend/internal/metrics"
	"github.com/yoadey/team-manager/backend/internal/notifications"
	"github.com/yoadey/team-manager/backend/internal/push"
	"github.com/yoadey/team-manager/backend/internal/teams"
)

// tracer is a no-op until observability.InitTracer sets the global
// TracerProvider (OTEL_EXPORTER_OTLP_ENDPOINT), same as everywhere else
// tracing is used in this codebase -- safe to use unconditionally.
var tracer = otel.Tracer("jobs")

// ─── Job args ────────────────────────────────────────────────────────────────

// NotificationArgs carries the data required to persist a single notification row.
type NotificationArgs struct {
	TeamID     uuid.UUID  `json:"team_id"`
	Type       string     `json:"type"` // "news" | "event_created" | "event_cancelled" | "poll"
	ActorID    uuid.UUID  `json:"actor_id"`
	EventID    *uuid.UUID `json:"event_id,omitempty"`
	EventTitle *string    `json:"event_title,omitempty"`
	EventDate  *time.Time `json:"event_date,omitempty"`
	Title      *string    `json:"title,omitempty"`
	Note       *string    `json:"note,omitempty"`
}

// Kind implements river.JobArgs — must be unique per worker type.
func (NotificationArgs) Kind() string { return "notification" }

// ─── Worker ───────────────────────────────────────────────────────────────────

// NotificationWorker inserts a notification row into Postgres.
// River provides at-least-once delivery (not exactly-once): the same job can
// run more than once if the process crashes after the INSERT commits but
// before River records the job as complete. The insert keys on job.ID (via
// the unique river_job_id column) with ON CONFLICT DO NOTHING to make a retry
// a no-op instead of creating a duplicate notification row.
type NotificationWorker struct {
	river.WorkerDefaults[NotificationArgs]
	pool *pgxpool.Pool

	// perms and pushRepo are nil unless WithPushDelivery was called (e.g. in
	// tests constructing a NotificationWorker directly via
	// NewNotificationWorker, which exercise only the insert). When both are
	// set, a successful insert also gates and enqueues Web Push deliveries.
	perms    permsChecker
	pushRepo subscriptionLister
}

// permsChecker returns the effective per-module permissions for a (team,
// user) pair -- satisfied by members.Repository, mirroring
// notifications.permChecker.
type permsChecker interface {
	GetPermissions(ctx context.Context, teamID, userID uuid.UUID) (teams.PermissionsJSON, error)
}

// subscriptionLister is satisfied by *push.Repository.
type subscriptionLister interface {
	ListForTeamExcludingUser(ctx context.Context, teamID, excludeUserID uuid.UUID) ([]push.SubscriptionForUser, error)
}

// NewNotificationWorker constructs a NotificationWorker backed by pool, with
// Web Push delivery disabled. Call WithPushDelivery to enable it.
func NewNotificationWorker(pool *pgxpool.Pool) *NotificationWorker {
	return &NotificationWorker{pool: pool}
}

// WithPushDelivery enables Web Push delivery: after a notification row is
// newly inserted, subscriptions belonging to current team members (other
// than the actor) are gated by perms/HasReadAccess and, if they pass,
// enqueued for delivery by PushDeliveryWorker. Returns w for chaining at
// construction time.
func (w *NotificationWorker) WithPushDelivery(perms permsChecker, pushRepo subscriptionLister) *NotificationWorker {
	w.perms = perms
	w.pushRepo = pushRepo
	return w
}

// Work is called by River for each job. It inserts a row into the
// notifications table and, when push delivery is enabled, enqueues a Web
// Push delivery job per qualifying subscription.
func (w *NotificationWorker) Work(ctx context.Context, job *river.Job[NotificationArgs]) (err error) {
	ctx, span := tracer.Start(ctx, "notification.work")
	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	a := job.Args
	tag, err := w.pool.Exec(ctx, `
		INSERT INTO notifications
		    (team_id, type, actor_id, event_id, event_title, event_date, title, note, river_job_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (river_job_id) WHERE river_job_id IS NOT NULL DO NOTHING
	`, a.TeamID, a.Type, a.ActorID, a.EventID, a.EventTitle, a.EventDate, a.Title, a.Note, job.ID)
	if err != nil {
		metrics.NotificationJobFailures.Inc()
		return fmt.Errorf("jobs.NotificationWorker: insert notification: %w", err)
	}

	// Only a genuinely new row (not a retry hitting ON CONFLICT DO NOTHING)
	// triggers push delivery -- a retry means either the first attempt
	// already enqueued deliveries before crashing, or another replica
	// processed it; re-enqueuing on every retry would risk duplicate pushes
	// for the same notification.
	if tag.RowsAffected() > 0 && w.perms != nil && w.pushRepo != nil {
		w.enqueuePushDeliveries(ctx, a)
	}
	return nil
}

// enqueuePushDeliveries looks up subscriptions for current team members
// (other than the actor), applies the same module-read-permission gate the
// in-app feed uses (notifications.NotificationModule/HasReadAccess), and
// enqueues a PushDeliveryArgs job per subscription that passes. Failures
// here are logged, not returned -- push delivery is best-effort and must
// never fail the notification row that already committed.
func (w *NotificationWorker) enqueuePushDeliveries(ctx context.Context, a NotificationArgs) {
	subs, err := w.pushRepo.ListForTeamExcludingUser(ctx, a.TeamID, a.ActorID)
	if err != nil {
		slog.ErrorContext(ctx, "jobs.NotificationWorker: list push subscriptions failed", "err", err)
		return
	}
	if len(subs) == 0 {
		return
	}

	// River injects the client currently processing this job into ctx; a
	// direct Work() call outside a real queue (e.g. NotificationWorker's own
	// unit tests) has none, so enqueuing is simply skipped rather than
	// failing the insert that already succeeded.
	rc, err := river.ClientFromContextSafely[pgx.Tx](ctx)
	if err != nil {
		return
	}

	module := notifications.NotificationModule(gen.NotificationType(a.Type))
	payload := pushPayloadForNotification(a)

	allowedCache := map[uuid.UUID]bool{}
	for _, s := range subs {
		allowed, cached := allowedCache[s.UserId]
		if !cached {
			perms, permErr := w.perms.GetPermissions(ctx, a.TeamID, s.UserId)
			if permErr != nil {
				slog.ErrorContext(ctx, "jobs.NotificationWorker: get permissions failed", "err", permErr)
				continue
			}
			allowed = notifications.HasReadAccess(perms, module)
			allowedCache[s.UserId] = allowed
		}
		if !allowed {
			continue
		}
		if _, insErr := rc.Insert(ctx, PushDeliveryArgs{
			SubscriptionID: s.Id,
			Endpoint:       s.Subscription.Endpoint,
			P256dh:         s.Subscription.P256dh,
			AuthKey:        s.Subscription.AuthKey,
			Title:          payload.Title,
			Body:           payload.Body,
			URL:            payload.URL,
		}, nil); insErr != nil {
			metrics.NotificationEnqueueFailures.Inc()
			slog.ErrorContext(ctx, "jobs.NotificationWorker: enqueue push delivery failed", "err", insErr)
		}
	}
}

// pushPayloadForNotification builds the push notification title/body from a
// job's fields. The backend has no i18n system (locale is a client-side-only
// preference, see frontend's useLocale) -- text uses the notification's own
// user-authored content where available (title/event title/note), falling
// back to a plain English label per type. Fully localized push text would
// need a stored per-user locale preference, which nothing today provides.
func pushPayloadForNotification(a NotificationArgs) push.Payload {
	label := "Notification"
	switch a.Type {
	case "news":
		label = "News"
	case "poll":
		label = "Poll"
	case "event_created":
		label = "New event"
	case "event_updated":
		label = "Event updated"
	case "event_cancelled":
		label = "Event cancelled"
	case "event_reactivated":
		label = "Event reactivated"
	case "event_deleted":
		label = "Event deleted"
	case "attendance":
		label = "Attendance update"
	}

	body := ""
	switch {
	case a.EventTitle != nil:
		body = *a.EventTitle
	case a.Title != nil:
		body = *a.Title
	case a.Note != nil:
		body = *a.Note
	}

	return push.Payload{Title: label, Body: body}
}

// ─── Client ───────────────────────────────────────────────────────────────────

// Client wraps river.Client[pgx.Tx] for enqueuing notification jobs from services.
type Client struct {
	rc *river.Client[pgx.Tx]
}

// PushDeps groups the dependencies needed to enable Web Push delivery.
// Pass nil to NewClient to run without Web Push (e.g. VAPID keys not
// configured) -- notifications are still inserted normally, just never
// pushed.
type PushDeps struct {
	Pusher push.Pusher
	Repo   *push.Repository
	Perms  permsChecker
}

// NewClient creates a River client backed by the given pool.
// retentionWorker is optional: pass nil to skip registering the retention job.
// pushDeps is optional: pass nil to run without Web Push delivery.
// Call Start() on the returned river.Client separately if running workers
// in the same process.
func NewClient(pool *pgxpool.Pool, retentionWorker *RetentionWorker, pushDeps *PushDeps) (client *Client, riverClient *river.Client[pgx.Tx], err error) {
	workers := river.NewWorkers()
	notifWorker := NewNotificationWorker(pool)
	if pushDeps != nil {
		notifWorker.WithPushDelivery(pushDeps.Perms, pushDeps.Repo)
		river.AddWorker(workers, NewPushDeliveryWorker(pushDeps.Pusher, pushDeps.Repo))
	}
	river.AddWorker(workers, notifWorker)

	var periodicJobs []*river.PeriodicJob
	if retentionWorker != nil {
		river.AddWorker(workers, retentionWorker)
		periodicJobs = append(periodicJobs, river.NewPeriodicJob(
			river.PeriodicInterval(24*time.Hour),
			func() (river.JobArgs, *river.InsertOpts) {
				return RetentionArgs{}, nil
			},
			&river.PeriodicJobOpts{RunOnStart: false},
		))
	}

	rc, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: 10},
		},
		Workers:         workers,
		PeriodicJobs:    periodicJobs,
		Logger:          slog.Default(),
		SoftStopTimeout: SoftStopTimeout,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("jobs.NewClient: %w", err)
	}
	return &Client{rc: rc}, rc, nil
}

// EnqueueNotification enqueues a notification job for delivery by the worker.
// The call is non-blocking; River persists the job to Postgres.
func (c *Client) EnqueueNotification(ctx context.Context, args NotificationArgs) error {
	_, err := c.rc.Insert(ctx, args, nil)
	if err != nil {
		metrics.NotificationEnqueueFailures.Inc()
		return fmt.Errorf("jobs.Client.EnqueueNotification: %w", err)
	}
	return nil
}

// SoftStopTimeout bounds how long river.Client.Stop waits for in-flight jobs
// to finish on their own before automatically escalating to a hard stop
// (cancelling their contexts) -- see river.Config.SoftStopTimeout's doc
// comment. Without it, Stop's own ctx timing out just makes Stop return an
// error early; it does NOT cancel the still-running job, which keeps
// executing (and holding a pool connection) for up to its own Timeout()
// budget -- RetentionWorker's is 150s. cmd/server/main.go's SIGTERM handler
// budgets its riverStopCtx around this constant with margin so Stop()
// reliably returns near this bound instead of racing pool.Close() against
// whichever job happened to be mid-run.
const SoftStopTimeout = 8 * time.Second

// riverMigrationLockKey is an arbitrary, fixed advisory-lock key used only to
// serialize MigrateRiver across processes -- distinct from any other
// advisory lock key used elsewhere in the codebase (those are all derived
// via hashtextextended on a UUID/string, not a plain integer literal).
const riverMigrationLockKey int64 = 720481920

// MigrateRiver runs River's built-in schema migrations against the given pool.
// It must be called once during startup, after the application schema migrations.
//
// Unlike db.RunMigrations (which passes goose a WithSessionLocker so
// concurrent Up() calls across processes serialize on a Postgres session
// advisory lock), River's own migrator (rivermigrate.Config) has no
// equivalent locking option. Without a lock here, several replicas
// migrating concurrently -- the Helm chart's per-pod migrate initContainer
// running on multiple pods during a rolling update, HPA scale-out, or a
// fresh multi-replica install -- can race on the same DDL: River's
// river_migration/river_job tables are created without IF NOT EXISTS and
// the migration-version bookkeeping insert has no ON CONFLICT, so a losing
// racer fails with a duplicate-object error and crash-loops until a retry
// sees the migration as already applied. Acquiring a dedicated connection
// and holding a session-level pg_advisory_lock for the duration mirrors
// goose's own approach and closes that race the same way.
func MigrateRiver(ctx context.Context, pool *pgxpool.Pool) error {
	// The lock connection is dialed directly (pgx.ConnectConfig), NOT drawn
	// via pool.Acquire, so it sits outside the shared pool's tracked
	// capacity entirely. migrator.Migrate below needs its own connection(s)
	// from that same pool to actually run -- if the lock instead held one
	// of the pool's own connections, N concurrent replicas each holding a
	// pool connection just to wait on the lock can exhaust the pool's
	// capacity before the lock's current holder ever gets a connection to
	// do the migration work itself, deadlocking all of them permanently
	// (caught by a concurrency regression test timing out in CI).
	connConfig := pool.Config().ConnConfig.Copy()
	lockConn, err := pgx.ConnectConfig(ctx, connConfig)
	if err != nil {
		return fmt.Errorf("jobs.MigrateRiver: dial lock connection: %w", err)
	}
	defer func() { _ = lockConn.Close(context.WithoutCancel(ctx)) }()

	if _, err := lockConn.Exec(ctx, `SELECT pg_advisory_lock($1)`, riverMigrationLockKey); err != nil {
		return fmt.Errorf("jobs.MigrateRiver: acquire advisory lock: %w", err)
	}
	// Unlock must still run even if ctx was cancelled/timed out by the time
	// Migrate returns -- otherwise the lock stays held until this dedicated
	// connection is closed (see the Close above), which happens regardless,
	// but releasing it explicitly first lets a waiting replica proceed
	// immediately rather than waiting on this connection's teardown too.
	defer func() {
		_, _ = lockConn.Exec(context.WithoutCancel(ctx), `SELECT pg_advisory_unlock($1)`, riverMigrationLockKey)
	}()

	migrator, err := rivermigrate.New(riverpgxv5.New(pool), nil)
	if err != nil {
		return fmt.Errorf("jobs.MigrateRiver: %w", err)
	}
	if _, err := migrator.Migrate(ctx, rivermigrate.DirectionUp, nil); err != nil {
		return fmt.Errorf("jobs.MigrateRiver: %w", err)
	}
	return nil
}
