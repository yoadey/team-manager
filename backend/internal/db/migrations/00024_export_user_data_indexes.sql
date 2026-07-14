-- +goose NO TRANSACTION

-- +goose Up

-- auth.Repository.ExportUserData (GDPR Art. 15 data export, GET /auth/me/export)
-- runs 9 queries sequentially inside one shared 15s context.WithTimeout, and
-- six of them filter on a user_id-shaped column with no supporting index --
-- each degrades to a full table scan of a table shared across every team in
-- the deployment, not just the requesting user's teams:
--   - event_comments.user_id  (only event_id is indexed, since 00014)
--   - news.author_id          (only team_id is indexed)
--   - polls.creator_id        (only team_id is indexed)
--   - poll_votes.user_id      (PK (poll_id, option_id, user_id) leads with
--                              poll_id, per 00013's own comment, so it can't
--                              serve a user_id-only predicate)
--   - penalty_assignments.user_id (only the (team_id, user_id) composite
--                              exists, which can't be used without a
--                              team_id predicate)
--   - contributions.user_id   (only (team_id, month) and the
--                              (team_id, user_id, month) UNIQUE constraint
--                              exist, same problem as penalty_assignments)
-- As these tables grow across all teams over the years, an export for any
-- single user runs multiple full-table sequential scans back-to-back
-- inside one fixed budget -- a slow early phase can exhaust it before
-- reaching later ones, and the whole request then fails with a generic
-- 500, repeatably, with no way for the user to self-serve their GDPR
-- request.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_event_comments_user_id     ON event_comments     (user_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_news_author_id             ON news               (author_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_polls_creator_id           ON polls              (creator_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_poll_votes_user_id         ON poll_votes         (user_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_penalty_assignments_user_id ON penalty_assignments (user_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_contributions_user_id      ON contributions      (user_id);

-- +goose Down

DROP INDEX IF EXISTS idx_event_comments_user_id;
DROP INDEX IF EXISTS idx_news_author_id;
DROP INDEX IF EXISTS idx_polls_creator_id;
DROP INDEX IF EXISTS idx_poll_votes_user_id;
DROP INDEX IF EXISTS idx_penalty_assignments_user_id;
DROP INDEX IF EXISTS idx_contributions_user_id;
