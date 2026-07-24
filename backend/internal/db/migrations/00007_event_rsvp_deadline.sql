-- +goose Up

-- Optional RSVP deadline: once passed, non-privileged members can no longer
-- change their attendance response for the event (see events.Service.
-- SetAttendance's deadline check). Nullable on both tables -- events.
-- rsvp_deadline is what's actually enforced at response time; event_series.
-- rsvp_deadline is the template value CreateSeries seeds each occurrence's
-- events.rsvp_deadline from, mirroring how every other per-occurrence field
-- (location, note, meet/start/end time, ...) is already templated from the
-- series row.
ALTER TABLE events        ADD COLUMN rsvp_deadline TIMESTAMPTZ;
ALTER TABLE event_series  ADD COLUMN rsvp_deadline TIMESTAMPTZ;

-- +goose Down

ALTER TABLE events        DROP COLUMN IF EXISTS rsvp_deadline;
ALTER TABLE event_series  DROP COLUMN IF EXISTS rsvp_deadline;
