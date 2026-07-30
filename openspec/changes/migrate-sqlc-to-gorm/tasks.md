## 1. Dependencies & driver wiring
- [ ] 1.1 Add `gorm.io/gorm` and `gorm.io/driver/postgres` to `go.mod`
- [ ] 1.2 Add a `gorm.Open` constructor to `internal/db/db.go` (next to `Connect`) that wraps the existing `*pgxpool.Pool` via `stdlib.OpenDBFromPool`, so pool config, `SlowQueryTracer`, and the Prometheus pool-stats collector are unaffected
- [ ] 1.3 Wire the new `*gorm.DB` through `cmd/server/main.go` alongside the existing `*pgxpool.Pool` (both coexist until the last repository is migrated)

## 2. Core `internal/db/gormx` package
- [ ] 2.1 `TeamScoped` interface + `ForTeam(teamID uuid.UUID)` scope helper
- [ ] 2.2 Scope-guard callback (`RegisterScopeGuard`) that fails a query/update/delete against a `TeamScoped` model when no `team_id` predicate is present; unit tests covering both the guarded-failure and correctly-scoped-success paths
- [ ] 2.3 `Patch` helper (`NewPatch`, `Set`, `Empty`, `Values`) with the same explicit-empty-set contract as `sqlbuilder.Builder`; unit tests including the empty-patch case
- [ ] 2.4 `gorm.ErrRecordNotFound` → domain-sentinel error translation helper (or documented per-module `errors.Is` pattern replacing today's `pgx.ErrNoRows` checks)
- [ ] 2.5 `roles.PermissionsJSON` gains `Scan`/`Value` methods for JSONB round-tripping; round-trip unit test
- [ ] 2.6 Confirm `uuid.UUID`'s existing `Scan`/`Value` methods round-trip correctly through the GORM/postgres driver with a unit test (expected to need no new code — verification only)

## 3. Pilot: `news`, `roles`
- [ ] 3.1 Define GORM models for `news`/`roles` tables (add `gorm:"column:...,primaryKey"` tags to existing `Row` structs or introduce parallel GORM-tagged structs, whichever keeps domain-layer types unchanged)
- [ ] 3.2 Migrate `news/repository.go` off `internal/db/gen` to GORM; keep the same public method signatures
- [ ] 3.3 Migrate `roles/repository.go` off `internal/db/gen` + `sqlbuilder` to GORM + `gormx.Patch`
- [ ] 3.4 Port/confirm each module's cross-team-isolation test (the `data-access` spec's "Cross-team lookup" scenario) against the GORM repositories
- [ ] 3.5 `make test` (unit + integration) green for both modules; `make lint` green

## 4. Wave 2 — zero/near-zero dynamic-query surface: `stats`, `notifications`, `push`, `calendarfeed`
- [ ] 4.1 Migrate `stats/repository.go` (read-only aggregation queries — explicit `Joins`/`Select`, no `Preload`)
- [ ] 4.2 Migrate `notifications/repository.go`
- [ ] 4.3 Migrate `push/repository.go`
- [ ] 4.4 Migrate `calendarfeed/repository.go`
- [ ] 4.5 Cross-team-isolation test per module; `make test`/`make lint` green

## 5. Wave 3 — light dynamic-query surface: `polls`, `auth`, `absences`, `calendarshare`
- [ ] 5.1 Migrate `polls/repository.go` (vote/option fan-out queries stay explicit, no association auto-loading)
- [ ] 5.2 Migrate `auth/repository.go`
- [ ] 5.3 Migrate `absences/repository.go`
- [ ] 5.4 Migrate `calendarshare/repository.go`
- [ ] 5.5 Cross-team-isolation test per module; `make test`/`make lint` green

## 6. Wave 4 — heavier dynamic-query surface: `members`, `teams`
- [ ] 6.1 Migrate `members/repository.go`; patch-field updates (member profile fields) go through `gormx.Patch`
- [ ] 6.2 Migrate `teams/repository.go`; patch-field updates go through `gormx.Patch`
- [ ] 6.3 Cross-team-isolation test per module; `make test`/`make lint` green

## 7. Wave 5 — highest risk: `finances`, `events`
- [ ] 7.1 Migrate `finances/repository.go` off `sqlbuilder` to `gormx.Patch` for transaction/penalty/contribution patches
- [ ] 7.2 Migrate `events/repository.go` off `sqlbuilder` to `gormx.Patch`, including series-update handling
- [ ] 7.3 Port `TestEventRepository_UpdateEvent_Series_OnlyDateSet_DoesNotCorruptSQL` to the GORM repository — MUST stay green, proving the original bug class is closed under the new implementation
- [ ] 7.4 Cross-team-isolation test per module; `make test`/`make lint` green

## 8. Cleanup
- [ ] 8.1 Delete `internal/db/sqlbuilder/` (no remaining callers after wave 5)
- [ ] 8.2 Delete `backend/sqlc.yaml`, `internal/db/queries/`, `internal/db/gen/`
- [ ] 8.3 `backend/Makefile`: remove `SQLC` var, the `sqlc` install line from `tools`, and the `$(SQLC) generate` line from `generate`
- [ ] 8.4 `.github/workflows/ci.yml`: remove `backend-openapi-drift`'s "Install sqlc" step and drop `internal/db/gen` from the drift-check file list/message; remove `backend-lint`'s sqlc-pin-sync assertion block
- [ ] 8.5 Remove the now-unused `*pgxpool.Pool`-only code paths from repositories (constructors etc.) where fully superseded by `*gorm.DB`; keep `internal/db.Connect`/pool itself (still needed for `gorm.Open`'s `stdlib.OpenDBFromPool` bridge, goose, metrics, River)
- [ ] 8.6 Update `CLAUDE.md` (Architecture → Backend, Repository Structure tree) and `openspec/config.yaml` (`context:` backend description) to describe GORM instead of sqlc

## 9. Verification
- [ ] 9.1 `make build` succeeds
- [ ] 9.2 `make test` (unit + integration) green across all 14 migrated modules
- [ ] 9.3 `make lint` (golangci-lint) green
- [ ] 9.4 `make vuln` (govulncheck) green
- [ ] 9.5 CI `backend-openapi-drift` green with the sqlc step removed (only oapi-codegen/genrbac remain)
- [ ] 9.6 CI migration-rollback (goose up → down → up) unaffected — confirms GORM's schema-mapping-only role didn't touch migrations
- [ ] 9.7 CI migration-safety (unsafe-DDL lint) unaffected
- [ ] 9.8 CI coverage gate (>=35% business logic) green with `internal/db/gormx` and all repositories counted (not excluded as generated code)
- [ ] 9.9 `openspec validate migrate-sqlc-to-gorm --strict` passes
