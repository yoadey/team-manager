## Why

RBAC enforcement in `middleware/authz.go` maps URL paths to modules by hand — via string-parsing (`subPathAfterTeam` using `strings.Index`) and several hand-maintained maps (`routeModule`, `knownSettingsSegments`, `selfServiceWritePaths`, `selfServiceModule`). This is a second source of truth beside the OpenAPI spec that already generates the routes, and it is **fail-open for GET**: `readModuleForPath` returns `restrict=false` for unknown segments, so a new module whose segment is forgotten becomes readable by every team member — contradicting the "`none` hides reads too" guarantee. (Writes are fail-closed via 404.)

## What Changes

- Tag every team-scoped operation in `openapi.yaml` with `x-rbac-module` and optional `x-rbac-self-service`.
- Generate an `operationId → {module, selfService}` table from the spec into `rbac_table.gen.go`, wired into `make generate`.
- Rewrite `RequirePermission` to look up the matched route's operation and enforce **fail-closed for all methods, including GET**.
- Remove the hand-maintained path-parsing maps.
- Add a CI check that every team-scoped operation carries an RBAC extension.

## Capabilities

### New Capabilities
- `authorization`: how team-scoped requests are mapped to RBAC modules and gated, with a single spec-derived source of truth and fail-closed defaults.

### Modified Capabilities
<!-- none -->

## Impact

- Backend: `openapi/openapi.yaml` (extensions), generator + `rbac_table.gen.go` (new), `middleware/authz.go` (rewritten mapping, old maps removed), `cmd/server/main.go` if operationId is injected into context, `Makefile`, `.github/workflows/ci.yml` (completeness check), `CLAUDE.md` (RBAC section).
- Security-sensitive: existing authz/IDOR/self-service tests must stay green unchanged.
