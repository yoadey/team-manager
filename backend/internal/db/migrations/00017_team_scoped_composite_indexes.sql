-- +goose NO TRANSACTION

-- +goose Up

-- news/polls/transactions each filter on team_id and then ORDER BY
-- additional columns before LIMIT, but migration 00006 only gave them a
-- single-column team_id index -- Postgres can use it to find matching rows,
-- but must still sort all of a team's matches in memory before applying
-- LIMIT, since the index doesn't return them pre-ordered. Replace each with
-- a composite index matching the query's actual sort key; the leading
-- team_id column still serves any other query that filters on team_id alone
-- (none currently exist for these three tables beyond id-scoped lookups,
-- which use the primary key instead), so the single-column index becomes
-- redundant rather than complementary.
DROP INDEX IF EXISTS idx_news_team_id;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_news_team_id
    ON news (team_id, pinned DESC, created_at DESC, id DESC);

DROP INDEX IF EXISTS idx_polls_team_id;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_polls_team_id
    ON polls (team_id, created_at DESC, id DESC);

DROP INDEX IF EXISTS idx_transactions_team_id;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_transactions_team_id
    ON transactions (team_id, date DESC, created_at DESC);

-- penalty_assignments' existing idx_penalty_assignments_team_user
-- (team_id, user_id) serves a different access pattern and stays as-is;
-- ListAssignments' own sort key (team_id, date DESC, id DESC, fixed in
-- round 37 for determinism) has no supporting index at all, so add one
-- alongside rather than replacing.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_penalty_assignments_team_date
    ON penalty_assignments (team_id, date DESC, id DESC);

-- +goose Down

DROP INDEX IF EXISTS idx_penalty_assignments_team_date;

DROP INDEX IF EXISTS idx_transactions_team_id;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_transactions_team_id ON transactions (team_id);

DROP INDEX IF EXISTS idx_polls_team_id;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_polls_team_id ON polls (team_id);

DROP INDEX IF EXISTS idx_news_team_id;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_news_team_id ON news (team_id);
