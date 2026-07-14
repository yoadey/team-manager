-- +goose Up

-- absences.Repository.Update's partial-PATCH path only validates
-- maxAbsenceSpanDays (handler.go) when the request supplies both from/to in
-- the same PATCH; a request patching only one of the two fields skips that
-- check entirely (the same partial-update gap events_end_after_start_time
-- closed for events' start/end times), letting a self-service absence PATCH
-- stretch to an arbitrary span, e.g. by re-patching only `to` after create.
-- Added NOT VALID so existing rows don't block deployment -- every future
-- write, from any code path, is enforced regardless.
ALTER TABLE absences ADD CONSTRAINT absences_span_within_limit
    CHECK (to_date - from_date <= 1095) NOT VALID;

-- +goose Down

ALTER TABLE absences DROP CONSTRAINT IF EXISTS absences_span_within_limit;
