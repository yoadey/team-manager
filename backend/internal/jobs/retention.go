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
// and audit_log. Thresholds for the first/second/fourth are configured at
// construction time; sensible defaults (90 / 30 / 365 days) are applied when
// zero values are passed.
type RetentionWorker struct {
	river.WorkerDefaults[RetentionArgs]
	pool                  *pgxpool.Pool
	notificationRetention time.Duration
	sessionRetention      time.Duration
	auditLogRetention     time.Duration
}

// NewRetentionWorker constructs a RetentionWorker. notifDays, sessionDays, and
// auditLogDays control how old a row must be before it is deleted. Pass 0 to
// use the defaults (90, 30, and 365 days respectively).
func NewRetentionWorker(pool *pgxpool.Pool, notifDays, sessionDays, auditLogDays int) *RetentionWorker {
	if notifDays <= 0 {
		notifDays = 90
	}
	if sessionDays <= 0 {
		sessionDays = 30
	}
	if auditLogDays <= 0 {
		auditLogDays = 365
	}
	return &RetentionWorker{
		pool:                  pool,
		notificationRetention: time.Duration(notifDays) * 24 * time.Hour,
		sessionRetention:      time.Duration(sessionDays) * 24 * time.Hour,
		auditLogRetention:     time.Duration(auditLogDays) * 24 * time.Hour,
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

// retentionPhaseTimeout bounds each of the four delete phases in Work
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
// it (a context's deadline can only be tightened, never loosened), so four
// sequential retentionPhaseTimeout budgets would be squeezed into a shared
// ~60s window, starving whichever phases run last -- including audit_log's
// compliance-mandated cleanup -- exactly the failure mode
// retentionPhaseTimeout's own comment describes, just via the outer River
// timeout instead of one phase hogging its own inner one. Budgeting for all
// four phases' full timeout plus margin ensures the outer deadline is never
// the binding constraint.
func (w *RetentionWorker) Timeout(*river.Job[RetentionArgs]) time.Duration {
	return 4*retentionPhaseTimeout + 30*time.Second
}

// Work is called by River once per scheduled run. It deletes old notifications
// and expired sessions from the database.
//
// Each of the four phases below runs regardless of whether an earlier phase
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
	//
	// Note: migration 00004_audit_log.sql's header comment says retention is
	// "handled at the infrastructure layer" and recommends granting the app
	// role no UPDATE/DELETE on this table -- that was the original design
	// intent, superseded by this job actually owning retention instead (no
	// migration ever revokes DELETE, so nothing is broken today, but an
	// operator who follows that comment's advice literally would silently
	// break this job every day going forward). Left as-is rather than editing
	// the historical migration file, since goose validates applied
	// migrations' checksums against their file content.
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

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	metrics.RetentionJobLastSuccessTimestamp.Set(float64(now.Unix()))
	return nil
}
