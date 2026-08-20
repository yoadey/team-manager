# finance-listing Specification

## Purpose
TBD - created by archiving change finance-pagination-and-idempotency. Update Purpose after archive.

## Requirements

### Requirement: Paginated finance access without a hard cap
Finance transactions MUST be reachable via cursor-based pagination so no rows become permanently invisible above a fixed cap.

#### Scenario: More rows than the overview page
- **WHEN** a team has more finance transactions than a single overview page
- **THEN** older transactions are reachable by following the pagination cursor

### Requirement: Client-settable transaction date
A finance transaction's date MUST be settable by the client (e.g. to back-date a receipt), defaulting to the server's current date when omitted.

#### Scenario: Back-dated transaction
- **WHEN** a client creates a transaction with an explicit past date
- **THEN** the stored transaction carries that date rather than today's

#### Scenario: Omitted date
- **WHEN** a client creates a transaction without a date
- **THEN** the transaction is stamped with the server's current date

### Requirement: Contribution matrix view
The finance UI MUST offer a member × fee-group matrix view of contributions,
alongside the existing list view, showing each member's status for each
non-archived fee group in a single grid.

#### Scenario: Switching to the matrix view
- **WHEN** the treasurer switches `Finanzen -> Beiträge` from list to matrix view
- **THEN** a grid is shown with one row per member who has at least one
  non-archived contribution and one column per non-archived fee group
- **AND** each cell reflects that member's paid/partial/open/overpaid status
  for that fee group, consistent with the same cell's status in the list view

#### Scenario: Archived fee groups excluded from the matrix
- **WHEN** a fee group's rows are all archived
- **THEN** that fee group does not appear as a column in the matrix

### Requirement: Overpayment is visible, not capped at the amount due
Wherever a contribution's paid amount is displayed, it MUST reflect the
actual paid amount, including any amount paid in excess of what was due —
not the fee's nominal amount.

#### Scenario: Member pays more than the fee amount
- **WHEN** a member's linked transactions sum to more than the contribution's
  amount
- **THEN** the contribution's status is shown distinctly as overpaid
- **AND** the displayed amount reflects the actual amount paid, including the
  excess, rather than being capped at the fee's amount

### Requirement: Matrix-based linking of a transaction to a contribution
Linking a new income transaction to the contribution it pays MUST offer a
member × fee-group matrix of single-selectable, non-archived, not-yet-fully-paid
contributions, showing each selectable cell's still-owed amount.

#### Scenario: Selecting a fee to link via the matrix
- **WHEN** the treasurer records a new income transaction and opens the
  contribution-linking matrix
- **THEN** only non-archived contributions that are not yet fully paid are
  selectable
- **AND** selecting a cell links the transaction to that member's contribution
  for that fee group
- **AND** the cell shows the amount still owed for that member's fee

### Requirement: Optional transaction note, hidden from the list
A finance transaction MAY carry an optional free-text note (≤10000
characters). The system MUST accept it on creation and update, MUST store it
as provided, and MUST NOT render it in the transaction list view.

#### Scenario: Recording a note on a transaction
- **WHEN** a treasurer adds a note while creating or editing a transaction
- **THEN** the note is stored with the transaction
- **AND** the transaction list view does not display the note
- **AND** the note is shown when the transaction is reopened for editing

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
