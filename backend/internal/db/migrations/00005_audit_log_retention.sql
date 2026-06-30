-- +goose NO TRANSACTION

-- +goose Up
-- +goose StatementBegin

-- BRIN index on occurred_at gives efficient time-range scans (compliance queries,
-- log shipping) at ~200x lower storage cost than a B-tree index. Effective because
-- audit_log rows are naturally ordered by insert time (correlates with occurred_at).
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_audit_log_occurred_at_brin
    ON audit_log USING BRIN (occurred_at);

-- Partial index speeds up actor-based lookups (e.g. "all actions by user X in the
-- last 90 days") without scanning unknown actors.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_audit_log_actor_occurred
    ON audit_log (actor_id, occurred_at DESC)
    WHERE actor_id IS NOT NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_audit_log_occurred_at_brin;
DROP INDEX IF EXISTS idx_audit_log_actor_occurred;
-- +goose StatementEnd
