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
