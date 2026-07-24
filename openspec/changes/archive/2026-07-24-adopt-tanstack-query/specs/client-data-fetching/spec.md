## ADDED Requirements

### Requirement: Server state managed by a query cache
Server-derived data MUST be fetched and cached through a dedicated query cache (TanStack Query), not stored in the global application context. Feature screens MUST read data via query hooks.

#### Scenario: Independent module loading
- **WHEN** the current team is loaded and one module's request fails
- **THEN** the other modules' data still renders from their own queries
- **AND** no manual `Promise.allSettled` orchestration is required in the context

### Requirement: Team-scoped cache prevents stale cross-team data
Query cache keys MUST be scoped by team id so that switching the active team cannot surface data fetched for a previous team.

#### Scenario: Rapid team switch
- **WHEN** the user switches from team A to team B while A's request is in flight
- **THEN** B's screen shows only B's data
- **AND** a late response for A does not overwrite B's cache

### Requirement: Per-operation pending state
Mutation in-flight state MUST be tracked per operation, not by a single shared flag, so concurrent actions of different kinds cannot re-enable each other's controls.

#### Scenario: Concurrent save and delete
- **WHEN** a save and a delete of different kinds are pending at once
- **THEN** each control reflects only its own pending state
- **AND** neither clears the other's disabled/spinner state

### Requirement: Auth failures are not retried
The retry policy MUST NOT retry responses representing authentication (401), authorization (403) or validation (422) failures.

#### Scenario: Forbidden response
- **WHEN** a query receives a 403 response
- **THEN** it fails immediately without retrying
