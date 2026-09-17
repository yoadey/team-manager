# demo-mode Specification

## Purpose
Defines how the frontend serves a backend-less demo/test experience: MSW intercepts the generated API client at the network layer against OpenAPI-shaped handlers, so there is a single business-logic implementation (the real backend) instead of a second one duplicated in-browser, and production builds fail safe rather than silently booting a mock.

## Requirements

### Requirement: Single client implementation
The application MUST use exactly one implementation of the API contract (`realApi`, the generated HTTP client) across production, development-demo, and test environments. A backend-less demo MUST be provided by intercepting HTTP requests (MSW), not by a second in-code business-logic implementation.

#### Scenario: Demo mode serves through the real client
- **WHEN** the app runs without a configured `API_BASE_URL` in development
- **THEN** requests go through the generated `openapi-fetch` client and are intercepted by MSW handlers
- **AND** no separate `_mockApi` implementation is invoked

### Requirement: Production fail-safe against unconfigured backend
A production build MUST refuse to fall back to the demo/mock backend. If `API_BASE_URL` is unset and `VITE_ALLOW_MOCK` is not explicitly enabled, startup MUST fail loudly.

#### Scenario: Prod build without backend URL
- **WHEN** a production build starts with an empty `API_BASE_URL` and no `VITE_ALLOW_MOCK`
- **THEN** the app throws a visible configuration error
- **AND** it does not boot the demo backend or a password-less admin session

### Requirement: Demo artifacts excluded from production bundle
The mock handlers and seed data MUST NOT be present in production JavaScript bundles.

#### Scenario: Seed data is tree-shaken
- **WHEN** a production bundle is built
- **THEN** the demo seed identifiers (e.g. sample member names) do not appear in any emitted chunk
- **AND** the `msw` package is not included in the production dependency graph

### Requirement: No unused configuration from removed implementations
Environment variables that no code path reads MUST NOT remain parsed into
application config, so `.env` documentation accurately reflects what
affects runtime behavior.

#### Scenario: A demo-mode implementation is replaced
- **WHEN** an implementation (e.g. the localStorage mock) is removed and
  replaced by another
- **THEN** any environment variable only that removed implementation
  consumed is deleted from config parsing, `.env.example`, and CLAUDE.md's
  env-var documentation in the same change

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
