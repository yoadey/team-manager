-- +goose Up

-- Free-text description a treasurer can attach to a fee beyond its short
-- name (e.g. what the fee actually covers). Nullable: most contributions
-- won't need one, mirroring penalty_assignments.note
-- (00008_penalty_assignment_note.sql).
ALTER TABLE contributions ADD COLUMN description TEXT;

-- Lets a treasurer move a no-longer-relevant fee period out of the default
-- view without deleting it (and without unlinking any transaction that
-- already paid it -- see finances.Repository.DeleteContribution's doc
-- comment on why deletion, unlike archiving, is destructive to the fee
-- record itself). Defaults to false so every existing row stays visible.
ALTER TABLE contributions ADD COLUMN archived BOOLEAN NOT NULL DEFAULT FALSE;

-- +goose Down

ALTER TABLE contributions DROP COLUMN IF EXISTS archived;
ALTER TABLE contributions DROP COLUMN IF EXISTS description;
