## Context

These are independent, low-risk cleanups. `toGenRole` copies are byte-identical mappers of `teams.RoleRow → gen.Role`. The `time.Time{}` lines exist to keep an otherwise-unused import; removing the line plus letting `gofumpt` drop the import is the intended fix (the repo already runs gofumpt as a formatter). `CreateAssignment` runs inside a transaction after a `PenaltyBelongsToTeam` service check, but the snapshot SELECT itself is unscoped.

## Goals / Non-Goals

**Goals:**
- One `toGenRole`; no dead-code import hacks; uniform team-scoping; no placeholder domain in prod errors; batched role insert.

**Non-Goals:**
- Broader refactors of the finances/members repositories.
- Changing the permission model or role shape.

## Decisions

- Put the shared mapper as a method `func (r RoleRow) ToGen() gen.Role` (or `teams.ToGenRole`) and call it from events/members/roles/teams.
- `ERROR_TYPE_BASE_URI` default → relative (`/errors/`) or empty; the `backend-lint` "forbid hardcoded type-URI literals" check must still pass. Update `CLAUDE.md` to match actual behavior.
- `SetRoles`: `INSERT INTO membership_roles (membership_id, role_id) SELECT $1, r FROM unnest($2::uuid[]) AS r`, shrinking lock hold time; keep the existing validation cap.
- Removing the `time.Time{}` lines: verify each file still compiles (the import may genuinely be used elsewhere; if so, just delete the dead line).

## Risks / Trade-offs

- Centralizing `toGenRole` creates a small cross-package dependency (already present: all import `teams`/`gen`).
- Changing the error type-URI default alters the `type` field of problem+json responses; ensure no test asserts the example domain.
- The `CreateAssignment` scoping change is defense-in-depth (no known exploit, `team_id` immutable) but aligns with the repo's discipline.
