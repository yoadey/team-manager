package jobs

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"go.opentelemetry.io/otel/codes"

	"github.com/yoadey/team-manager/backend/internal/metrics"
	"github.com/yoadey/team-manager/backend/internal/notifications"
	"github.com/yoadey/team-manager/backend/internal/push"
)

// EventReminderArgs are the (empty) arguments for the periodic
// event-reminder scan -- like RetentionArgs, this job carries no per-run
// data; every run re-derives its work from current DB state.
type EventReminderArgs struct{}

// Kind implements river.JobArgs.
func (EventReminderArgs) Kind() string { return "event_reminder" }

// eventReminderInterval is how often the periodic scan runs. Five minutes
// balances promptness (a reminder can fire up to this long after a member's
// configured lead time) against load; the daily RetentionWorker cadence is
// far too coarse for a "few hours before" reminder, and per-minute is
// unnecessary precision.
const eventReminderInterval = 5 * time.Minute

// eventReminderLookaheadSlack pads both ends of the window
// eventsReminderLister.ListUpcomingForReminders is queried with, to
// accommodate the timezone-aware conversion of an event's date/time fields
// into an absolute instant (an event dated "today" in UTC terms can already
// be "tomorrow" in the event's local timezone, or vice versa near
// midnight).
const eventReminderLookaheadSlack = 24 * time.Hour

// ReminderCandidate is a non-cancelled upcoming event, reduced to exactly
// what EventReminderWorker needs: its computed absolute start instant, not
// the raw date/time fields that instant is derived from. Deliberately not
// events.EventRow -- the events package already imports this package (to
// enqueue notification jobs via jobs.Client), so this package cannot import
// events back without an import cycle. cmd/server bridges the two via an
// adapter that calls events.EventStartInstant itself.
type ReminderCandidate struct {
	EventID uuid.UUID
	TeamID  uuid.UUID
	Title   string
	Start   time.Time
}

// eventsReminderLister is implemented by an adapter in cmd/server wrapping
// *events.Repository -- see ReminderCandidate's doc comment for why this
// package can't depend on *events.Repository directly.
type eventsReminderLister interface {
	ListUpcomingForReminders(ctx context.Context, from, to time.Time) ([]ReminderCandidate, error)
}

// reminderSubscriptionLister is satisfied by *push.Repository. Unlike
// subscriptionLister (used by NotificationWorker), ListForTeam excludes no
// one -- a time-triggered reminder has no "actor" to leave out.
type reminderSubscriptionLister interface {
	ListForTeam(ctx context.Context, teamID uuid.UUID) ([]push.SubscriptionForUser, error)
	GetPreferences(ctx context.Context, teamID, userID uuid.UUID) (push.CategoryPreferences, error)
}

// EventReminderWorker periodically scans for non-cancelled upcoming events
// and, for each team member who has enabled event reminders and currently
// has read access to the events module, sends a push notification once the
// member's configured lead time before the event's start has been reached.
// Idempotent across ticks and replicas via the event_reminders_sent table
// (see the migration adding it).
type EventReminderWorker struct {
	river.WorkerDefaults[EventReminderArgs]
	pool       *pgxpool.Pool
	eventsRepo eventsReminderLister
	pushRepo   reminderSubscriptionLister
	perms      permsChecker
}

// NewEventReminderWorker constructs an EventReminderWorker.
func NewEventReminderWorker(pool *pgxpool.Pool, eventsRepo eventsReminderLister, pushRepo reminderSubscriptionLister, perms permsChecker) *EventReminderWorker {
	return &EventReminderWorker{pool: pool, eventsRepo: eventsRepo, pushRepo: pushRepo, perms: perms}
}

// Timeout bounds a single run. Generous relative to the 5-minute interval
// since a run that overruns one tick simply delays the next, rather than
// stacking concurrently (River doesn't schedule a periodic job's next run
// until the previous one completes).
func (w *EventReminderWorker) Timeout(*river.Job[EventReminderArgs]) time.Duration {
	return 60 * time.Second
}

// Work is called by River on each periodic tick. It lists candidate events,
// then for each one, fans out to current team members, gating on module
// read permission, the member's own reminder preference, and an idempotency
// insert -- only a genuinely new event_reminders_sent row triggers an actual
// push enqueue.
//
// Per-member errors from the marker-insert/enqueue transaction (see
// remindMember) are collected via errors.Join and returned from Work, so
// River retries the whole tick instead of silently swallowing them --
// candidates already handled this run are safe to redo thanks to the
// event_reminders_sent ON CONFLICT DO NOTHING, since the marker and the
// enqueue now commit atomically (never one without the other).
func (w *EventReminderWorker) Work(ctx context.Context, _ *river.Job[EventReminderArgs]) (err error) {
	ctx, span := tracer.Start(ctx, "event_reminder.work")
	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()

	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	now := time.Now()
	from := now.Truncate(24 * time.Hour).Add(-eventReminderLookaheadSlack)
	to := now.Add(time.Duration(push.MaxEventReminderHoursBefore)*time.Hour + eventReminderLookaheadSlack)

	candidates, err := w.eventsRepo.ListUpcomingForReminders(ctx, from, to)
	if err != nil {
		metrics.EventReminderJobFailures.Inc()
		return fmt.Errorf("jobs.EventReminderWorker: list events: %w", err)
	}
	if len(candidates) == 0 {
		return nil
	}

	// River injects the client currently processing this job into ctx; a
	// direct Work() call outside a real queue (e.g. this worker's own unit
	// tests) has none, so there's nothing to enqueue push deliveries into --
	// mirrors NotificationWorker.enqueuePushDeliveries' identical guard.
	rc, err := river.ClientFromContextSafely[pgx.Tx](ctx)
	if err != nil {
		return nil
	}

	// Cached per team_id across candidates in this run -- multiple events
	// commonly belong to the same team, and membership rarely changes
	// within a single 5-minute tick.
	membersCache := map[uuid.UUID][]push.SubscriptionForUser{}
	permsCache := map[uuid.UUID]bool{}
	var errs []error
	for _, c := range candidates {
		if remindErr := w.remindForEvent(ctx, rc, c, now, membersCache, permsCache); remindErr != nil {
			errs = append(errs, remindErr)
		}
	}
	if len(errs) > 0 {
		metrics.EventReminderJobFailures.Inc()
		return fmt.Errorf("jobs.EventReminderWorker: %w", errors.Join(errs...))
	}
	return nil
}

// remindForEvent evaluates every current member of c's team against c's
// start instant and enqueues a push for each member whose reminder is due
// and hasn't already been sent. Errors from individual members' marker
// insert/enqueue transactions (remindMember) are collected via errors.Join
// rather than aborting the rest of the team -- a transient failure for one
// member shouldn't block reminders that are otherwise ready to send.
func (w *EventReminderWorker) remindForEvent(
	ctx context.Context,
	rc *river.Client[pgx.Tx],
	c ReminderCandidate,
	now time.Time,
	membersCache map[uuid.UUID][]push.SubscriptionForUser,
	permsCache map[uuid.UUID]bool,
) error {
	if !c.Start.After(now) {
		return nil
	}

	members, cached := membersCache[c.TeamID]
	if !cached {
		var err error
		members, err = w.pushRepo.ListForTeam(ctx, c.TeamID)
		if err != nil {
			slog.ErrorContext(ctx, "jobs.EventReminderWorker: list team subscriptions failed", "err", err, "team_id", c.TeamID)
			return nil
		}
		membersCache[c.TeamID] = members
	}

	var errs []error
	for _, m := range members {
		if !w.isAllowed(ctx, c.TeamID, m.UserId, permsCache) {
			continue
		}
		if err := w.remindMember(ctx, rc, c, m, now); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// isAllowed reports whether userID currently has read access to the events
// module in teamID, caching the result per (team, user) for the duration of
// one Work() run.
func (w *EventReminderWorker) isAllowed(ctx context.Context, teamID, userID uuid.UUID, permsCache map[uuid.UUID]bool) bool {
	permKey := permCacheKey(teamID, userID)
	allowed, ok := permsCache[permKey]
	if ok {
		return allowed
	}
	perms, err := w.perms.GetPermissions(ctx, teamID, userID)
	if err != nil {
		slog.ErrorContext(ctx, "jobs.EventReminderWorker: get permissions failed", "err", err)
		return false
	}
	allowed = notifications.HasReadAccess(perms, "events")
	permsCache[permKey] = allowed
	return allowed
}

// remindMember checks m's own reminder preference against c's start instant
// and, if due and not already sent, marks it sent and enqueues the push.
//
// The marker insert and the push-delivery enqueue happen inside a single DB
// transaction: either both commit or neither does. Doing them as two
// independent statements (the marker committed via w.pool, then a separate
// rc.Insert call) left a gap where a crash -- or a transient rc.Insert
// error, which used to be logged and swallowed here -- between the two could
// durably commit the marker without ever enqueuing the push. Because the
// marker's ON CONFLICT DO NOTHING is the only thing that makes retries safe,
// that gap permanently blocked the (event, user) reminder from ever being
// retried: nothing on a later tick recognizes "marked but never delivered".
// Wrapping both in one transaction removes the gap -- a failure anywhere
// before commit rolls back the marker too, so a retried tick naturally
// re-attempts both statements together, still protected against genuine
// duplicates by the same ON CONFLICT.
func (w *EventReminderWorker) remindMember(ctx context.Context, rc *river.Client[pgx.Tx], c ReminderCandidate, m push.SubscriptionForUser, now time.Time) error {
	prefs, err := w.pushRepo.GetPreferences(ctx, c.TeamID, m.UserId)
	if err != nil {
		slog.ErrorContext(ctx, "jobs.EventReminderWorker: get push preferences failed", "err", err)
		return nil
	}
	if !prefs.EventReminderEnabled {
		return nil
	}

	due := c.Start.Add(-time.Duration(prefs.EventReminderHoursBefore) * time.Hour)
	if now.Before(due) {
		return nil
	}

	tx, err := w.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("jobs.EventReminderWorker: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx, `
		INSERT INTO event_reminders_sent (event_id, user_id)
		VALUES ($1, $2)
		ON CONFLICT (event_id, user_id) DO NOTHING
	`, c.EventID, m.UserId)
	if err != nil {
		return fmt.Errorf("jobs.EventReminderWorker: mark reminder sent: %w", err)
	}
	if tag.RowsAffected() == 0 {
		// Already reminded for this (event, member) pair on an earlier tick
		// (or -- pre-fix -- a tick that committed the marker but crashed
		// before enqueuing; a fresh reminder can never be pieced together
		// for it now, but at least no *new* rows can end up in that state).
		return nil
	}

	payload := eventReminderPayload(c, prefs.EventReminderHoursBefore)
	if _, err := rc.InsertTx(ctx, tx, PushDeliveryArgs{
		SubscriptionID: m.Id,
		Endpoint:       m.Subscription.Endpoint,
		P256dh:         m.Subscription.P256dh,
		AuthKey:        m.Subscription.AuthKey,
		Title:          payload.Title,
		Body:           payload.Body,
	}, nil); err != nil {
		metrics.NotificationEnqueueFailures.Inc()
		return fmt.Errorf("jobs.EventReminderWorker: enqueue push delivery: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("jobs.EventReminderWorker: commit tx: %w", err)
	}
	return nil
}

// permCacheKey deterministically combines a team and user UUID into a
// single cache key without allocating a struct key type -- permsCache is
// keyed by (team, user) since a member's permissions are always evaluated
// per team, matching how NotificationWorker's own allowedCache is scoped
// (there it's fine to key by user alone since it only ever processes one
// team per Work() call; this worker processes many teams per run).
func permCacheKey(teamID, userID uuid.UUID) uuid.UUID {
	return uuid.NewSHA1(teamID, userID[:])
}

// eventReminderPayload builds the push notification title/body for an event
// reminder. Like pushPayloadForNotification, this has no i18n system
// available server-side (locale is a client-side-only preference) -- text
// uses the event's own title with a plain English label.
func eventReminderPayload(c ReminderCandidate, hoursBefore int16) push.Payload {
	return push.Payload{
		Title: "Event reminder",
		Body:  fmt.Sprintf("%s starts in about %dh", c.Title, hoursBefore),
	}
}
