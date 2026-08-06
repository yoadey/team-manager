## Purpose

A standalone, re-runnable command-line tool that imports a club's members, events,
attendance, and planned absences out of SpielerPlus into Teamverwaltung ahead of real
user logins, so migrating clubs don't lose their history.

## ADDED Requirements

### Requirement: Pre-created accounts usable via forgot-password
Each imported SpielerPlus member with an email not already present in `users` MUST get
a new `users` row that is immediately usable through the existing forgot-password flow,
without requiring self-registration or email verification.

#### Scenario: New member becomes login-ready
- **WHEN** the tool imports a SpielerPlus member whose email has no existing
  Teamverwaltung account
- **THEN** a `users` row is created with `email_verified_at` already set and a
  non-empty `password_hash`
- **AND** that person can complete `POST /auth/forgot-password` for that email and log
  in afterwards

#### Scenario: Existing account is left alone
- **WHEN** the tool imports a SpielerPlus member whose email already has a
  Teamverwaltung `users` row
- **THEN** no new user is created and the existing row is not modified

### Requirement: Re-running the import does not duplicate data
Running the tool again against the same SpielerPlus data and the same Teamverwaltung
database MUST NOT create duplicate users, memberships, events, attendance, or
absences.

#### Scenario: Second run after a successful first run
- **WHEN** the tool is run twice in a row against unchanged SpielerPlus data
- **THEN** the second run creates zero new `users`, `memberships`, `events`, or
  `absences` rows for records already imported by the first run
- **AND** `attendance` rows are updated in place rather than duplicated

### Requirement: Dry-run makes no database writes
The tool MUST support a mode that performs the full read/scrape and mapping pipeline
and reports what it would write, without writing to the database.

#### Scenario: Dry-run reports without persisting
- **WHEN** the tool is run with `--dry-run`
- **THEN** it prints a summary of records it would create/update/skip
- **AND** no row in the target database is created, updated, or deleted as a result

### Requirement: Invalid absences are skipped, not fatal
An imported absence that would violate Teamverwaltung's absence constraints (end date
before start date, span over 1095 days, or overlapping an existing absence for that
member) MUST be skipped and reported, without aborting the rest of the run.

#### Scenario: Overlapping historical absence
- **WHEN** an imported absence's date range overlaps an absence already recorded for
  that member
- **THEN** the tool skips creating that absence, logs it in the run summary, and
  continues importing the remaining records

### Requirement: Unmapped SpielerPlus roles fail loudly
A SpielerPlus role with no corresponding entry in the configured role-mapping table
MUST cause the run to fail with an error naming the unmapped role, rather than
importing the affected member with a default or guessed role.

#### Scenario: Role mapping is missing an entry
- **WHEN** a SpielerPlus member has a role that is not present in the role-mapping
  configuration
- **THEN** the tool exits with a non-zero status and an error naming that role
- **AND** no partial import for that run is left half-committed to the database

### Requirement: Requests to SpielerPlus are throttled
The tool MUST enforce a minimum, configurable gap between requests it makes to
SpielerPlus, so a full run's many per-member and per-event requests cannot fire back
to back.

#### Scenario: Default throttling applies
- **WHEN** the tool runs without `SPIELERPLUS_REQUEST_DELAY` set
- **THEN** consecutive requests to SpielerPlus are spaced at least 500ms apart

#### Scenario: Throttling is configurable
- **WHEN** `SPIELERPLUS_REQUEST_DELAY` is set to a valid Go duration
- **THEN** consecutive requests are spaced at least that duration apart

### Requirement: The active SpielerPlus team is confirmed before writing anything
Since SpielerPlus scopes every page the tool reads by a session-level "active team"
with no team id in the URL to cross-check, the tool MUST determine that team's name
and have it confirmed - interactively or via configuration - before any database
write occurs.

#### Scenario: Interactive confirmation accepted
- **WHEN** the tool runs without `SPIELERPLUS_EXPECTED_TEAM_NAME` set and the operator
  answers "y" or "yes" to the printed confirmation prompt
- **THEN** the run proceeds

#### Scenario: Interactive confirmation declined
- **WHEN** the tool runs without `SPIELERPLUS_EXPECTED_TEAM_NAME` set and the operator
  answers anything other than "y"/"yes" (including no input)
- **THEN** the run aborts before any database write, with no partial import committed

#### Scenario: Non-interactive confirmation via configuration
- **WHEN** `SPIELERPLUS_EXPECTED_TEAM_NAME` is set and matches the detected active
  team's name
- **THEN** the run proceeds without prompting

#### Scenario: Non-interactive mismatch aborts
- **WHEN** `SPIELERPLUS_EXPECTED_TEAM_NAME` is set and does not match the detected
  active team's name
- **THEN** the run aborts before any database write, with an error naming both the
  expected and detected team
