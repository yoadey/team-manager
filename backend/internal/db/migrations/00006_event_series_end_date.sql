-- +goose Up

-- Alternative to event_series.repeat_weeks: a series may instead be defined
-- by an end date, with the occurrence count derived (weekly cadence, up to
-- and including the end date) rather than supplied directly. Nullable and
-- additive -- repeat_weeks keeps its NOT NULL DEFAULT 1 for backward
-- compatibility with every series created before this column existed. A
-- series only uses one mode or the other in practice (enforced in
-- events.Service.CreateEvent, not a CHECK constraint here, since the two
-- knobs being mutually exclusive is a property of the request shape, not
-- something the stored row alone can express -- repeat_weeks is still
-- populated with the derived occurrence count either way, so read paths
-- that only ever consulted repeat_weeks keep working unchanged).
ALTER TABLE event_series ADD COLUMN repeat_end_date DATE;

-- +goose Down

ALTER TABLE event_series DROP COLUMN IF EXISTS repeat_end_date;
