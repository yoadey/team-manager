## 1. Spec annotations
- [ ] 1.1 Add `x-rbac-module` / `x-rbac-self-service` to every team-scoped operation in `openapi.yaml`, mirroring today's `authz.go` behavior exactly
- [ ] 1.2 Confirm `nominations` is `x-rbac-module: events` with NO self-service flag; `roles`/`photo`/`logo`/`invite` → settings (or documented membership-only)
- [ ] 1.3 Map self-service-with-read-gate (attendance, comments, vote) vs self-service-without (absences, notifications/seen)

## 2. Generator
- [ ] 2.1 Add a generator that parses `openapi.yaml` and emits `internal/middleware/rbac_table.gen.go` (`operationId → {Module, SelfService}`)
- [ ] 2.2 Wire it into `make generate` (after oapi-codegen)

## 3. Middleware rewrite
- [ ] 3.1 Resolve the matched chi route to its operationId (context injection or generated pattern map)
- [ ] 3.2 Rewrite `RequirePermission`: self-service → pass (read-gate if module set); GET → require read; mutation → require write; unknown op → 404 for all methods
- [ ] 3.3 Remove `routeModule`, `knownSettingsSegments`, `readModuleForPath`, `subPathAfterTeam` and the self-service maps superseded by the table

## 4. CI guard
- [ ] 4.1 Add a check that fails the build if any team-scoped operation lacks an RBAC extension

## 5. Verification
- [ ] 5.1 `make generate` → `rbac_table.gen.go` current, no diff; openapi-drift green (also `make generate-ts`)
- [ ] 5.2 `make lint`, `make test` green — all existing authz/IDOR/self-service/nominations tests unchanged
- [ ] 5.3 New negative test: a team-scoped route with no table entry → GET returns 404/403, not 200
- [ ] 5.4 Removing an `x-rbac-*` extension makes CI red (completeness guard works)
- [ ] 5.5 Coverage gate green (generated table excluded like other `*.gen.go`)
