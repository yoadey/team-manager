-- +goose NO TRANSACTION

-- +goose Up

-- ListEvents filters on team_id and keyset-orders on (date, id) for all
-- three scopes (past: date DESC, id DESC; upcoming/all: date ASC, id ASC),
-- but the only supporting index -- idx_events_team_id_date (team_id, date) --
-- has no id column. Since events.id is a random gen_random_uuid() PK with no
-- correlation to date, any team with more than one event sharing a date
-- forces Postgres into an extra sort/scan instead of getting rows
-- pre-ordered off the index with LIMIT pushed down. events is the hottest
-- table in the schema (every team's calendar view hits ListEvents on
-- essentially every page load); this composite index mirrors the fix
-- migration 00017 already applied to news/polls/transactions and 00020
-- applied to absences. A single ascending (team_id, date, id) index serves
-- both ASC (forward scan) and DESC (backward scan) orderings, so one index
-- replaces the old 2-column one rather than adding alongside it.
DROP INDEX IF EXISTS idx_events_team_id_date;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_events_team_date_id
    ON events (team_id, date, id);

-- +goose Down

DROP INDEX IF EXISTS idx_events_team_date_id;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_events_team_id_date ON events (team_id, date);
