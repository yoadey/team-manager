# poll-transparency Specification

## Purpose
Defines transparency safeguards for polls: the confirmation dialog shown before deleting a poll must name the specific poll (its question text) being deleted, and for a non-anonymous poll every team member can see which members voted for each option, while an anonymous poll never shows voter names for any option.
## Requirements
### Requirement: The delete-poll confirmation names the poll
The confirmation dialog shown before deleting a poll MUST name the specific poll being deleted.

#### Scenario: Confirming deletion of a poll
- **WHEN** a user with polls-write permission clicks delete on a poll
- **THEN** the confirmation dialog's message includes that poll's question text

### Requirement: Non-anonymous polls show who voted for each option
For a poll created without the anonymous flag, every team member viewing the poll MUST be able to see which members voted for each option.

#### Scenario: Viewing a non-anonymous poll with votes
- **WHEN** a user views a non-anonymous poll that has votes
- **THEN** each option with at least one vote shows the names of the members who voted for it

#### Scenario: Viewing an anonymous poll with votes
- **WHEN** a user views an anonymous poll that has votes
- **THEN** no voter names are shown for any option

