-- +goose Up

-- Marks an absence's covered event dates as excluded entirely from that
-- member's attendance statistics (neither counted as attending nor as
-- absent), instead of the default "counts as absent" behavior. Settable by
-- the absence's own owner unconditionally, or by an events:write holder on
-- another member's absence (enforced in absences.Service, not by a DB
-- constraint -- see the accompanying OpenSpec change's design.md).
ALTER TABLE absences ADD COLUMN not_relevant_for_stats BOOLEAN NOT NULL DEFAULT false;

-- Audit trail for who set the flag -- the first place one member can cause
-- a write against another member's absence row, so worth being able to
-- answer "who marked this."
ALTER TABLE absences ADD COLUMN not_relevant_set_by UUID REFERENCES users(id);

-- +goose Down

ALTER TABLE absences DROP COLUMN IF EXISTS not_relevant_set_by;
ALTER TABLE absences DROP COLUMN IF EXISTS not_relevant_for_stats;
