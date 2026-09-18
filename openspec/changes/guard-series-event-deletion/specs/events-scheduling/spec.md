## ADDED Requirements

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
- **AND** once no instance references the series any more, the
  `event_series` row is removed

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
