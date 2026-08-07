## ADDED Requirements

### Requirement: Contribution description
A contribution MAY carry an optional free-text description (≤2000 characters),
editable after creation alongside its name/amount/due date via the existing
per-row update endpoint.

#### Scenario: Adding a description to an existing fee
- **WHEN** the treasurer edits a contribution and sets a description
- **THEN** the contribution is updated with that description
- **AND** the description does not affect the contribution's amount, name, due
  date, or paid status

#### Scenario: Description omitted
- **WHEN** a contribution is created or updated without a description
- **THEN** it has no description
- **AND** this is not an error

### Requirement: Archiving a contribution
A contribution MUST be archivable and un-archivable via the existing per-row
update endpoint, without deleting the row or any transaction linked to it.

#### Scenario: Archiving a no-longer-relevant fee
- **WHEN** the treasurer archives a contribution
- **THEN** the contribution's `archived` flag becomes true
- **AND** any transaction linked to it remains linked and unchanged

#### Scenario: Un-archiving a contribution
- **WHEN** the treasurer un-archives a previously archived contribution
- **THEN** the contribution's `archived` flag becomes false
- **AND** the contribution is indistinguishable from one that was never archived

### Requirement: Archived contributions excluded from open-count aggregate
The open-contributions count (`FinanceOverview.contribOpen`) MUST exclude
archived contributions, regardless of their paid state.

#### Scenario: Archiving an open fee removes it from the open count
- **WHEN** a contribution that is not fully paid is archived
- **THEN** `contribOpen` no longer counts that contribution
