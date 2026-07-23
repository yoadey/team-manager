package jobs

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"go.opentelemetry.io/otel/codes"

	"github.com/yoadey/team-manager/backend/internal/metrics"
)

// RetentionArgs are the arguments for the daily retention cleanup job.
type RetentionArgs struct{}

func (RetentionArgs) Kind() string { return "retention" }

// inviteRetention is how long past its (fixed, 7-day) expiry an invite row
// is kept before the retention job deletes it. Unlike notifications/
// sessions/audit_log, this isn't independently configurable: expired invites
// are inherently disposable (the code can never be redeemed again), so a
// grace period matching sessions' default is generous enough without adding
// another env var for a low-stakes cleanup. teams.Service.CreateInvite is
// called every time the invite sheet is opened with no reuse of unexpired
// codes, so this table grows unboundedly without this cleanup.
const inviteRetention = 30 * 24 * time.Hour

// RetentionWorker deletes stale rows from notifications, sessions, invites,
// audit_log, and never-verified user accounts. Thresholds are configured at
// construction time; sensible defaults (90 / 30 / 365 / 7 days) are applied
// when zero values are passed.
type RetentionWorker struct {
	river.WorkerDefaults[RetentionArgs]
	pool                       *pgxpool.Pool
	notificationRetention      time.Duration
	sessionRetention           time.Duration
	auditLogRetention          time.Duration
	unverifiedAccountRetention time.Duration
}

// NewRetentionWorker constructs a RetentionWorker. notifDays, sessionDays,
// auditLogDays, and unverifiedAccountDays control how old a row must be
// before it is deleted. Pass 0 to use the defaults (90, 30, 365, and 7 days
// respectively).
func NewRetentionWorker(pool *pgxpool.Pool, notifDays, sessionDays, auditLogDays, unverifiedAccountDays int) *RetentionWorker {
	if notifDays <= 0 {
		notifDays = 90
	}
	if sessionDays <= 0 {
		sessionDays = 30
	}
	if auditLogDays <= 0 {
		auditLogDays = 365
	}
	if unverifiedAccountDays <= 0 {
		unverifiedAccountDays = 7
	}
	return &RetentionWorker{
		pool:                       pool,
		notificationRetention:      time.Duration(notifDays) * 24 * time.Hour,
		sessionRetention:           time.Duration(sessionDays) * 24 * time.Hour,
		auditLogRetention:          time.Duration(auditLogDays) * 24 * time.Hour,
		unverifiedAccountRetention: time.Duration(unverifiedAccountDays) * 24 * time.Hour,
	}
}

// retentionBatchSize caps each DELETE statement issued by deleteBatched. A
// single unbounded DELETE on a large table can hold its lock and generate a
// burst of WAL for the entire duration of the statement; deleting in bounded
// batches keeps each transaction short so it doesn't block concurrent reads
// or writes on the table for long.
const retentionBatchSize = 1000

// deleteBatched repeatedly deletes up to retentionBatchSize rows where
// dateColumn is older than cutoff, looping until fewer than a full batch is
// removed (i.e. the table is exhausted). It returns the total number of rows
// deleted. table and dateColumn are always fixed, internal literals (never
// derived from request input), so building the query string is safe.
func deleteBatched(ctx context.Context, pool *pgxpool.Pool, table, dateColumn string, cutoff time.Time) (int64, error) {
	query := fmt.Sprintf(
		`DELETE FROM %s WHERE ctid IN (SELECT ctid FROM %s WHERE %s < $1 LIMIT %d)`,
		table, table, dateColumn, retentionBatchSize,
	)

	var total int64
	for {
		tag, err := pool.Exec(ctx, query, cutoff)
		if err != nil {
			return total, err
		}
		total += tag.RowsAffected()
		if tag.RowsAffected() < retentionBatchSize {
			return total, nil
		}
	}
}

// deleteUnverifiedUsers repeatedly deletes up to retentionBatchSize rows from
// users where email_verified_at IS NULL and created_at is older than cutoff,
// looping until fewer than a full batch is removed. Mirrors deleteBatched's
// batching strategy, but needs its own query since deleteBatched only
// supports a single "dateColumn < cutoff" condition, not the additional
// email_verified_at IS NULL guard that distinguishes an abandoned
// registration from a long-lived verified account.
func deleteUnverifiedUsers(ctx context.Context, pool *pgxpool.Pool, cutoff time.Time) (int64, error) {
	query := fmt.Sprintf(
		`DELETE FROM users WHERE ctid IN (SELECT ctid FROM users WHERE email_verified_at IS NULL AND created_at < $1 LIMIT %d)`,
		retentionBatchSize,
	)

	var total int64
	for {
		tag, err := pool.Exec(ctx, query, cutoff)
		if err != nil {
			return total, err
		}
		total += tag.RowsAffected()
		if tag.RowsAffected() < retentionBatchSize {
			return total, nil
		}
	}
}

// retentionPhaseTimeout bounds each of the six delete phases in Work
// independently. A single shared timeout for the whole run would let an
// unusually large backlog in one table (e.g. notifications, always deleted
// first) exhaust the entire budget and starve the phases after it -- since
// the ordering is fixed, that would silently and repeatedly block
// audit_log's compliance-mandated cleanup on every run until an operator
// intervenes.
const retentionPhaseTimeout = 30 * time.Second

// Timeout overrides river.WorkerDefaults' zero-value Timeout(), which would
// otherwise fall back to River's own JobTimeoutDefault (1 minute, see
// jobs.NewClient -- river.Config.JobTimeout is left unset). That outer
// per-job context deadline caps every phase's own context.WithTimeout below
// it (a context's deadline can only be tightened, never loosened), so six
// sequential retentionPhaseTimeout budgets would be squeezed into a shared
// ~60s window, starving whichever phases run last -- including audit_log's
// compliance-mandated cleanup -- exactly the failure mode
// retentionPhaseTimeout's own comment describes, just via the outer River
// timeout instead of one phase hogging its own inner one. Budgeting for all
// six phases' full timeout plus margin ensures the outer deadline is never
// the binding constraint.
func (w *RetentionWorker) Timeout(*river.Job[RetentionArgs]) time.Duration {
	return 6*retentionPhaseTimeout + 30*time.Second
}

// Work is called by River once per scheduled run. It deletes old notifications
// and expired sessions from the database.
//
// Each of the six phases below runs regardless of whether an earlier phase
// failed or timed out, and their errors are joined at the end rather than
// returned immediately -- returning early on the first error would let an
// unusually large backlog in one table (e.g. notifications, always deleted
// first) that exceeds its own retentionPhaseTimeout silently abort every
// later phase on every run, including audit_log's compliance-mandated
// cleanup, defeating the whole point of giving each phase an independent
// timeout budget (see retentionPhaseTimeout's doc comment) by introducing
// the same starvation failure mode through a different mechanism.
func (w *RetentionWorker) Work(ctx context.Context, _ *river.Job[RetentionArgs]) (err error) {
	ctx, span := tracer.Start(ctx, "retention.work")
	defer func() {
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		span.End()
	}()

	now := time.Now()
	var errs []error

	// Delete old notifications.
	notifCtx, notifCancel := context.WithTimeout(ctx, retentionPhaseTimeout)
	notifCutoff := now.Add(-w.notificationRetention)
	notifRows, notifErr := deleteBatched(notifCtx, w.pool, "notifications", "created_at", notifCutoff)
	notifCancel()
	if notifErr != nil {
		metrics.RetentionJobFailures.WithLabelValues("notifications").Inc()
		errs = append(errs, fmt.Errorf("retention: delete notifications: %w", notifErr))
	} else {
		metrics.RetentionJobRowsDeleted.WithLabelValues("notifications").Add(float64(notifRows))
		slog.Info("retention: deleted old notifications", "rows", notifRows, "cutoff", notifCutoff)
	}

	// Delete sessions that expired more than sessionRetention ago. Keying off
	// expires_at (not created_at) is required for correctness: a long-lived
	// session (e.g. SESSION_TTL_HOURS raised above RETENTION_SESSIONS_DAYS*24)
	// must never be purged while it's still valid, only once it has actually
	// expired and the retention grace period has passed.
	sessionCtx, sessionCancel := context.WithTimeout(ctx, retentionPhaseTimeout)
	sessionCutoff := now.Add(-w.sessionRetention)
	sessionRows, sessionErr := deleteBatched(sessionCtx, w.pool, "sessions", "expires_at", sessionCutoff)
	sessionCancel()
	if sessionErr != nil {
		metrics.RetentionJobFailures.WithLabelValues("sessions").Inc()
		errs = append(errs, fmt.Errorf("retention: delete sessions: %w", sessionErr))
	} else {
		metrics.RetentionJobRowsDeleted.WithLabelValues("sessions").Add(float64(sessionRows))
		slog.Info("retention: deleted old sessions", "rows", sessionRows, "cutoff", sessionCutoff)
	}

	// Delete invites that expired more than inviteRetention ago. Keyed off
	// expires_at, same reasoning as sessions above.
	inviteCtx, inviteCancel := context.WithTimeout(ctx, retentionPhaseTimeout)
	inviteCutoff := now.Add(-inviteRetention)
	inviteRows, inviteErr := deleteBatched(inviteCtx, w.pool, "invites", "expires_at", inviteCutoff)
	inviteCancel()
	if inviteErr != nil {
		metrics.RetentionJobFailures.WithLabelValues("invites").Inc()
		errs = append(errs, fmt.Errorf("retention: delete invites: %w", inviteErr))
	} else {
		metrics.RetentionJobRowsDeleted.WithLabelValues("invites").Add(float64(inviteRows))
		slog.Info("retention: deleted expired invites", "rows", inviteRows, "cutoff", inviteCutoff)
	}

	// Delete old audit_log entries. Compliance regulations typically require a
	// minimum retention period (e.g. 1 year); RETENTION_AUDIT_LOG_DAYS defaults
	// to 365. The index on occurred_at makes the time-range scan behind each
	// batch efficient.
	auditCtx, auditCancel := context.WithTimeout(ctx, retentionPhaseTimeout)
	auditCutoff := now.Add(-w.auditLogRetention)
	auditRows, auditErr := deleteBatched(auditCtx, w.pool, "audit_log", "occurred_at", auditCutoff)
	auditCancel()
	if auditErr != nil {
		metrics.RetentionJobFailures.WithLabelValues("audit_log").Inc()
		errs = append(errs, fmt.Errorf("retention: delete audit_log: %w", auditErr))
	} else {
		metrics.RetentionJobRowsDeleted.WithLabelValues("audit_log").Add(float64(auditRows))
		slog.Info("retention: deleted old audit_log entries", "rows", auditRows, "cutoff", auditCutoff)
	}

	// Delete accounts that never completed email verification, once older
	// than unverifiedAccountRetention -- otherwise an attacker (or a user who
	// abandons signup) can squat an email address indefinitely, since
	// users.email is UNIQUE and Service.Register never overwrites an
	// existing row. Deleting the users row cascades to
	// email_verification_tokens (ON DELETE CASCADE) and sessions, so no
	// separate cleanup is needed for either of those on this path.
	unverifiedCtx, unverifiedCancel := context.WithTimeout(ctx, retentionPhaseTimeout)
	unverifiedCutoff := now.Add(-w.unverifiedAccountRetention)
	unverifiedRows, unverifiedErr := deleteUnverifiedUsers(unverifiedCtx, w.pool, unverifiedCutoff)
	unverifiedCancel()
	if unverifiedErr != nil {
		metrics.RetentionJobFailures.WithLabelValues("users").Inc()
		errs = append(errs, fmt.Errorf("retention: delete unverified users: %w", unverifiedErr))
	} else {
		metrics.RetentionJobRowsDeleted.WithLabelValues("users").Add(float64(unverifiedRows))
		slog.Info("retention: deleted never-verified accounts", "rows", unverifiedRows, "cutoff", unverifiedCutoff)
	}

	// Delete verification tokens that have simply expired, independent of the
	// unverifiedAccountRetention grace period above -- e.g. a verified user's
	// stale token, or an unverified user's earlier token superseded by a
	// resend. Unlike sessions/invites there's no reason to keep an expired,
	// unusable token around at all, so the cutoff is "now" rather than a
	// further grace window.
	tokenCtx, tokenCancel := context.WithTimeout(ctx, retentionPhaseTimeout)
	tokenRows, tokenErr := deleteBatched(tokenCtx, w.pool, "email_verification_tokens", "expires_at", now)
	tokenCancel()
	if tokenErr != nil {
		metrics.RetentionJobFailures.WithLabelValues("email_verification_tokens").Inc()
		errs = append(errs, fmt.Errorf("retention: delete expired email_verification_tokens: %w", tokenErr))
	} else {
		metrics.RetentionJobRowsDeleted.WithLabelValues("email_verification_tokens").Add(float64(tokenRows))
		slog.Info("retention: deleted expired email verification tokens", "rows", tokenRows, "cutoff", now)
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	metrics.RetentionJobLastSuccessTimestamp.Set(float64(now.Unix()))
	return nil
}
