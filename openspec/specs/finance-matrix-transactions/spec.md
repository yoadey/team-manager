# finance-matrix-transactions Specification

## Purpose
Defines the finance UI's matrix-first workflow for contributions and transactions: the contributions tab defaults to the member × fee-group matrix (list remains reachable via toggle), the link-picker matrix used from the transaction form renders compactly (no photo, minimal spacing, square selection cells), the transaction form exposes direct fee/penalty linking buttons and an editable date field, the transactions list is filterable by category, and clicking a populated matrix cell always opens that contribution's read-only detail view.

## Requirements

### Requirement: Contribution matrix is the default view
The contributions ("Beiträge") tab MUST default to the matrix view instead of the list view.

#### Scenario: Opening the contributions tab
- **WHEN** a member opens the finances contributions tab
- **THEN** the member x fee-group matrix is shown
- **AND** the list view remains reachable via the view toggle

### Requirement: Compact link-picker matrix
The contribution-linking matrix dialog opened from the transaction form MUST render without a member photo, with minimal inter-column spacing, and without rounded corners on its interior selection cells.

#### Scenario: Opening the link-picker matrix
- **WHEN** a member opens the fee-linking matrix from the transaction form
- **THEN** no member photo/avatar is rendered in the grid
- **AND** the interior cell selection controls have no border radius

### Requirement: Single-step fee/penalty linking
The transaction form's linking control MUST show a "Verknüpfen mit" heading with two direct buttons (fees, penalties) rather than a collapsed toggle that must be expanded first; each button MUST open its own popup for making the selection.

#### Scenario: Linking control is visible without an extra click
- **WHEN** a member opens the new-transaction form with open fees or penalties available
- **THEN** the "Verknüpfen mit" heading and both the fee and the penalty buttons are visible immediately, with no intermediate toggle

#### Scenario: Selecting a fee
- **WHEN** a member clicks the fee-linking button
- **THEN** the fee matrix popup opens for selection

#### Scenario: Selecting a penalty
- **WHEN** a member clicks the penalty-linking button
- **THEN** a popup listing open penalty assignments opens for selection

### Requirement: Transaction date field
The transaction form MUST expose an editable date field, defaulting to today when creating a new transaction and to the transaction's existing date when editing one.

#### Scenario: Creating a transaction
- **WHEN** a member opens the transaction form to create a new transaction
- **THEN** the date field is pre-filled with today's date
- **AND** the member can change it before saving

#### Scenario: Editing a transaction
- **WHEN** a member reopens an existing transaction for editing
- **THEN** the date field is pre-filled with that transaction's stored date

### Requirement: Filter transactions by category
The transactions ("Umsätze") list MUST be filterable by category.

#### Scenario: Filtering by a category
- **WHEN** a member selects a category filter on the transactions list
- **THEN** only transactions with that category are shown

#### Scenario: Clearing the filter
- **WHEN** a member clears the category filter
- **THEN** every transaction in the list is shown again

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
