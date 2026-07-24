## Why

Several small backend code-quality findings from the audit remain and add up to avoidable maintenance risk:

- **`var _ = time.Time{}` dead-code import hacks** in 4 files (`auth/handler.go:379`, `events/service.go:751`, `members/service.go:219`, `members/handler.go:270`).
- **`toGenRole` duplicated 4×** (`teams/service.go:414`, `events/service.go:712`, `roles/service.go:84`, `members/service.go:200`) — a new permission module must be updated in four places.
- **`CreateAssignment` snapshot read is not team-scoped**: `SELECT amount, label FROM penalties WHERE id = $1` (`finances/repository.go:471`) lacks `AND team_id = $2`; safe today only via a prior unlocked service check — a break from the repo's otherwise strict in-query scoping.
- **`ERROR_TYPE_BASE_URI` defaults to a live example domain** `https://teammanager.example/errors/` (`apierror/apierror.go:17`), leaking a placeholder into production problem+json and contradicting the docs.
- **`SetRoles` inserts membership roles one row at a time under an exclusive advisory lock** (`members/repository.go:382`) — the repo already has the `UNNEST` batch pattern elsewhere.

## What Changes

- Remove the `time.Time{}` hacks and their now-unused imports (let `gofumpt` handle imports).
- Centralize `toGenRole` into one shared mapper (in `teams`, where `RoleRow` lives).
- Add `AND team_id = $2` to the `CreateAssignment` snapshot read.
- Make `ERROR_TYPE_BASE_URI` default to relative paths (or empty) instead of the example domain; align the docs.
- Batch the `SetRoles` insert via `UNNEST`.

## Capabilities

### New Capabilities
- `backend-consistency`: shared role mapping, uniformly tenant-scoped queries, and no placeholder domains in error responses.

### Modified Capabilities
<!-- none -->

## Impact

- Backend: the four files above, `finances/repository.go`, `members/repository.go`, `apierror/apierror.go`, plus one shared mapper location; tests. `CLAUDE.md` `ERROR_TYPE_BASE_URI` note.
- No API/schema change; no migration.
