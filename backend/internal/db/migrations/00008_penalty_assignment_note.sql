-- +goose Up

-- Free-text note explaining why a penalty was assigned (e.g. "missed training
-- without excuse"). Nullable: most assignments won't need one, and the
-- existing CreateAssignment call chain never populated anything here before
-- this migration.
ALTER TABLE penalty_assignments ADD COLUMN note TEXT;

-- +goose Down

ALTER TABLE penalty_assignments DROP COLUMN IF EXISTS note;
