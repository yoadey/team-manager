# events-scheduling Specification

## Purpose
Defines correctness rules for how events are scheduled and classified: today counts as upcoming rather than past, cancelling the remainder of a recurring series leaves past instances untouched, attendance can no longer be changed on a cancelled event, and a non-recurring event may span multiple days via an optional `multiDayEndDate` (bounded, never before `date`, clearable) with upcoming/past classification everywhere in the app keyed off the event's last occurring day rather than its start day.

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

### Requirement: Series-wide deletion affects only future instances
Deleting the remainder of a recurring series MUST NOT remove instances
dated before today, or their recorded attendance/comments. The
specifically addressed event is still deleted regardless of its own
date.

#### Scenario: Delete remainder of a series with past attendance
- **WHEN** a series with past and future instances is deleted with
  `scope=series`
- **THEN** instances dated today or later, and their attendance and
  comments, are deleted
- **AND** instances dated before today keep their existing rows,
  attendance, and comments
- **AND** the specific event the deletion was invoked on is deleted
  regardless of its own date

#### Scenario: Series definition outlives its surviving past instances
- **WHEN** a series delete leaves at least one past instance in place
- **THEN** the `event_series` definition row is kept, so the surviving
  instances stay linked to their series rather than being detached
  (`events.series_id` is `ON DELETE SET NULL`)
- **AND** once a series-scoped delete leaves no instance referencing the
  series, the `event_series` row is removed (a `scope=single` delete never
  performs this cleanup, so a series whose last instance is removed
  individually keeps an unreferenced definition row)

### Requirement: Series-wide update affects only future instances
Bulk-updating a recurring series MUST NOT change instances dated before
today. The specifically addressed event is still updated regardless of
its own date.

#### Scenario: Update remainder of a series with past occurrences
- **WHEN** a series with past and future instances is updated with
  `scope=series`
- **THEN** instances dated today or later are updated
- **AND** instances dated before today keep their previous values
- **AND** the specific event the update was invoked on is updated
  regardless of its own date

### Requirement: Statistics relevance is exempt from the series date guard
A series-wide change to an occurrence's *statistics relevance*
(`excludeFromStats`) MUST apply to every occurrence in the series,
including those dated before today. Unlike the fields the guard protects,
this flag does not describe what took place — it decides whether the
occurrence counts — and the occurrences already held are the only ones
statistics have counted so far.

#### Scenario: Excluding a mis-counted recurring event from statistics
- **WHEN** a series with past and future instances is updated with
  `scope=series` and `excludeFromStats` set
- **THEN** every instance of the series, past ones included, is excluded
  from statistics
- **AND** fields that describe the occurrence (title, times, nominations)
  still leave instances dated before today unchanged

### Requirement: The demo backend applies the same series guards
The MSW demo backend MUST apply the same "today onwards" scoping to
series-wide delete, cancel/reactivate and edit as the real backend, so
demo mode cannot destroy or rewrite already-held occurrences either.

#### Scenario: Series delete in demo mode
- **WHEN** a series with past and future occurrences is deleted with
  `scope=series` against the demo backend
- **THEN** only occurrences dated today or later are removed
- **AND** past occurrences keep their rows and their attendance

#### Scenario: Series cancel in demo mode
- **WHEN** a series is cancelled with `scope=series` against the demo
  backend
- **THEN** only occurrences dated today or later change status
- **AND** past occurrences keep their existing status
