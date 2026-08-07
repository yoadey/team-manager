-- +goose Up

-- Free-text note about a transaction (e.g. "Bar erhalten, Quittung Nr. 12"),
-- shown only when the transaction is opened for editing -- never in the
-- transaction list. Nullable, mirroring penalty_assignments.note
-- (00008_penalty_assignment_note.sql).
ALTER TABLE transactions ADD COLUMN note TEXT;

-- +goose Down

ALTER TABLE transactions DROP COLUMN IF EXISTS note;
