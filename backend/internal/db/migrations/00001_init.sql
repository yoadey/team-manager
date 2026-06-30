-- +goose NO TRANSACTION
-- +goose Up
-- +goose StatementBegin

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ── Core Entities ──────────────────────────────────────────────────────────

CREATE TABLE users (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name          TEXT        NOT NULL,
    email         TEXT        UNIQUE NOT NULL,
    phone         TEXT,
    avatar_color  TEXT        NOT NULL DEFAULT '#6366f1',
    photo_data    BYTEA,
    photo_mime    TEXT,
    birthday      DATE,
    address       TEXT,
    password_hash TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE teams (
    id                           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name                         TEXT        NOT NULL,
    short                        TEXT,
    icon                         TEXT,
    icon_bg                      TEXT,
    icon_fg                      TEXT,
    photo_data                   BYTEA,
    photo_mime                   TEXT,
    logo_data                    BYTEA,
    logo_mime                    TEXT,
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
    UNIQUE (team_id, user_id)
);

CREATE TABLE roles (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id     UUID        NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    name        TEXT        NOT NULL,
    system      BOOLEAN     NOT NULL DEFAULT false,
    color       TEXT,
    permissions JSONB       NOT NULL DEFAULT '{"events":"none","members":"none","finances":"none","news":"none","polls":"none","settings":"none"}'
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
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now()
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
    CHECK (to_date >= from_date)
);

-- ── Communication ────────────────────────────────────────────────────────────

CREATE TABLE news (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id    UUID        NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    author_id  UUID        NOT NULL REFERENCES users(id),
    title      TEXT        NOT NULL,
    body       TEXT        NOT NULL,
    pinned     BOOLEAN     NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE polls (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id    UUID        NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    creator_id UUID        NOT NULL REFERENCES users(id),
    question   TEXT        NOT NULL,
    multiple   BOOLEAN     NOT NULL DEFAULT false,
    anonymous  BOOLEAN     NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
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

CREATE TABLE transactions (
    id         UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id    UUID           NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    type       TEXT           NOT NULL CHECK (type IN ('income', 'expense')),
    title      TEXT           NOT NULL,
    amount     NUMERIC(10,2)  NOT NULL,
    date       DATE           NOT NULL DEFAULT CURRENT_DATE,
    category   TEXT,
    created_at TIMESTAMPTZ    NOT NULL DEFAULT now()
);

CREATE TABLE penalties (
    id      UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id UUID          NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    label   TEXT          NOT NULL,
    amount  NUMERIC(10,2) NOT NULL
);

CREATE TABLE penalty_assignments (
    id         UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id    UUID          NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    user_id    UUID          NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    penalty_id UUID          NOT NULL REFERENCES penalties(id) ON DELETE CASCADE,
    paid       BOOLEAN       NOT NULL DEFAULT false,
    date       DATE          NOT NULL DEFAULT CURRENT_DATE
);

CREATE TABLE contributions (
    id      UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id UUID          NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    user_id UUID          NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    month   CHAR(7)       NOT NULL,
    label   TEXT,
    amount  NUMERIC(10,2) NOT NULL,
    status  TEXT          NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'paid')),
    UNIQUE (team_id, user_id, month)
);

-- ── Notifications & Auth ──────────────────────────────────────────────────────

CREATE TABLE notifications (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id     UUID        NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    type        TEXT        NOT NULL,
    actor_id    UUID        REFERENCES users(id) ON DELETE SET NULL,
    status      TEXT,
    title       TEXT,
    event_id    UUID        REFERENCES events(id) ON DELETE SET NULL,
    event_title TEXT,
    event_date  DATE,
    note        TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
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

CREATE TABLE oidc_accounts (
    id       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id  UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider TEXT NOT NULL,
    subject  TEXT NOT NULL,
    UNIQUE (provider, subject)
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
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_events_team_id_date          ON events              (team_id, date);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_attendance_event_id          ON attendance          (event_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_attendance_user_id           ON attendance          (user_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_notifications_team_created   ON notifications       (team_id, created_at DESC);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_absences_team_dates          ON absences            (team_id, from_date, to_date);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_penalty_assignments_team_user ON penalty_assignments (team_id, user_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_contributions_team_month     ON contributions       (team_id, month);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_membership_roles_membership  ON membership_roles    (membership_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_sessions_user_id             ON sessions            (user_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_sessions_expires_at          ON sessions            (expires_at);

-- +goose Down
-- +goose StatementBegin

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
