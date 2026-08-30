# penalty-history Specification

## Purpose
Defines how penalty assignment history survives deletion of the penalty catalog entry it came from: deleting a catalog entry never deletes or alters existing assignments, each assignment keeps its snapshot label and amount and remains readable and correctly valued with a null catalog reference, and finance overview paid/open sums for those assignments are unaffected.
## Requirements
### Requirement: Deleting a penalty preserves its assignments
Deleting a penalty catalog entry MUST NOT delete or alter existing penalty assignments. Assignments MUST remain readable and correctly valued from their snapshot after the catalog entry is removed.

#### Scenario: Delete a penalty with paid assignments
- **WHEN** a penalty with paid and unpaid assignments is deleted
- **THEN** all existing assignments remain
- **AND** each still shows its snapshot amount and label
- **AND** the finance overview's paid/open sums are unchanged for those assignments

#### Scenario: Assignment survives catalog removal
- **WHEN** an assignment's penalty catalog entry has been deleted
- **THEN** the assignment lists with its snapshot label and amount and a null catalog reference

