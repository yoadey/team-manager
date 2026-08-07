## ADDED Requirements

### Requirement: Linked entries visible in finance detail views

Each of the three finance detail views (a transaction, a penalty assignment,
a contribution) MUST show which other finance entries it is linked to.

#### Scenario: Transaction linked to a contribution or penalty

- **WHEN** a user opens an existing transaction that has a linked contribution
  or penalty assignment
- **THEN** the transaction's detail view shows the linked contribution's or
  penalty's name and amount

#### Scenario: Contribution with paying transactions

- **WHEN** a user opens a contribution that one or more transactions have
  been booked against
- **THEN** the contribution's detail view lists those transactions

#### Scenario: Penalty assignment with paying transactions

- **WHEN** a user opens a penalty assignment that one or more transactions
  have been booked against
- **THEN** the penalty assignment's detail view lists those transactions

#### Scenario: No linked entries

- **WHEN** a user opens a transaction, contribution, or penalty assignment
  that has no linked entries
- **THEN** the detail view shows an empty state instead of omitting the
  section entirely
