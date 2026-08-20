# events-scheduling Specification

## Purpose
TBD - created by archiving change fix-events-attendance-correctness. Update Purpose after archive.

## Requirements

### Requirement: Today's events are upcoming
The upcoming-events listing MUST include events dated today. The past-events listing MUST exclude events dated today.

#### Scenario: Event scheduled for today
- **WHEN** a team has an event dated today and the upcoming scope is requested
- **THEN** the event appears in the upcoming list
- **AND** it does not appear in the past list

### Requirement: Series cancellation affects only future instances
Cancelling the remainder of a recurring series MUST NOT change the status of instances dated before today.

#### Scenario: Cancel remainder of a series
- **WHEN** a series with past and future instances is cancelled
- **THEN** instances dated today or later become cancelled
- **AND** instances dated before today keep their existing status

### Requirement: No attendance changes on cancelled events
A self-service attendance change MUST be rejected when the target event is cancelled.

#### Scenario: Attendance on a cancelled event
- **WHEN** a member attempts to set their attendance on a cancelled event
- **THEN** the request is rejected with a client error
- **AND** the stored attendance is unchanged

### Requirement: Non-recurring event may span multiple days
A non-recurring event MUST support an optional `multiDayEndDate` marking the
last day of a multi-day span; when set, the event is considered to occur on
every calendar day from `date` through `multiDayEndDate` inclusive.
`multiDayEndDate`, when set, MUST NOT be earlier than `date`. A recurring
event (`recurring: true`) MUST NOT set `multiDayEndDate`.

#### Scenario: Organizer creates a multi-day event
- **WHEN** an organizer creates an event with `date` 2026-08-14 and
  `multiDayEndDate` 2026-08-16
- **THEN** the event is created spanning 2026-08-14 through 2026-08-16
- **AND** the calendar shows the event on all three days

#### Scenario: multiDayEndDate before date is rejected
- **WHEN** a create or update request sets `multiDayEndDate` earlier than
  `date`
- **THEN** the request is rejected with a client error and no event is
  created or changed

#### Scenario: multiDayEndDate rejected on a recurring event
- **WHEN** a create request sets both `recurring: true` and
  `multiDayEndDate`
- **THEN** the request is rejected with a client error

#### Scenario: Single-day event omits multiDayEndDate
- **WHEN** an event has no `multiDayEndDate`
- **THEN** it is considered to occur only on `date`, matching existing
  single-day behavior

#### Scenario: Multi-day span exceeds the maximum
- **WHEN** a create or update request would leave `multiDayEndDate` more
  than 1095 days after `date`
- **THEN** the request is rejected with a client error

#### Scenario: Organizer clears a multi-day span back to single-day
- **WHEN** an update request sets `clearMultiDayEndDate: true` on an event
  that currently has `multiDayEndDate` set
- **THEN** the event's `multiDayEndDate` is cleared and it is henceforth
  considered a single-day event occurring only on `date`

### Requirement: Ongoing multi-day events count as upcoming, not past
The upcoming/past listing scope MUST key off an event's last occurring day
(`multiDayEndDate` when set, otherwise `date`), not its start day alone.

#### Scenario: Multi-day event already started but not finished
- **WHEN** a multi-day event's `date` is before today but its
  `multiDayEndDate` is today or later
- **THEN** the event appears in the upcoming list
- **AND** it does not appear in the past list
- **AND** it is treated as upcoming, not past, everywhere the client
  classifies events this way (event lists, cards, RSVP controls, and
  navigation/dashboard pending-response counts)
