-- +goose Up

-- Optional multi-day span for a non-recurring event: when set, the event
-- occurs on every calendar day from `date` through `end_date` inclusive,
-- mirroring absences' from_date/to_date range. Series events never set this
-- (enforced at the application layer, not by a CHECK, since it depends on
-- the recurring flag which lives on event_series, not events).
ALTER TABLE events ADD COLUMN end_date DATE;

ALTER TABLE events ADD CONSTRAINT events_end_date_after_date
    CHECK (end_date IS NULL OR end_date >= date);

-- Caps the span the same way absences_span_within_limit caps from_date/
-- to_date (see 00001_init.sql): without a bound, an arbitrarily large span
-- would make every calendar render (which expands the event across every
-- day it covers) do unbounded work per event.
ALTER TABLE events ADD CONSTRAINT events_multiday_span_within_limit
    CHECK (end_date IS NULL OR end_date - date <= 1095);

-- +goose Down

ALTER TABLE events DROP CONSTRAINT IF EXISTS events_multiday_span_within_limit;
ALTER TABLE events DROP CONSTRAINT IF EXISTS events_end_date_after_date;
ALTER TABLE events DROP COLUMN IF EXISTS end_date;
