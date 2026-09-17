## MODIFIED Requirements

### Requirement: Paid amount and status derived from linked transactions
A contribution's paid amount MUST be the sum of income transactions linked
to it, and its status (open, partial, or paid) MUST be derived by comparing
that sum to the contribution's amount, rather than stored as an
independently settable value. Editing a linked transaction's `type` away
from `income` MUST be rejected while the link still exists, so a
contribution's or penalty assignment's paid amount can never be silently
reduced by an edit that looks unrelated to the link itself.

#### Scenario: Recording a partial payment
- **WHEN** the treasurer books an income transaction for less than a
  contribution's full amount and links it to that contribution
- **THEN** the contribution's paid amount reflects the linked transaction's
  amount
- **AND** the contribution's status is "partial"

#### Scenario: Completing payment across multiple transactions
- **WHEN** the treasurer links a second income transaction to the same
  contribution, and the sum of both linked transactions equals or exceeds
  the contribution's amount
- **THEN** the contribution's status is "paid"

#### Scenario: Deleting a linked transaction
- **WHEN** an income transaction linked to a contribution is deleted
- **THEN** the contribution's paid amount no longer includes that
  transaction's amount

#### Scenario: Linking a non-income transaction
- **WHEN** the treasurer attempts to link an expense transaction to a
  contribution
- **THEN** the request is rejected

#### Scenario: Changing a linked transaction's type away from income
- **WHEN** the treasurer edits a transaction that has a `contributionId` or
  `penaltyAssignmentId` set, changing its `type` from `income` to `expense`
- **THEN** the request is rejected with a 400 error
- **AND** the transaction's type, link, and the contribution's/assignment's
  paid amount are unchanged

#### Scenario: Changing an unlinked transaction's type
- **WHEN** the treasurer edits a transaction that has no `contributionId`
  and no `penaltyAssignmentId`, changing its `type`
- **THEN** the request succeeds and the transaction's type is updated

#### Scenario: Changing the amount of a linked transaction
- **WHEN** the treasurer edits a transaction that has a `contributionId` or
  `penaltyAssignmentId` set, changing its `amount` but not its `type`
- **THEN** the request succeeds
- **AND** the linked contribution's or penalty assignment's paid amount
  reflects the new amount
