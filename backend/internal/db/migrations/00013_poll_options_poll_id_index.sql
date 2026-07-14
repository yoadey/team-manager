-- +goose NO TRANSACTION

-- +goose Up

-- poll_options has no index on poll_id (only a PK on id) -- migration
-- 00006_team_scoped_indexes.sql added the missing team_id indexes but only
-- looked at tables with a team_id column, missing this join table filtered
-- by a different foreign key. polls.Repository.ListOptions/ListOptionsByPollIDs
-- run "WHERE poll_id = $1"/"WHERE poll_id = ANY($1)" with no supporting
-- index, forcing a sequential scan of the entire poll_options table (across
-- every team's polls, not just the requesting team) on every page load of
-- GET /teams/{teamId}/polls. poll_votes doesn't need the same fix: its PK
-- (poll_id, option_id, user_id) already leads with poll_id.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_poll_options_poll_id ON poll_options (poll_id);

-- +goose Down

DROP INDEX IF EXISTS idx_poll_options_poll_id;
