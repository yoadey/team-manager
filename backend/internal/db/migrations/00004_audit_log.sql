-- +goose NO TRANSACTION
-- +goose Up
-- +goose StatementBegin

-- Persistent audit log for security-sensitive operations.
-- Records are immutable (no UPDATE/DELETE permissions should be granted to the
-- application role); retention and archival are handled at the infrastructure
-- layer (e.g. pg_partman, log shipping, or a separate SIEM export job).
CREATE TABLE audit_log (
    id         BIGSERIAL    PRIMARY KEY,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    event      TEXT         NOT NULL,
    outcome    TEXT         NOT NULL CHECK (outcome IN ('success', 'failure')),
    actor_id   TEXT,
    attrs      JSONB        NOT NULL DEFAULT '{}'
);

-- Index for time-range queries (compliance review, incident investigation).
CREATE INDEX CONCURRENTLY audit_log_occurred_at_idx ON audit_log (occurred_at DESC);

-- Index for per-actor audit queries (e.g. "all actions by user X").
CREATE INDEX CONCURRENTLY audit_log_actor_id_idx ON audit_log (actor_id) WHERE actor_id IS NOT NULL;

-- Index for event-type filtering (e.g. "all login failures in the last 7 days").
CREATE INDEX CONCURRENTLY audit_log_event_idx ON audit_log (event, occurred_at DESC);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS audit_log;
-- +goose StatementEnd
