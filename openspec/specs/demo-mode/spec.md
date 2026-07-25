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

