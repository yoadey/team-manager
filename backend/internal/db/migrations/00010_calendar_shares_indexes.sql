-- +goose NO TRANSACTION

-- +goose Up

-- Serves listSharedCalendarSources's "which owner teams have shared with
-- viewer_team_id" query direction; the PK (owner_team_id, viewer_team_id)
-- only serves the opposite direction (listCalendarShares).
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_calendar_shares_viewer
    ON calendar_shares (viewer_team_id);

-- +goose Down

DROP INDEX IF EXISTS idx_calendar_shares_viewer;
