# poll-visibility Specification

## Purpose
TBD - created by archiving change poll-vote-visibility. Update Purpose after archive.
## Requirements
### Requirement: Per-option voter lists for non-anonymous polls
For a non-anonymous poll, the UI MUST be able to show, for each option, the list of members who selected it.

#### Scenario: Viewing who picked an option
- **WHEN** a user opens the voter details of a non-anonymous poll
- **THEN** each option shows the list of members (avatar + name) who selected it

### Requirement: User×option vote matrix
For a non-anonymous poll, the UI MUST offer a matrix view with the options numbered across the top and one row per user marking which options that user selected.

#### Scenario: Matrix view
- **WHEN** the user switches to the matrix view
- **THEN** the options appear as columns numbered 1..n and each user's row marks the options they chose

### Requirement: Anonymous polls reveal no identities
An anonymous poll MUST NOT expose who voted, in any view.

#### Scenario: Anonymous poll
- **WHEN** a poll is anonymous
- **THEN** no voter identities are returned or shown, only aggregate counts and percentages

