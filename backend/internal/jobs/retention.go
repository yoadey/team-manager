package jobs

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
)

// RetentionArgs are the arguments for the daily retention cleanup job.
type RetentionArgs struct{}

func (RetentionArgs) Kind() string { return "retention" }

// RetentionWorker deletes stale rows from notifications, sessions, and audit_log.
// Thresholds are configured at construction time; sensible defaults
// (90 / 30 / 365 days) are applied when zero values are passed.
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

// Work is called by River once per scheduled run. It deletes old notifications
// and expired sessions from the database.
func (w *RetentionWorker) Work(ctx context.Context, _ *river.Job[RetentionArgs]) error {
	now := time.Now()

	// Delete old notifications.
	notifCutoff := now.Add(-w.notificationRetention)
	tag, err := w.pool.Exec(ctx,
		"DELETE FROM notifications WHERE created_at < $1", notifCutoff)
	if err != nil {
		return fmt.Errorf("retention: delete notifications: %w", err)
	}
	slog.Info("retention: deleted old notifications", "rows", tag.RowsAffected(), "cutoff", notifCutoff)

	// Delete expired sessions. The sessions table may not exist in all
	// environments, so a failure here is logged as a warning only.
	sessionCutoff := now.Add(-w.sessionRetention)
	tag, err = w.pool.Exec(ctx,
		"DELETE FROM sessions WHERE created_at < $1", sessionCutoff)
	if err != nil {
		slog.Warn("retention: delete sessions skipped (table may not exist)", "err", err)
	} else {
		slog.Info("retention: deleted old sessions", "rows", tag.RowsAffected(), "cutoff", sessionCutoff)
	}

	// Delete old audit_log entries. Compliance regulations typically require a
	// minimum retention period (e.g. 1 year); RETENTION_AUDIT_LOG_DAYS defaults
	// to 365. Rows older than the cutoff are deleted in a single statement; the
	// BRIN index on occurred_at makes time-range scans efficient.
	auditCutoff := now.Add(-w.auditLogRetention)
	tag, err = w.pool.Exec(ctx,
		"DELETE FROM audit_log WHERE occurred_at < $1", auditCutoff)
	if err != nil {
		return fmt.Errorf("retention: delete audit_log: %w", err)
	}
	slog.Info("retention: deleted old audit_log entries", "rows", tag.RowsAffected(), "cutoff", auditCutoff)

	return nil
}
