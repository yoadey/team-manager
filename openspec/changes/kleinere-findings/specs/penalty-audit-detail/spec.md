## ADDED Requirements

### Requirement: Penalty assignments record when it was earned and why
Assigning a penalty to a member MUST allow specifying the date the penalty was earned (independent of the date it is recorded) and an optional free-text note.

#### Scenario: Assigning a penalty for a past date
- **WHEN** a team member with finance-write permission assigns a penalty and specifies a date in the past
- **THEN** the penalty assignment is stored with that date, not the current date

#### Scenario: Assigning a penalty with a note
- **WHEN** a penalty assignment includes a note
- **THEN** the note is stored and shown alongside the assignment

#### Scenario: Assigning a penalty without a note
- **WHEN** a penalty assignment omits the note
- **THEN** the assignment is created successfully with no note recorded
