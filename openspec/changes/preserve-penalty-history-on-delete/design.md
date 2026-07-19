## Context

`penalty_assignments` carries snapshot `amount`/`label` since `00025`. Reads (`ListAssignments`, `GetAssignmentByID`, `ListOpenPenaltiesByUser`) already use the snapshot columns, not a live join to `penalties`. The FK currently cascades deletes. Highest migration on `main` is `00026`, so the next is `00027`.

## Goals / Non-Goals

**Goals:**
- Deleting a catalog penalty must not remove or alter existing assignments (paid or unpaid).
- Keep the assignment fully displayable from its snapshot after the catalog entry is gone.

**Non-Goals:**
- Changing how assignments are created or how amounts are snapshotted.
- Soft-deleting penalties (out of scope; SET NULL is sufficient).

## Decisions

- **FK → `ON DELETE SET NULL`**, `penalty_id` becomes nullable. Migration `00027` drops and re-adds the constraint; the column already has data so this is a metadata + null-allow change (no table rewrite). Follow the repo's migration-safety conventions (constraint changes done safely; annotate if needed).
- Reads must treat `penalty_id` as optional; display uses the snapshot `label`/`amount`. Confirm no read path dereferences `penalty_id` expecting a live penalty row.
- **Additional guard (implement):** in the delete path, still allow deletion (SET NULL preserves history), but ensure the finance overview's open/paid sums are computed from assignments' snapshots, unaffected by the now-null `penalty_id`.
- Reject-if-unpaid is considered but **not** chosen: SET NULL preserves history for both paid and unpaid, and blocking deletion would be a more surprising UX; documented as the rejected alternative.

## Risks / Trade-offs

- Migration must be reversible (down: restore `NOT NULL` + CASCADE only if no null rows exist; document the caveat).
- sqlc-generated models for `penalty_assignments` may need reguration if `penalty_id` becomes nullable in a generated struct; run `make generate` and commit.
- Any code assuming a non-null `penalty_id` must be updated; audit read/aggregate paths.
