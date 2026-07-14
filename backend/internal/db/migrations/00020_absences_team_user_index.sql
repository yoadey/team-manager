-- +goose NO TRANSACTION

-- +goose Up

-- ListByUser ("My Absences") filters on team_id AND user_id and orders by
-- from_date DESC, id DESC, but the only supporting index is
-- idx_absences_team_dates (team_id, from_date, to_date), which has no
-- user_id column. Postgres can use it for the team_id filter but still has
-- to scan/filter every absence row for the whole team to find the caller's
-- own rows on every "My Absences" request. Add a composite index matching
-- this query's actual filter+sort key, mirroring the reasoning in migration
-- 00017 for news/polls/transactions.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_absences_team_user
    ON absences (team_id, user_id, from_date DESC, id DESC);

-- +goose Down

DROP INDEX IF EXISTS idx_absences_team_user;
