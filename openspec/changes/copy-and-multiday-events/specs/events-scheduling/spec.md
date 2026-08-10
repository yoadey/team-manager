## ADDED Requirements

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

### Requirement: Ongoing multi-day events count as upcoming, not past
The upcoming/past listing scope MUST key off an event's last occurring day
(`multiDayEndDate` when set, otherwise `date`), not its start day alone.

#### Scenario: Multi-day event already started but not finished
- **WHEN** a multi-day event's `date` is before today but its
  `multiDayEndDate` is today or later
- **THEN** the event appears in the upcoming list
- **AND** it does not appear in the past list
