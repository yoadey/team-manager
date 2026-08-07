## MODIFIED Requirements

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
  been recorded against the row

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
