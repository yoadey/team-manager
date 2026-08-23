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

### Requirement: Cashbox transactions are imported as Teamverwaltung transactions
The tool MUST import the team's SpielerPlus cashbox ledger into Teamverwaltung's
`transactions`, preserving title, date, amount, and income/expense direction.

#### Scenario: An expense transaction is imported
- **WHEN** the SpielerPlus cashbox shows a ledger entry with a negative amount
- **THEN** a `transactions` row is created with `type = 'expense'` and a positive
  `amount` equal to the entry's absolute value

#### Scenario: An income transaction is imported
- **WHEN** the SpielerPlus cashbox shows a ledger entry with a non-negative amount
- **THEN** a `transactions` row is created with `type = 'income'`

### Requirement: Membership dues are imported one row per due column
The tool MUST import SpielerPlus's per-member membership-dues matrix into
Teamverwaltung's `contributions`, one row per (member, due column), using the
column's label as the contribution's name. The tool MUST track imported dues in
its local idempotency state, since `contributions` has no natural unique key to
dedupe on.

#### Scenario: A member with multiple simultaneous dues
- **WHEN** a member has more than one SpielerPlus due column
- **THEN** each column is imported as its own `contributions` row for that member

#### Scenario: Re-running the import does not duplicate a due
- **WHEN** the tool is run twice against unchanged SpielerPlus dues data
- **THEN** the second run creates no additional `contributions` rows for dues
  already recorded in the idempotency state from the first run

### Requirement: Paid status for dues and penalties is not imported
Since Teamverwaltung derives a contribution's or penalty assignment's paid status
from income transactions linked to it, rather than storing it directly, the tool
MUST NOT attempt to link an imported transaction to the due/penalty it may pay.
Every imported due/penalty MUST be left unlinked, and the tool MUST report, in
the run summary, how many imported dues/penalties were marked paid on
SpielerPlus, so an operator knows how many to reconcile by hand.

#### Scenario: A due paid on SpielerPlus is imported
- **WHEN** a SpielerPlus due column is marked paid for a member
- **THEN** the imported `contributions` row has no linked transaction (shows as
  open in Teamverwaltung)
- **AND** the run summary's count of paid-on-SpielerPlus-but-unlinked dues
  includes it

#### Scenario: A penalty paid on SpielerPlus is imported
- **WHEN** an assigned punishment is marked paid on SpielerPlus
- **THEN** the imported `penalty_assignments` row has no linked transaction
  (shows as open in Teamverwaltung)
- **AND** the run summary's count of paid-on-SpielerPlus-but-unlinked penalties
  includes it

### Requirement: Penalties are imported with best-effort name matching
The tool MUST import SpielerPlus's penalty catalog into Teamverwaltung's
`penalties`, and assigned punishments into `penalty_assignments`. Since
SpielerPlus identifies an assigned punishment's member by display name only (no
id or profile link), an assignment MUST be matched to the imported roster by an
exact match on member name; an assignment whose name does not match any imported
member MUST be skipped and reported, not imported against a guessed member.

#### Scenario: An assignment matches an imported member by name
- **WHEN** an assigned punishment's member name exactly matches an imported
  member's name
- **THEN** a `penalty_assignments` row is created for that member

#### Scenario: An assignment's name does not match any imported member
- **WHEN** an assigned punishment's member name does not exactly match any
  imported member's name
- **THEN** the tool skips that assignment, logs it in the run summary, and
  continues importing the remaining records

### Requirement: Event location is imported from the event's own detail page
An event's location is not present on the events list page. The tool MUST fetch
each event's own detail page to import its location, and MUST NOT fail an
event's import solely because its location could not be fetched or parsed.

#### Scenario: An event has a location set
- **WHEN** an event's detail page shows an address
- **THEN** the imported `events` row's location is set to that address

#### Scenario: An event's location fetch fails
- **WHEN** fetching or parsing an event's detail page fails
- **THEN** the event is still imported, without a location, and the failure is
  logged

### Requirement: Every imported event is visible to its own team
Since Teamverwaltung determines an event's visibility via `event_teams` rather than
`events.team_id` alone, the tool MUST create a matching `event_teams` row for every
event it imports, and MUST NOT leave an `events` row committed without one.

#### Scenario: An event is imported
- **WHEN** the tool inserts a new `events` row for the target team
- **THEN** a corresponding `event_teams` row for that event and team is created in
  the same operation

#### Scenario: The event_teams write fails
- **WHEN** the `events` row was inserted but its `event_teams` row fails to insert
- **THEN** neither row is committed, and the event is retried on the next run
  rather than being recorded as already imported

### Requirement: A multi-day event's end date is imported
When an event's end time carries a date later than its start date, the tool MUST
import that as the event's end date, so a multi-day event renders as such in
Teamverwaltung rather than collapsing onto a single day.

#### Scenario: An event spans multiple days
- **WHEN** an event's end time is on a later calendar day than its start date
- **THEN** the imported `events` row's end date is set to that later day

#### Scenario: An event's end time has no distinct later date
- **WHEN** an event's end time carries no date, or one that is not later than its
  start date
- **THEN** the imported `events` row's end date is left unset

### Requirement: Member photos are imported for newly created users when configured
When object-store configuration is provided, the tool MUST upload each newly
created member's SpielerPlus profile photo to the same object store
Teamverwaltung's backend uses, under the same key convention, and point that
user's `photo_object_key` at it. Photo import MUST be entirely optional: without
object-store configuration, the tool MUST skip photo import for the whole run
without failing it. A member's generic placeholder ("no photo set") MUST NOT be
imported as if it were a real photo. Photo import MUST only apply to newly
created users - an existing user's photo MUST NOT be modified.

#### Scenario: A newly created member has a photo
- **WHEN** object-store configuration is present and a newly created member has
  a custom photo on SpielerPlus
- **THEN** the photo is uploaded to the object store and the user's
  `photo_object_key` is set to point at it

#### Scenario: Object-store configuration is absent
- **WHEN** the tool runs without object-store configuration
- **THEN** no photo is fetched or uploaded for any member, and the run completes
  normally

#### Scenario: A member has no custom photo
- **WHEN** a member's SpielerPlus profile shows the generic placeholder photo
- **THEN** no photo is uploaded for that member

#### Scenario: An existing account is not touched
- **WHEN** an imported member's email already has an existing Teamverwaltung
  `users` row
- **THEN** that user's existing photo (if any) is left unchanged, even if
  object-store configuration is present and the SpielerPlus profile has a photo

#### Scenario: A photo fetch, validation, or upload failure is not fatal
- **WHEN** fetching, validating, or uploading a member's photo fails
- **THEN** the tool skips that member's photo, logs it in the run summary, and
  continues importing the remaining records
