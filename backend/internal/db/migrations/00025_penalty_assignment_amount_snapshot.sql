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

-- Deliberately NOT enforced NOT NULL here (no VALIDATE CONSTRAINT / SET NOT
-- NULL step), unlike 00016/00018's expand/contract precedent. Under a
-- RollingUpdate deploy (the default, replicaCount > 1 in prod -- see
-- docs/operations.md's "Rolling upgrades & schema-changing migrations"),
-- old-version pods still running the pre-this-release binary run
-- CreateAssignment's INSERT without amount/label at all; a NOT NULL
-- constraint landing in the SAME release as the code that starts always
-- supplying them would make every one of those concurrent old-pod inserts
-- fail for the whole rollout window. gen.PenaltyAssignment.Amount/Label are
-- already *int64/*string (nullable) end-to-end -- toGenAssignment,
-- PenaltyAssignmentRow, and the OpenAPI schema all already tolerate a NULL
-- value -- so nothing downstream requires this to be non-null at the DB
-- level. A future migration, once this release has been fully rolled out
-- (no old-code pod can still be running), MAY add NOT NULL via the standard
-- NOT VALID -> VALIDATE -> SET NOT NULL -> DROP CONSTRAINT sequence if
-- desired; it isn't required for correctness.

-- +goose Down

ALTER TABLE penalty_assignments DROP COLUMN IF EXISTS label;
ALTER TABLE penalty_assignments DROP COLUMN IF EXISTS amount;
