-- +goose Up

-- The fixed-date RSVP deadline is superseded by cancel_lead_minutes (added
-- in 00011), which expresses the same cutoff as a lead time before start --
-- the only form that survives recurring series correctly. Dropping this
-- column is destructive: events/series with rsvp_deadline set lose that
-- cutoff outright (no cancel_lead_minutes backfill -- a single stored
-- absolute deadline doesn't decompose into "N minutes before start" for
-- every future series occurrence).
ALTER TABLE events        DROP COLUMN IF EXISTS rsvp_deadline;
ALTER TABLE event_series  DROP COLUMN IF EXISTS rsvp_deadline;

-- +goose Down

-- Re-adds the column but cannot restore prior values -- irreversible data
-- loss on Up, same as any DROP COLUMN.
ALTER TABLE events        ADD COLUMN rsvp_deadline TIMESTAMPTZ;
ALTER TABLE event_series  ADD COLUMN rsvp_deadline TIMESTAMPTZ;
