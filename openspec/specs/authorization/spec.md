# authorization Specification

## Purpose
Defines how team-scoped API requests are authorized: the route-to-RBAC-module mapping is generated from `openapi.yaml`'s `x-rbac-module` / `x-rbac-self-service` extensions (via `cmd/genrbac` into `internal/middleware/rbac_table.gen.go`), and `RequirePermission` enforces it fail-closed for every HTTP method, including GET.
## Requirements
### Requirement: Route-to-module mapping derived from the spec
The mapping from a team-scoped route to its RBAC module and self-service classification MUST be generated from the OpenAPI specification (via `x-rbac-module` / `x-rbac-self-service` extensions), not hand-maintained in the middleware.

#### Scenario: Regenerated table matches the spec
- **WHEN** `make generate` runs after editing an operation's RBAC extension
- **THEN** the generated authorization table reflects the extension
- **AND** there is no hand-edited path-parsing map providing a competing mapping

### Requirement: Fail-closed authorization for all methods
An authenticated team-scoped request whose operation has no RBAC classification MUST be rejected, for every HTTP method including GET.

#### Scenario: Unknown GET route
- **WHEN** a GET request targets a team-scoped operation with no RBAC classification
- **THEN** the request is rejected (not served on membership alone)

#### Scenario: Read gating unchanged for known modules
- **WHEN** a member with `none` on a module issues a GET for that module's data
- **THEN** the request is forbidden, exactly as before

### Requirement: Behavior parity for existing decisions
Except for unknown-route GET becoming fail-closed, every existing authorization decision MUST be preserved, including that nomination is an events:write-only action and not self-service.

#### Scenario: Nomination stays write-only
- **WHEN** a non-writer attempts to nominate another member for an event
- **THEN** the request is forbidden

