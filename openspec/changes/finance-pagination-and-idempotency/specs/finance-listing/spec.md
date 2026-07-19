## ADDED Requirements

### Requirement: Paginated finance access without a hard cap
Finance transactions MUST be reachable via cursor-based pagination so no rows become permanently invisible above a fixed cap.

#### Scenario: More rows than the overview page
- **WHEN** a team has more finance transactions than a single overview page
- **THEN** older transactions are reachable by following the pagination cursor

### Requirement: Idempotent paid-state changes
Changing the paid state of a penalty assignment or a contribution MUST be idempotent — the same request applied twice yields the same final state.

#### Scenario: Retried paid update
- **WHEN** a client sets an assignment's paid state to true and retries the same request after a lost response
- **THEN** the assignment ends up paid
- **AND** it is not flipped back to unpaid by the retry
