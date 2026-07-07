-- +goose NO TRANSACTION

-- +goose Up

-- jobs.RetentionWorker deletes notifications older than
-- RETENTION_NOTIFICATIONS_DAYS via "DELETE ... WHERE created_at < $1", which
-- has no team_id predicate. The only existing index,
-- idx_notifications_team_created (team_id, created_at DESC), can't be used
-- for a scan with no team_id filter, so every batch of the daily retention
-- job's loop does a full sequential scan of the table. A BRIN index on
-- created_at (rows are naturally insert-ordered by their own created_at, and
-- BRIN is cheap to maintain on an append-mostly table) gives it an efficient
-- time-range scan, matching the same pattern already used for audit_log's
-- retention query (see 00005_audit_log_retention.sql).
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_notifications_created_at_brin
    ON notifications USING BRIN (created_at);

-- +goose Down

DROP INDEX IF EXISTS idx_notifications_created_at_brin;
