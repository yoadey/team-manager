-- +goose NO TRANSACTION

-- +goose Up

-- Serves the "which cross-team events target team X" read direction (the
-- EXISTS-join every relaxed events/attendance/comments query now uses); the
-- PK (event_id, team_id) only serves the opposite direction (per-event
-- target lookups).
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_event_teams_team
    ON event_teams (team_id, event_id);

-- +goose Down

DROP INDEX IF EXISTS idx_event_teams_team;
