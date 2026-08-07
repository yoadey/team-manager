-- +goose Up

-- Mirrors 00018_flexible_membership_fees.sql's treatment of contributions:
-- a penalty assignment's paid state is no longer a hand-flipped boolean --
-- it's derived from the sum of income transactions linked to it via the new
-- transactions.penalty_assignment_id below (see
-- finances.Repository.ListAssignments), so a manually-toggled flag and a
-- separately booked income transaction can never disagree about whether a
-- fine was actually paid.
ALTER TABLE penalty_assignments DROP COLUMN paid;

-- Links an income transaction to the penalty assignment it (fully) pays.
-- ON DELETE SET NULL (not CASCADE): deleting a penalty assignment must never
-- delete income that was genuinely received, only the courtesy link
-- describing which fine it paid.
ALTER TABLE transactions ADD COLUMN penalty_assignment_id UUID REFERENCES penalty_assignments(id) ON DELETE SET NULL;

-- +goose Down

-- Irreversible data loss on Up (dropped paid values and any
-- transaction<->assignment links), same as any DROP COLUMN -- see
-- 00012_remove_event_rsvp_deadline.sql.
ALTER TABLE transactions DROP COLUMN IF EXISTS penalty_assignment_id;
ALTER TABLE penalty_assignments ADD COLUMN paid BOOLEAN NOT NULL DEFAULT false;
