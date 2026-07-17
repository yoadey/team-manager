## Context

`RequirePermission` computes the sub-path after `{teamId}` and looks it up in `routeModule`/`selfService*` maps. Reads for unmapped segments fall through (`restrict=false`); writes 404. The `nominations` sub-path was historically misclassified as self-service and must remain events:write-only.

## Goals / Non-Goals

**Goals:**
- One source of truth: the OpenAPI spec, via `x-rbac-module` / `x-rbac-self-service` extensions.
- Fail-closed for all methods: an operation with no RBAC classification is rejected, including GET.
- Behavior-preserving otherwise: no existing authorization decision changes except the GET fail-open fix.

**Non-Goals:**
- Introducing an external policy engine (Cerbos/OpenFGA) — spec-generation is the proportionate choice here.
- Changing the permission model (`none < read < write`) or module set.

## Decisions

- A small generator parses `openapi.yaml` and emits `middleware/rbac_table.gen.go` mapping `operationId → {Module, SelfService}`; wired into `make generate`.
- The middleware resolves the matched chi route to its operationId (via a thin per-handler context injection or a generated pattern→operationId map) and enforces:
  - self-service → pass, but still require module `read` where a module is set;
  - GET/HEAD/OPTIONS → require `read` on the module;
  - mutations → require `write`;
  - unknown operation → 404 (fail-closed for every method).
- A CI completeness check fails the build if any team-scoped operation lacks an RBAC extension.

## Risks / Trade-offs

- **Behavior parity is mandatory**: the only intended change is unknown-GET becoming fail-closed. Any diff in existing authz/IDOR/self-service tests is a bug, not an expected update.
- `nominations` must be `x-rbac-module: events` with **no** self-service flag.
- Generated table must be checked in and drift-free; exclude it from coverage like other `*.gen.go`.
