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

### Requirement: Client-settable transaction date
A finance transaction's date MUST be settable by the client (e.g. to back-date a receipt), defaulting to the server's current date when omitted.

#### Scenario: Back-dated transaction
- **WHEN** a client creates a transaction with an explicit past date
- **THEN** the stored transaction carries that date rather than today's

#### Scenario: Omitted date
- **WHEN** a client creates a transaction without a date
- **THEN** the transaction is stamped with the server's current date
