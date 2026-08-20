## ADDED Requirements

### Requirement: MSW mock backend enforces per-module RBAC

Every team-scoped MSW handler in `frontend/src/mocks/handlers.ts` whose
corresponding OpenAPI operation carries `x-rbac-module` other than
`public` MUST reject the request with 403 when the caller's effective
permission on that module (folded across their roles for the team, max
across roles) falls short of the level the real backend's
`RequirePermission` middleware would require: `read` for a GET/HEAD-
equivalent request or an `x-rbac-self-service: true` route regardless of
method, `write` for every other mutating request. A module permission of
`none` MUST block GET requests, not only mutations. `x-rbac-module:
public` routes remain gated by authentication only, matching the real
backend.

#### Scenario: Module set to none blocks reads
- **WHEN** a caller whose role has a module set to `none` issues a GET
  request against that module's route
- **THEN** the mock responds 403 with an `application/problem+json` body

#### Scenario: Read-only caller cannot mutate
- **WHEN** a caller with only `read` on a module issues a POST, PUT,
  PATCH, or DELETE against a non-self-service route gated by that module
- **THEN** the mock responds 403

#### Scenario: Self-service route allows a read-only caller acting on their own record
- **WHEN** a caller with only `read` on a module issues a mutating request
  against an `x-rbac-self-service: true` route acting on their own record
  (their own member title, their own event comment, their own attendance,
  their own poll vote)
- **THEN** the mock allows the request

#### Scenario: Self-service route still blocks acting on another member's record
- **WHEN** a caller without `write` on the module issues a self-service
  mutating request naming a different member's record than their own
  (e.g. setting another member's event attendance)
- **THEN** the mock responds 403, unless the caller has `write` on the
  module
