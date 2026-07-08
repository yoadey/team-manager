-- +goose NO TRANSACTION

-- +goose Up

-- events has no index on series_id -- the only relevant index is
-- idx_events_team_id_date (team_id, date). UpdateEvent/SetStatus/DeleteEvent's
-- scope=series paths all filter on series_id (UpdateEvent's series-wide
-- UPDATE doesn't even filter team_id in that statement), so editing/
-- cancelling/deleting an entire recurring series -- a normal, common admin
-- action -- forces a scan of every event the team has ever had (or, for
-- UpdateEvent, the whole table) rather than just the <=104 rows in that
-- series. These queries run inside the same pg_advisory_xact_lock-guarded
-- transaction CreateSeries' batching fix (migration-adjacent code change)
-- was written to keep short, since a slow scan here serializes every other
-- team mutation for its duration. Partial index since most events are
-- standalone (series_id IS NULL).
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_events_series_id
    ON events (series_id) WHERE series_id IS NOT NULL;

-- +goose Down

DROP INDEX IF EXISTS idx_events_series_id;
