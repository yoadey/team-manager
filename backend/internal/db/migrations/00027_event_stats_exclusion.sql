-- +goose Up

-- Marks an event (or, on event_series, its template) as excluded from
-- attendance statistics while remaining a fully normal event otherwise
-- (RSVP, comments, notifications, cancellation are all unaffected). A
-- constant boolean default is a metadata-only change in Postgres, not a
-- full-table rewrite, so no NOT VALID/VALIDATE CONSTRAINT dance is needed
-- here (unlike ALTER COLUMN ... SET NOT NULL).
ALTER TABLE event_series ADD COLUMN exclude_from_stats BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE events ADD COLUMN exclude_from_stats BOOLEAN NOT NULL DEFAULT false;

-- +goose Down

ALTER TABLE events DROP COLUMN IF EXISTS exclude_from_stats;
ALTER TABLE event_series DROP COLUMN IF EXISTS exclude_from_stats;
