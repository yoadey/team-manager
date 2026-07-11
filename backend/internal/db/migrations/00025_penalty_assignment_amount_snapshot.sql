-- +goose Up

-- penalty_assignments only stored a penalty_id FK; every read (ListAssignments,
-- GetAssignmentByID, ListOpenPenaltiesByUser) joined penalties and read its
-- label/amount live. That means UpdatePenalty -- callable at any time by any
-- finances:write holder, with no restriction once a penalty has assignments --
-- silently rewrote the amount/label of every past assignment referencing it,
-- including ones already marked paid. A member who agreed to (and paid) a €5
-- penalty could see it retroactively become €50 with no notice, and the
-- audit log only records that *a* penalty changed, not what past finance
-- overviews looked like at the time. Snapshotting amount/label onto the
-- assignment row at creation time makes each assignment an immutable record
-- of what was actually assigned, matching how contributions already store
-- their own per-row amount instead of joining a shared definition.
-- BIGINT (integer cents), matching every other amount column since 00008
-- converted them off NUMERIC(10,2) precisely to avoid float/decimal
-- boundary issues -- not NUMERIC(10,2), which can't even hold
-- maxAmountCents (100_000_000, i.e. the app-allowed $1,000,000.00 penalty)
-- without a numeric field overflow, since NUMERIC(10,2) caps out just under 10^8.
ALTER TABLE penalty_assignments ADD COLUMN amount BIGINT;
ALTER TABLE penalty_assignments ADD COLUMN label TEXT;

-- Backfill: existing assignments have no historical amount to recover, so
-- the best available approximation is the penalty definition's current
-- value -- same tradeoff 00023 made backfilling roles from current data.
UPDATE penalty_assignments pa
SET amount = p.amount, label = p.label
FROM penalties p
WHERE p.id = pa.penalty_id AND pa.amount IS NULL;

-- A plain "ALTER COLUMN ... SET NOT NULL" takes an ACCESS EXCLUSIVE lock for
-- the duration of a full-table scan to verify no NULLs remain. Validating a
-- NOT VALID CHECK constraint first (SHARE UPDATE EXCLUSIVE, doesn't block
-- reads/writes) lets Postgres's planner skip that scan when SET NOT NULL
-- runs next, following the same expand/contract discipline as 00016+00018 --
-- maxAssignmentsPerTeam (100k) bounds a single team, but this table has no
-- overall cap across teams, so it isn't safe to assume small.
ALTER TABLE penalty_assignments ADD CONSTRAINT penalty_assignments_amount_not_null CHECK (amount IS NOT NULL) NOT VALID;
ALTER TABLE penalty_assignments ADD CONSTRAINT penalty_assignments_label_not_null CHECK (label IS NOT NULL) NOT VALID;
ALTER TABLE penalty_assignments VALIDATE CONSTRAINT penalty_assignments_amount_not_null;
ALTER TABLE penalty_assignments VALIDATE CONSTRAINT penalty_assignments_label_not_null;
ALTER TABLE penalty_assignments ALTER COLUMN amount SET NOT NULL;
ALTER TABLE penalty_assignments ALTER COLUMN label SET NOT NULL;
ALTER TABLE penalty_assignments DROP CONSTRAINT penalty_assignments_amount_not_null;
ALTER TABLE penalty_assignments DROP CONSTRAINT penalty_assignments_label_not_null;

-- +goose Down

ALTER TABLE penalty_assignments DROP COLUMN IF EXISTS label;
ALTER TABLE penalty_assignments DROP COLUMN IF EXISTS amount;
