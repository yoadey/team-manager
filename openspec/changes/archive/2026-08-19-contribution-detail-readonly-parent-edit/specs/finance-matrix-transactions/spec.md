## RENAMED Requirements
- FROM: `### Requirement: Click a matrix cell to book or inspect a payment`
- TO: `### Requirement: Click a matrix cell to open the contribution detail view`

## MODIFIED Requirements

### Requirement: Click a matrix cell to open the contribution detail view
Clicking a populated matrix cell (a member with a contribution row in that
fee group) MUST always open that row's read-only detail view, regardless of
whether a payment has already been recorded.

#### Scenario: Cell with no payment recorded
- **WHEN** a member clicks a matrix cell whose contribution has
  `paidAmount === 0`
- **THEN** the contribution's read-only detail view opens
- **AND** a "Beitrag erfassen" button in that view opens the
  transaction-creation form pre-linked to that contribution

#### Scenario: Cell with at least one payment recorded
- **WHEN** a member clicks a matrix cell whose contribution has
  `paidAmount > 0`
- **THEN** the contribution's read-only detail view opens
- **AND** it lists the transactions already linked to it

#### Scenario: Empty cell
- **WHEN** a member clicks a matrix cell for a member with no contribution
  row in that fee group
- **THEN** nothing opens
