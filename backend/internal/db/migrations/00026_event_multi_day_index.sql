-- +goose NO TRANSACTION

-- +goose Up

-- ListEvents' upcoming/past scope filters on COALESCE(end_date, date) (see
-- internal/events/repository.go) so an ongoing multi-day event stays
-- "upcoming" until its last day passes -- idx_events_team_date_id
-- (team_id, date, id) can't serve that as a sargable range condition since
-- the expression isn't a bare column. Without a matching expression index,
-- the WHERE clause degrades from an index range scan on `date` to an
-- Index Scan across every one of the team's rows with the COALESCE
-- evaluated as a non-indexed Filter -- effectively a full scan of the
-- team's event history on every load of the events list, its most
-- frequently hit query.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_events_team_coalesce_enddate_id
    ON events (team_id, (COALESCE(end_date, date)), id);

-- +goose Down

DROP INDEX IF EXISTS idx_events_team_coalesce_enddate_id;
