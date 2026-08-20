## ADDED Requirements

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
