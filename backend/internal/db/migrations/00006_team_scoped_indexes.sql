-- +goose NO TRANSACTION

-- +goose Up

-- news, polls, transactions, penalties, roles, and event_series were missing
-- an index on team_id even though every list query filters on it directly
-- (e.g. "SELECT ... FROM transactions WHERE team_id = $1"). Without an index
-- these degrade to sequential scans as a team's history grows. Every other
-- team-scoped table (events, attendance, notifications, absences,
-- penalty_assignments, contributions) already has one; this closes the gap.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_news_team_id          ON news          (team_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_polls_team_id         ON polls         (team_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_transactions_team_id  ON transactions  (team_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_penalties_team_id     ON penalties     (team_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_roles_team_id         ON roles         (team_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_event_series_team_id  ON event_series  (team_id);

-- +goose Down

DROP INDEX IF EXISTS idx_news_team_id;
DROP INDEX IF EXISTS idx_polls_team_id;
DROP INDEX IF EXISTS idx_transactions_team_id;
DROP INDEX IF EXISTS idx_penalties_team_id;
DROP INDEX IF EXISTS idx_roles_team_id;
DROP INDEX IF EXISTS idx_event_series_team_id;
