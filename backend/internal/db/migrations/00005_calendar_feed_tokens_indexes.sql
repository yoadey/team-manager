-- +goose NO TRANSACTION

-- +goose Up

-- Enforces "one active token per user+team" at the DB layer -- IssueToken
-- revokes any existing active row before inserting a new one, but this
-- index closes the race if two issue requests for the same pair ever ran
-- at the same time.
CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS idx_calendar_feed_tokens_active_user_team
    ON calendar_feed_tokens (user_id, team_id)
    WHERE revoked_at IS NULL;

-- +goose Down

DROP INDEX IF EXISTS idx_calendar_feed_tokens_active_user_team;
