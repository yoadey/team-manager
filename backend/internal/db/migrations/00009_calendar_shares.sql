-- +goose Up

-- One row per (owner_team_id, viewer_team_id) grant: owner_team_id has
-- allowed viewer_team_id's members to read a redacted (time/location/title/
-- type only) projection of its calendar. Managing a grant requires
-- settings:write on owner_team_id (see internal/calendarshare); reading the
-- redacted projection requires membership in viewer_team_id and an existing
-- row here for (owner, viewer) -- no RBAC module gate beyond that, since
-- this is deliberately weaker access than any module permission grants.
CREATE TABLE calendar_shares (
    owner_team_id  UUID        NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    viewer_team_id UUID        NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (owner_team_id, viewer_team_id),
    CONSTRAINT calendar_shares_not_self CHECK (owner_team_id <> viewer_team_id)
);

-- The reverse lookup direction ("which owner teams have shared with
-- viewer_team_id", listSharedCalendarSources) isn't served by the PK's
-- leading column -- see the follow-up indexes migration for that index,
-- built CONCURRENTLY since CREATE INDEX without it takes a table-level lock.

-- +goose Down

DROP TABLE IF EXISTS calendar_shares;
