# event-experience Specification

## Purpose
TBD - created by archiving change kleinere-findings. Update Purpose after archive.
## Requirements
### Requirement: Event location autocomplete
The event create/edit form MUST suggest previously used locations for the current team, deduplicated case-insensitively, while still allowing free-text entry.

#### Scenario: Typing a location that matches a past event
- **WHEN** a user opens the location field in the event create/edit form
- **THEN** locations used by the team's existing events are offered as suggestions
- **AND** each distinct location (case-insensitively) appears at most once
- **AND** the user may still type any value not in the list

### Requirement: Recurring event series may end on a date instead of a count
Creating a recurring event series MUST support specifying an end date as an alternative to a fixed occurrence count.

#### Scenario: Creating a series with an end date
- **WHEN** a user creates a recurring series and specifies an end date instead of a repeat count
- **THEN** occurrences are generated weekly up to and including that end date
- **AND** no occurrence is generated after the end date

### Requirement: Member birthdays appear in the calendar
The event calendar MUST show a yearly recurring entry for each visible member's birthday, subject to the same visibility rule as the member's birthday on their profile.

#### Scenario: Viewing a calendar month containing a member's birthday
- **WHEN** a user with permission to see a member's birthday views a calendar month containing it
- **THEN** a birthday entry for that member appears on the correct day
- **AND** the entry recurs every year without a stored per-year event row

#### Scenario: Viewing a calendar without birthday visibility
- **WHEN** a user without permission to see a member's birthday views the same calendar month
- **THEN** no birthday entry for that member appears

### Requirement: Configurable RSVP deadline with a countdown
An event MAY have a response deadline. Once set, non-privileged members MUST be blocked from changing their attendance response after the deadline passes, while privileged roles remain unaffected. The event detail view MUST show a countdown once less than 24 hours remain before the deadline.

#### Scenario: Responding before the deadline
- **WHEN** a member sets their attendance response before the event's RSVP deadline
- **THEN** the response is accepted

#### Scenario: Responding after the deadline
- **WHEN** a member without a role permitting late responses attempts to respond after the RSVP deadline
- **THEN** the response is rejected

#### Scenario: Privileged role responding after the deadline
- **WHEN** a member with a role permitted to bypass the deadline (e.g. an administrator) responds after the RSVP deadline
- **THEN** the response is accepted

#### Scenario: Countdown display
- **WHEN** an event's RSVP deadline is less than 24 hours away
- **THEN** the event detail view shows a countdown to the deadline

