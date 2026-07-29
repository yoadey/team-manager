-- +goose Up

-- Optional cancellation/RSVP-change lead time, expressed as minutes before
-- the event's start: once that instant has passed, non-privileged members
-- can no longer change their attendance response (see events.Service.
-- SetAttendance's cutoff check). Relative to start (rather than an absolute
-- timestamp like rsvp_deadline) so a single series definition applies the
-- same cutoff to every occurrence without recomputing it. Nullable on both
-- tables -- events.cancel_lead_minutes is what's actually enforced at
-- response time; event_series.cancel_lead_minutes is the template value
-- CreateSeries seeds each occurrence's events.cancel_lead_minutes from.
ALTER TABLE events        ADD COLUMN cancel_lead_minutes INTEGER;
ALTER TABLE event_series  ADD COLUMN cancel_lead_minutes INTEGER;

-- +goose Down

ALTER TABLE events        DROP COLUMN IF EXISTS cancel_lead_minutes;
ALTER TABLE event_series  DROP COLUMN IF EXISTS cancel_lead_minutes;
