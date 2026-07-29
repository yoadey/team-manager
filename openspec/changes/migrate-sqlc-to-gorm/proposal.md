## Why

The repository layer is currently split across three patterns: 2 of 14 modules (`news`, `roles`) use sqlc-generated static queries, `events`/`finances`/`roles` build dynamic `UPDATE ... SET` clauses through the hand-written `internal/db/sqlbuilder` package, and the remaining 11 modules still hand-write raw pgx (35 files). This mixed state was a deliberate, staged rollout (`openspec/changes/archive/2026-07-18-adopt-sqlc`), but the team has now decided the three-pattern split itself is the bigger cost versus sqlc's compile-time query checking, and wants one consistent data-access layer across the whole backend: GORM.

This change replaces sqlc + hand-written pgx + `sqlbuilder` with GORM everywhere, while explicitly re-closing the two bug classes that motivated the original sqlc adoption: (1) dynamic `SET`-clause construction with a no-op fallback that can silently bind to the wrong column (guarded today by `TestEventRepository_UpdateEvent_Series_OnlyDateSet_DoesNotCorruptSQL`), and (2) missing tenant (`team_id`) scoping on a by-id query. Both are addressed structurally in the design below, not just by convention.

## What Changes

- Add `gorm.io/gorm` + `gorm.io/driver/postgres` and migrate all 14 `repository.go` files from sqlc-generated calls / raw pgx to GORM, module by module, in risk order (lowest dynamic-SQL surface first, `events`/`finances` last).
- Introduce `internal/db/gormx`, a new small package providing:
  - a `TeamScoped` marker + a mandatory `ForTeam(teamID)` scope helper, enforced by a registered GORM callback so a team-scoped query issued without it fails closed instead of silently returning cross-team rows;
  - a `Patch` helper that builds an explicit `map[string]any` of only the changed columns (replacing `sqlbuilder.Builder` with the same "no implicit fallback" discipline, but without manual placeholder-index bookkeeping — GORM binds parameters itself, which removes the original bug's root cause outright);
  - `gorm.ErrRecordNotFound` → existing domain-sentinel error translation, replacing today's `pgx.ErrNoRows` translation at each repository.
- Keep `gorm.io/driver/postgres` on top of the **existing** `*pgxpool.Pool` via `stdlib.OpenDBFromPool` (the same bridge `internal/db.RunMigrations` already uses for goose) so the current pool sizing, the `SlowQueryTracer` (`internal/db/tracer.go`), and the Prometheus pool-stats collector (`internal/db/metrics.go`) keep working unchanged.
- Keep goose + `internal/db/migrations/*.sql` as the schema source of truth; GORM is not used for `AutoMigrate`.
- Remove sqlc once migration is complete: `backend/sqlc.yaml`, `internal/db/queries/`, `internal/db/gen/`, the `sqlc` tool pin in `Makefile`/`ci.yml`, and the `internal/db/sqlbuilder` package.
- Port the existing dynamic-update corruption regression test to its GORM equivalent for `events` and `finances`, and add an equivalent per-module cross-team-isolation test for every migrated repository (the same scenario the `data-access` spec already requires).

## Capabilities

### Modified Capabilities
- `data-access`: static queries are no longer sqlc-generated; dynamic queries no longer go through `sqlbuilder`. Both requirements are replaced by GORM-based equivalents that preserve the same guarantees (type-safety at the Go-struct level instead of at SQL-compile time, explicit-column dynamic updates, and tenant scoping — now structurally enforced via a callback instead of only by SQL text review).

## Impact

- Backend, all 14 modules: `internal/{teams,auth,stats,calendarshare,absences,notifications,events,news,finances,push,members,roles,polls,calendarfeed}/repository.go` (and their `*_test.go`).
- New: `internal/db/gormx/` (scoping callback, `Patch` helper, error translation, unit tests).
- Removed: `backend/sqlc.yaml`, `internal/db/queries/`, `internal/db/gen/`, `internal/db/sqlbuilder/`.
- `go.mod`/`go.sum`: add `gorm.io/gorm`, `gorm.io/driver/postgres`; remove the `sqlc` tool pin from `Makefile`.
- `backend/Makefile`: `tools`/`generate` targets drop the `sqlc` install/run step.
- `.github/workflows/ci.yml`: `backend-openapi-drift` drops its "Install sqlc" step and the `internal/db/gen` drift check; `backend-lint`'s tool-pin-sync check drops its sqlc-pin assertion.
- `internal/db/db.go`: add a `gorm.Open(...)` constructor next to `Connect`, reusing the same `*pgxpool.Pool`.
- Docs: `CLAUDE.md` (Architecture → Backend section, Repository Structure tree), `openspec/config.yaml` (`context:` backend description and the "dependency-light" convention note, justified here per that same rule).
