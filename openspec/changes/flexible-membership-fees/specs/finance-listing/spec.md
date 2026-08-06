## MODIFIED Requirements

### Requirement: Idempotent paid-state changes
Changing the paid state of a penalty assignment MUST be idempotent — the
same request applied twice yields the same final state. A contribution's
paid state is derived from its linked transactions (see the `membership-fees`
capability) rather than being an independently settable value, so this
requirement no longer applies to contributions.

#### Scenario: Retried paid update
- **WHEN** a client sets a penalty assignment's paid state to true and
  retries the same request after a lost response
- **THEN** the assignment ends up paid
- **AND** it is not flipped back to unpaid by the retry
