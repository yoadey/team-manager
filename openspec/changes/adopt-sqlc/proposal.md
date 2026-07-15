## Why

The repository layer uses raw pgx with hand-written SQL. Two recurring weaknesses: hand-rolled dynamic SET builders (`events/repository.go:514-574`, several in `finances/repository.go`) whose positional-arg + no-op `SET id = $N` fallback design has already nearly produced a SQL corruption (a regression test exists), and `fmt.Sprintf` in query strings (currently safe — only fixed internal constants + `$N` — but fragile and a recurring review theme). Adopting sqlc gives type-safe generated Go from SQL, matching the existing spec-first codegen philosophy.

## What Changes

- Add sqlc (pinned) and wire `sqlc generate` into `make generate`.
- Migrate **static** queries to sqlc-generated, type-safe functions (`internal/db/gen`).
- Replace hand-rolled dynamic SET builders with a small, unit-tested `sqlbuilder` (dynamic queries stay pgx, not forced into sqlc).
- Preserve team-scoping and error-wrapping discipline throughout.

## Capabilities

### New Capabilities
- `data-access`: how repositories issue database queries — static queries type-generated from SQL, dynamic queries via a tested builder — while preserving tenant scoping.

### Modified Capabilities
<!-- none -->

## Impact

- Backend: `sqlc.yaml` (new), `internal/db/queries/*.sql` (new), generated `internal/db/gen/*` (checked in), `internal/db/sqlbuilder/*` (new), migrated `repository.go` files, `Makefile` (`tools` + `generate`), Dockerfile if tools installed there.
- CI: tool-pin sync check, golangci-lint, coverage gate, govulncheck; optional `sqlc`-drift job.
