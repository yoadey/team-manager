-- +goose Up

-- Preserve penalty history on catalog deletion. penalty_assignments.penalty_id
-- was ON DELETE CASCADE, so deleting a penalty definition erased every
-- assignment referencing it -- including paid, historical ones -- retroactively
-- changing the finance overview and defeating 00025's whole purpose (making
-- each assignment an immutable record via a snapshotted amount/label).
--
-- Switch the FK to ON DELETE SET NULL: deleting a penalty now detaches its
-- assignments (penalty_id -> NULL) but keeps them. Since 00025 snapshots
-- amount/label onto the assignment row, a null penalty_id still renders fully
-- from the snapshot; no read path joins penalties for display.
--
-- penalty_id must be nullable for SET NULL to work. DROP NOT NULL is
-- metadata-only (no table scan). The FK is re-added NOT VALID + VALIDATE'd in a
-- separate statement so the migration never holds an ACCESS EXCLUSIVE lock
-- while scanning existing rows (every current assignment already has a valid
-- penalty_id, so VALIDATE is effectively instant) -- the same expand/contract
-- discipline 00018 documents.
ALTER TABLE penalty_assignments ALTER COLUMN penalty_id DROP NOT NULL;
ALTER TABLE penalty_assignments DROP CONSTRAINT penalty_assignments_penalty_id_fkey;
ALTER TABLE penalty_assignments
    ADD CONSTRAINT penalty_assignments_penalty_id_fkey
    FOREIGN KEY (penalty_id) REFERENCES penalties(id) ON DELETE SET NULL NOT VALID;
ALTER TABLE penalty_assignments VALIDATE CONSTRAINT penalty_assignments_penalty_id_fkey;

-- +goose Down

-- Restore ON DELETE CASCADE. penalty_id is intentionally left nullable rather
-- than restored to NOT NULL: a raw SET NOT NULL scans the whole table under an
-- ACCESS EXCLUSIVE lock and would fail outright if any penalty was deleted
-- while this migration was applied (leaving null rows). Old-version code always
-- inserts a non-null penalty_id, so a nullable column is harmless on rollback.
ALTER TABLE penalty_assignments DROP CONSTRAINT penalty_assignments_penalty_id_fkey;
ALTER TABLE penalty_assignments
    ADD CONSTRAINT penalty_assignments_penalty_id_fkey
    FOREIGN KEY (penalty_id) REFERENCES penalties(id) ON DELETE CASCADE NOT VALID;
ALTER TABLE penalty_assignments VALIDATE CONSTRAINT penalty_assignments_penalty_id_fkey;
