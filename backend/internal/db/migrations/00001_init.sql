-- Initial-setup migration. Teamverwaltung has only ever shipped under an
-- `alpha` tag and has never been deployed anywhere real, so this is written
-- as the schema a fresh install needs from day one rather than as the first
-- of an incremental history — there is no live deployment whose migration
-- history (or legacy data) needs to be preserved. See
-- openspec/changes/alpha-initial-setup for the change that squashed the
-- prior 00001-00029 sequence into this single file.

-- +goose NO TRANSACTION
-- +goose Up
-- +goose StatementBegin

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ── Core Entities ──────────────────────────────────────────────────────────

CREATE TABLE users (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name              TEXT        NOT NULL,
    email             TEXT        UNIQUE NOT NULL,
    phone             TEXT,
    avatar_color      TEXT        NOT NULL DEFAULT '#6366f1',
    photo_object_key  TEXT,
    birthday          DATE,
    address           TEXT,
    password_hash     TEXT,
    email_verified_at TIMESTAMPTZ,
    deleted_at        TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE teams (
    id                           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name                         TEXT        NOT NULL,
    short                        TEXT,
    icon                         TEXT,
    icon_bg                      TEXT,
    icon_fg                      TEXT,
    photo_object_key             TEXT,
    logo_object_key              TEXT,
    description                  TEXT,
    reason_visibility_role_ids   UUID[]      NOT NULL DEFAULT '{}',
    created_at                   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE memberships (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id    UUID        NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    "group"    TEXT,
    joined_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (team_id, user_id)
);

CREATE TABLE roles (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id     UUID        NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    name        TEXT        NOT NULL,
    system      BOOLEAN     NOT NULL DEFAULT false,
    color       TEXT,
    permissions JSONB       NOT NULL DEFAULT '{"events":"none","members":"none","finances":"none","news":"none","polls":"none","settings":"none"}',
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE membership_roles (
    membership_id UUID NOT NULL REFERENCES memberships(id) ON DELETE CASCADE,
    role_id       UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    PRIMARY KEY (membership_id, role_id)
);

CREATE TABLE invites (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id    UUID        NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    code       TEXT        UNIQUE NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ── Events ──────────────────────────────────────────────────────────────────

CREATE TABLE event_series (
    id                    UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id               UUID        NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    type                  TEXT        NOT NULL CHECK (type IN ('training', 'auftritt', 'event')),
    title                 TEXT        NOT NULL,
    location              TEXT,
    note                  TEXT,
    meet_time             TIME,
    start_time            TIME,
    end_time              TIME,
    meet_time_mandatory   BOOLEAN     NOT NULL DEFAULT false,
    response_mode         TEXT        NOT NULL DEFAULT 'opt_in' CHECK (response_mode IN ('opt_in', 'opt_out')),
    nominated_role_ids    UUID[]      NOT NULL DEFAULT '{}',
    repeat_weeks          INT         NOT NULL DEFAULT 1,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE events (
    id                    UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id               UUID        NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    series_id             UUID        REFERENCES event_series(id) ON DELETE SET NULL,
    type                  TEXT        NOT NULL CHECK (type IN ('training', 'auftritt', 'event')),
    title                 TEXT        NOT NULL,
    date                  DATE        NOT NULL,
    location              TEXT,
    note                  TEXT,
    result                TEXT,
    meet_time             TIME,
    start_time            TIME,
    end_time              TIME,
    meet_time_mandatory   BOOLEAN     NOT NULL DEFAULT false,
    response_mode         TEXT        NOT NULL DEFAULT 'opt_in' CHECK (response_mode IN ('opt_in', 'opt_out')),
    nominated_role_ids    UUID[]      NOT NULL DEFAULT '{}',
    recurring             BOOLEAN     NOT NULL DEFAULT false,
    status                TEXT        NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'cancelled')),
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT events_end_after_start_time CHECK (start_time IS NULL OR end_time IS NULL OR end_time > start_time)
);

CREATE TABLE attendance (
    id                 UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id           UUID        NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    user_id            UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status             TEXT        NOT NULL CHECK (status IN ('yes','no','maybe','pending','not_nominated')),
    reason             TEXT,
    reason_id          TEXT,
    reason_visibility  TEXT        CHECK (reason_visibility IN ('trainers', 'team')),
    at                 TIMESTAMPTZ,
    UNIQUE (event_id, user_id)
);

CREATE TABLE event_comments (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id   UUID        NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    text       TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE absences (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    team_id    UUID        NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    from_date  DATE        NOT NULL,
    to_date    DATE        NOT NULL,
    reason     TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (to_date >= from_date),
    CONSTRAINT absences_span_within_limit CHECK (to_date - from_date <= 1095)
);

-- ── Communication ────────────────────────────────────────────────────────────

CREATE TABLE news (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id    UUID        NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    author_id  UUID        NOT NULL REFERENCES users(id),
    title      TEXT        NOT NULL,
    body       TEXT        NOT NULL,
    pinned     BOOLEAN     NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE polls (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id    UUID        NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    creator_id UUID        NOT NULL REFERENCES users(id),
    question   TEXT        NOT NULL,
    multiple   BOOLEAN     NOT NULL DEFAULT false,
    anonymous  BOOLEAN     NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE poll_options (
    id         UUID    PRIMARY KEY DEFAULT gen_random_uuid(),
    poll_id    UUID    NOT NULL REFERENCES polls(id) ON DELETE CASCADE,
    text       TEXT    NOT NULL,
    sort_order INT     NOT NULL
);

CREATE TABLE poll_votes (
    poll_id   UUID NOT NULL REFERENCES polls(id) ON DELETE CASCADE,
    option_id UUID NOT NULL REFERENCES poll_options(id) ON DELETE CASCADE,
    user_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    PRIMARY KEY (poll_id, option_id, user_id)
);

-- ── Finances ─────────────────────────────────────────────────────────────────
-- Amounts are stored as integer cents (BIGINT), not NUMERIC, so no Go call
-- site or JSON-number API boundary can reintroduce binary floating-point
-- imprecision.

CREATE TABLE transactions (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id    UUID        NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    type       TEXT        NOT NULL CHECK (type IN ('income', 'expense')),
    title      TEXT        NOT NULL,
    amount     BIGINT      NOT NULL CONSTRAINT transactions_amount_positive CHECK (amount > 0)
                           CONSTRAINT transactions_amount_max CHECK (amount <= 100000000),
    date       DATE        NOT NULL DEFAULT CURRENT_DATE,
    category   TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE penalties (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id    UUID        NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    label      TEXT        NOT NULL,
    amount     BIGINT      NOT NULL CONSTRAINT penalties_amount_positive CHECK (amount > 0)
                           CONSTRAINT penalties_amount_max CHECK (amount <= 100000000),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE penalty_assignments (
    id         UUID    PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id    UUID    NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    user_id    UUID    NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    penalty_id UUID    REFERENCES penalties(id) ON DELETE SET NULL,
    paid       BOOLEAN NOT NULL DEFAULT false,
    date       DATE    NOT NULL DEFAULT CURRENT_DATE,
    -- Snapshotted at assignment time so a later edit to the penalty catalog
    -- (label/amount) never retroactively changes a past, possibly-already-paid
    -- assignment. Nullable: no read path requires a value, and no code path
    -- exists that would need to distinguish "not yet snapshotted" from
    -- "snapshotted as empty".
    amount     BIGINT,
    label      TEXT
);

CREATE TABLE contributions (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id    UUID        NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    month      CHAR(7)     NOT NULL,
    label      TEXT,
    amount     BIGINT      NOT NULL CONSTRAINT contributions_amount_positive CHECK (amount > 0)
                           CONSTRAINT contributions_amount_max CHECK (amount <= 100000000),
    status     TEXT        NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'paid')),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (team_id, user_id, month)
);

-- ── Notifications & Auth ──────────────────────────────────────────────────────

CREATE TABLE notifications (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id      UUID        NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    type         TEXT        NOT NULL,
    actor_id     UUID        REFERENCES users(id) ON DELETE SET NULL,
    status       TEXT,
    title        TEXT,
    event_id     UUID        REFERENCES events(id) ON DELETE SET NULL,
    event_title  TEXT,
    event_date   DATE,
    note         TEXT,
    -- River guarantees at-least-once delivery, not exactly-once; the worker
    -- uses ON CONFLICT DO NOTHING against the partial unique index below to
    -- make a retried job's notification insert idempotent.
    river_job_id BIGINT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE notif_seen (
    team_id  UUID        NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    user_id  UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    seen_at  TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (team_id, user_id)
);

CREATE TABLE sessions (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT        UNIQUE NOT NULL,
    provider   TEXT        NOT NULL DEFAULT 'password',
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Forward-compatible scaffolding for a possible future OIDC-only account (no
-- password); no OIDC integration exists yet and no code path writes here.
CREATE TABLE oidc_accounts (
    id       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id  UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider TEXT NOT NULL,
    subject  TEXT NOT NULL,
    UNIQUE (provider, subject)
);

CREATE TABLE email_verification_tokens (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash  TEXT        UNIQUE NOT NULL,
    expires_at  TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ── Audit log ──────────────────────────────────────────────────────────────────
-- Persistent audit log for security-sensitive operations. Records are
-- immutable from the application's perspective (no code path issues an
-- UPDATE/DELETE against this table outside of retention); retention is
-- enforced by internal/jobs.RetentionWorker (RETENTION_AUDIT_LOG_DAYS,
-- default 365).

CREATE TABLE audit_log (
    id          BIGSERIAL   PRIMARY KEY,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    event       TEXT        NOT NULL,
    outcome     TEXT        NOT NULL CHECK (outcome IN ('success', 'failure')),
    actor_id    TEXT,
    attrs       JSONB       NOT NULL DEFAULT '{}'
);

-- ── River job queue tables (managed by River) ─────────────────────────────────

-- River requires its own schema; installed via river migrate-up in the Makefile.
-- See: https://riverqueue.com/docs/schema

-- +goose StatementEnd

-- Indexes use CONCURRENTLY so they build without holding a full table lock,
-- which matters on large datasets in production. CONCURRENTLY cannot run
-- inside a transaction or a StatementBegin/End batch, so these statements
-- live outside the StatementBegin/End block above. The NO TRANSACTION
-- annotation at the top of this file is required for CONCURRENTLY to work.
-- IF NOT EXISTS makes re-runs of this migration safe.

-- ── Indexes ───────────────────────────────────────────────────────────────────
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_memberships_user_id          ON memberships           (user_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_roles_team_id                ON roles                 (team_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_membership_roles_membership  ON membership_roles      (membership_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_membership_roles_role_id     ON membership_roles      (role_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_invites_expires_at           ON invites               (expires_at);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_event_series_team_id         ON event_series          (team_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_events_team_date_id          ON events                (team_id, date, id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_events_series_id             ON events                (series_id) WHERE series_id IS NOT NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_attendance_event_id          ON attendance            (event_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_attendance_user_id           ON attendance            (user_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_event_comments_event_id      ON event_comments        (event_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_event_comments_user_id       ON event_comments        (user_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_absences_team_user           ON absences              (team_id, user_id, from_date DESC, id DESC);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_absences_team_date_id        ON absences              (team_id, from_date DESC, id DESC);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_news_team_id                 ON news                  (team_id, pinned DESC, created_at DESC, id DESC);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_news_author_id               ON news                  (author_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_polls_team_id                ON polls                 (team_id, created_at DESC, id DESC);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_polls_creator_id             ON polls                 (creator_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_poll_options_poll_id         ON poll_options          (poll_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_poll_votes_user_id           ON poll_votes            (user_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_transactions_team_id         ON transactions          (team_id, date DESC, created_at DESC);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_penalties_team_id            ON penalties             (team_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_penalty_assignments_team_user ON penalty_assignments  (team_id, user_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_penalty_assignments_team_date ON penalty_assignments  (team_id, date DESC, id DESC);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_penalty_assignments_user_id  ON penalty_assignments   (user_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_contributions_team_month     ON contributions         (team_id, month);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_contributions_user_id        ON contributions         (user_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_notifications_team_created   ON notifications         (team_id, created_at DESC);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_notifications_created_at_brin ON notifications USING BRIN (created_at);
CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS notifications_river_job_id_idx ON notifications (river_job_id) WHERE river_job_id IS NOT NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_sessions_user_id             ON sessions              (user_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_sessions_expires_at          ON sessions              (expires_at);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_email_verification_tokens_user_id     ON email_verification_tokens (user_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_email_verification_tokens_expires_at  ON email_verification_tokens (expires_at);
CREATE INDEX CONCURRENTLY IF NOT EXISTS audit_log_occurred_at_idx        ON audit_log             (occurred_at DESC);
CREATE INDEX CONCURRENTLY IF NOT EXISTS audit_log_actor_id_idx           ON audit_log             (actor_id) WHERE actor_id IS NOT NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS audit_log_event_idx              ON audit_log             (event, occurred_at DESC);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_audit_log_occurred_at_brin   ON audit_log USING BRIN (occurred_at);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_audit_log_actor_occurred     ON audit_log             (actor_id, occurred_at DESC) WHERE actor_id IS NOT NULL;

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS audit_log;
DROP TABLE IF EXISTS email_verification_tokens;
DROP TABLE IF EXISTS oidc_accounts;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS notif_seen;
DROP TABLE IF EXISTS notifications;
DROP TABLE IF EXISTS contributions;
DROP TABLE IF EXISTS penalty_assignments;
DROP TABLE IF EXISTS penalties;
DROP TABLE IF EXISTS transactions;
DROP TABLE IF EXISTS poll_votes;
DROP TABLE IF EXISTS poll_options;
DROP TABLE IF EXISTS polls;
DROP TABLE IF EXISTS news;
DROP TABLE IF EXISTS absences;
DROP TABLE IF EXISTS event_comments;
DROP TABLE IF EXISTS attendance;
DROP TABLE IF EXISTS events;
DROP TABLE IF EXISTS event_series;
DROP TABLE IF EXISTS invites;
DROP TABLE IF EXISTS membership_roles;
DROP TABLE IF EXISTS roles;
DROP TABLE IF EXISTS memberships;
DROP TABLE IF EXISTS teams;
DROP TABLE IF EXISTS users;

-- +goose StatementEnd
