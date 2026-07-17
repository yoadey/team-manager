## 1. Spec annotations
- [x] 1.1 Add `x-rbac-module` / `x-rbac-self-service` to every team-scoped operation in `openapi.yaml`, mirroring today's `authz.go` behavior exactly
- [x] 1.2 Confirm `nominations` is `x-rbac-module: events` with NO self-service flag; `roles`/`photo`/`logo`/`invite` → settings (or documented membership-only)
- [x] 1.3 Map self-service-with-read-gate (attendance, comments, vote) vs self-service-without (absences, notifications/seen — both modeled as `x-rbac-module: public`, since neither carried a module read-gate under the old code either)

## 2. Generator
- [x] 2.1 Add a generator (`cmd/genrbac`) that parses `openapi.yaml` and emits `internal/middleware/rbac_table.gen.go` (`{Method, Segments} → {Module, SelfService}`, keyed by operation rather than bare `operationId` since matching happens against the request's method+path)
- [x] 2.2 Wire it into `make generate` (after oapi-codegen)

## 3. Middleware rewrite
- [x] 3.1 Resolve the request to its RBAC entry via a generated segment-matcher (`matchRBACRoute`) instead of chi-route/operationId context injection — matches the real request path against each table entry's path-parameter-aware segment template, so it works identically in unit tests (no real chi routing needed) and in production
- [x] 3.2 Rewrite `RequirePermission`: `module: public` → pass; self-service or GET/HEAD/OPTIONS → require read; mutation → require write; unmatched method+path → 404 for all methods
- [x] 3.3 Remove `routeModule`, `knownSettingsSegments`, `selfServiceWritePaths*`, `selfServiceModule`, `readModuleForPath`, `moduleForPath`, `isMemberRolesPath`, `selfServiceLeaf`. `subPathAfterTeam` was kept (unlike the task's original wording assumed) — it's still needed to extract the segment string the generated table is matched against

## 4. CI guard
- [x] 4.1 `cmd/genrbac` itself fails (non-zero exit) if any team-scoped operation lacks `x-rbac-module`, and `make generate` (which runs it) is already a required CI step in `backend-openapi-drift` — no separate CI job needed

## 5. Verification
- [x] 5.1 `make generate` → `rbac_table.gen.go` current, no diff; `make generate-ts` produces no diff to `types.gen.ts` (x-rbac-* extensions don't affect the OpenAPI schema types)
- [x] 5.2 `make lint`, `make test` green — all existing authz/IDOR/self-service/nominations tests pass; 3 pre-existing tests were updated because they exercised fictitious routes (`POST /photo`/`/logo`, `POST /members`, `GET /finances/transactions`) that don't exist in `openapi.yaml` and only "passed" before because the old first-segment matching didn't verify method+path against real operations — real routes are unaffected
- [x] 5.3 New negative test `TestRequirePermission_GET_UnknownRoute_Returns404` — a team-scoped GET route with no table entry now returns 404, not 200 (this is the fail-open bug this change fixes)
- [x] 5.4 Removing an `x-rbac-*` extension makes `make generate` (hence CI) fail with a clear error listing every operation missing the extension
- [x] 5.5 Generated table (`internal/middleware/rbac_table.gen.go`) excluded from golangci-lint like `internal/gen/`; contributes no executable statements so it's coverage-neutral without needing a coverage-gate exclusion; `cmd/genrbac` itself is covered by `cmd/genrbac/main_test.go`
