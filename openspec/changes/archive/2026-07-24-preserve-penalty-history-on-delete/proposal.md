## Why

Migration `00025` snapshotted a penalty's `amount`/`label` onto each `penalty_assignments` row precisely so assignments are an immutable financial record. But `DeletePenalty` (`finances/repository.go:333`) still does `DELETE FROM penalties WHERE id = $1 AND team_id = $2`, which cascades to **all** assignments — including paid, historical ones. A treasurer tidying the catalog silently erases paid penalty history, and the finance overview changes retroactively. The snapshot's whole purpose is defeated by one click.

## What Changes

- Change the `penalty_assignments.penalty_id` foreign key from `ON DELETE CASCADE` to `ON DELETE SET NULL` (migration `00027`), and make `penalty_id` nullable. The snapshot (`amount`/`label`) already carries everything needed to display the assignment after its catalog entry is gone.
- Adjust repository/service reads to tolerate a null `penalty_id` (they already read the snapshot columns).
- Alternative/stricter guard also specified: optionally refuse deletion while unpaid assignments exist — decided in design.

## Capabilities

### New Capabilities
- `penalty-history`: how deleting a penalty catalog entry preserves already-issued (especially paid) assignments as an immutable record.

### Modified Capabilities
<!-- none -->

## Impact

- Backend: migration `00027_penalty_fk_set_null.sql` (up/down), `finances/repository.go` (nullable `penalty_id` reads), possibly sqlc query/model regen, finances tests.
- CI gates: migration-safety lint, migration-rollback, backend test/lint.
