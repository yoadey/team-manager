-- +goose NO TRANSACTION

-- +goose Up

-- Backs CountOpenContributions' new "AND NOT c.archived" filter and every
-- other "exclude archived" scan added by this change (ListContributions'
-- default view, the linking-picker matrix) -- partial on the common case
-- (most rows aren't archived) rather than a full-column index.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_contributions_team_archived ON contributions (team_id) WHERE NOT archived;

-- +goose Down

DROP INDEX IF EXISTS idx_contributions_team_archived;
