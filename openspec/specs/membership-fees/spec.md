# membership-fees Specification

## Purpose
Defines how membership fees (contributions) and penalty assignments are defined, paid, and reviewed: a contribution is a free-text name plus amount with an optional due date and no automatic recurrence, can fan out to several members in one request, and its paid amount/status is always derived from the sum of income transactions linked to it rather than stored independently; deleting a contribution or penalty catalog entry preserves its linked/snapshot history, a transaction links to at most one fee or penalty, fee/penalty selection stays searchable at scale, and editing a fee period's shared fields is a group-level action while a single member's row stays a read-only detail view.

## Requirements

### Requirement: Free-text fee name with optional due date
A contribution MUST be defined by a free-text name and an amount, with an
optional due date, instead of a fixed calendar month.

#### Scenario: Creating a fee with a due date
- **WHEN** the treasurer creates a contribution named "Turnieranmeldung
  Frühjahr" with an amount and a due date
- **THEN** the contribution is stored with that name, amount, and due date
- **AND** it has no month field

#### Scenario: Creating a fee without a due date
- **WHEN** the treasurer creates a contribution without specifying a due
  date
- **THEN** the contribution is created successfully with no due date set

### Requirement: No automatic recurrence
The system MUST NOT generate contribution rows automatically for any period.
Each contribution instance is created explicitly.

#### Scenario: Recurring fee across several months
- **WHEN** the treasurer wants the same fee charged in each of several
  months
- **THEN** the treasurer creates a separate contribution for each month
- **AND** no contribution is created by the system on its own

### Requirement: Multi-member fan-out creation
Creating a contribution MUST let the treasurer select one or more members in
a single request, producing one contribution row per selected member sharing
the same name, amount, and due date.

#### Scenario: Assigning a fee to several members at once
- **WHEN** the treasurer creates a contribution and selects three members
- **THEN** three contribution rows are created, one per selected member,
  each with the given name, amount, and due date

#### Scenario: Selecting a non-member
- **WHEN** the treasurer submits a member who does not belong to the team
- **THEN** the request is rejected and no contribution rows are created for
  that request

### Requirement: Paid amount and status derived from linked transactions
A contribution's paid amount MUST be the sum of income transactions linked
to it, and its status (open, partial, or paid) MUST be derived by comparing
that sum to the contribution's amount, rather than stored as an
independently settable value.

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

### Requirement: Deleting a contribution preserves its booked income
Deleting a contribution MUST NOT delete any transaction linked to it; linked
transactions are unlinked instead.

#### Scenario: Deleting a fee with a recorded payment
- **WHEN** the treasurer deletes a contribution that has an income
  transaction linked to it
- **THEN** the contribution is removed
- **AND** the previously linked transaction still exists, no longer linked
  to any contribution

### Requirement: Penalty assignment paid state derived from linked transactions
A penalty assignment's paid amount MUST be the sum of income transactions
linked to it, and its paid state MUST be derived by comparing that sum to
the assignment's amount, rather than stored as an independently settable
value. A new penalty assignment defaults to unpaid.

#### Scenario: A new assignment is unpaid
- **WHEN** a penalty is assigned to a member
- **THEN** the assignment's paid state is false and its paid amount is zero

#### Scenario: Recording full payment of a penalty
- **WHEN** the treasurer books an income transaction for a penalty
  assignment's full amount and links it to that assignment
- **THEN** the assignment's paid state becomes true

#### Scenario: Deleting the linked transaction reverts the paid state
- **WHEN** an income transaction linked to a paid penalty assignment is
  deleted
- **THEN** the assignment's paid state becomes false again

### Requirement: A transaction links to at most one fee or penalty
A transaction MUST NOT be linked to both a contribution and a penalty
assignment at the same time.

#### Scenario: Attempting to link both
- **WHEN** the treasurer attempts to create a transaction with both a
  contribution and a penalty assignment specified
- **THEN** the request is rejected

### Requirement: Searchable link selection at real-club scale
Selecting which fee or penalty a transaction pays MUST remain usable when a
team has many members and many open fees/penalties — the selection MUST NOT
be presented as a single list enumerating every member/fee (or
member/penalty) combination.

#### Scenario: Many members and many open fees
- **WHEN** a team has 40 members and 20 open contributions
- **THEN** the treasurer can find and select the specific member's specific
  fee by searching, without being shown all 800 combinations at once

### Requirement: Contribution description
A contribution MAY carry an optional free-text description (≤2000 characters).
The system MUST store it as provided and MUST make it editable after creation
alongside the contribution's name/amount/due date, via the group-level edit
action described under "Editing a fee period is a group-level action" (not a
per-row update).

#### Scenario: Adding a description to an existing fee
- **WHEN** the treasurer uses the group-level edit action on a fee period and
  sets a description
- **THEN** every row in that group is updated with that description
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

### Requirement: Single-contribution detail view is read-only
Opening a single member's contribution row MUST show a read-only detail
view: the member's name, the paid/required amount, and every transaction
linked to that row. It MUST NOT offer editing of the row's `label`,
`amount`, `description`, or `dueDate`, and MUST NOT offer per-row
archive/unarchive or delete actions.

#### Scenario: Opening a member's contribution row
- **WHEN** a member with finance permissions opens a single contribution row
  (from the list view's row action or a matrix cell)
- **THEN** the detail view shows the member's name, the paid amount, the
  required amount, and the list of transactions linked to that row
- **AND** no field on the row is editable
- **AND** no archive, unarchive, or delete action is shown

#### Scenario: Recording a payment from the detail view
- **WHEN** a member with finance permissions opens the "Beitrag erfassen"
  button in the detail view
- **THEN** the transaction-booking form opens, pre-linked to that
  contribution row
- **AND** this is available regardless of whether a payment has already
  been recorded

### Requirement: Editing a fee period is a group-level action
Changing a contribution's `label`, `amount`, `description`, or `dueDate`
MUST be done via a group-level action that applies the change to every row
sharing that fee period's group key, not by editing an individual member's
row.

#### Scenario: Editing a fee period's shared fields
- **WHEN** a member with finance permissions uses the group-level "Bearbeiten"
  action on a fee period
- **THEN** a form opens prefilled with that group's current `label`,
  `amount`, `description`, and `dueDate`
- **AND** saving it applies the new values to every row in that group
- **AND** the rows remain grouped together afterward (they still share the
  same `label`/`dueDate`)

#### Scenario: Partial failure editing a group
- **WHEN** the group-level edit action fails to update some but not all
  rows in the group
- **THEN** the member is told how many of the group's rows failed to update
- **AND** re-running the edit action is safe
