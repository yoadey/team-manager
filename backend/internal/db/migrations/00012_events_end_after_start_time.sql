-- +goose Up

-- events.UpdateEvent's partial-PATCH path only validates endTime against
-- startTime when the handler receives both fields in the same request (see
-- validateEventTimeFields in internal/events/handler.go); a PATCH containing
-- only one of the two fields skips that check entirely and can leave the row
-- with end_time <= start_time. absences already has an equivalent CHECK
-- constraint protecting the same partial-update scenario for from_date/
-- to_date; events had none. Added NOT VALID so existing rows (if any predate
-- CreateEvent's ordering check) don't block deployment -- every future write,
-- from any code path, is enforced regardless.
ALTER TABLE events ADD CONSTRAINT events_end_after_start_time
    CHECK (start_time IS NULL OR end_time IS NULL OR end_time > start_time) NOT VALID;

-- +goose Down

ALTER TABLE events DROP CONSTRAINT IF EXISTS events_end_after_start_time;
