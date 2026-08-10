## ADDED Requirements

### Requirement: Duplicate an event into a new standalone event
A caller with write permission on the `events` module MUST be able to
duplicate an existing event into a new, independent, non-recurring event
pre-filled from the source event's fields, without copying the source
event's attendance responses or comments.

#### Scenario: Duplicate a standalone event
- **WHEN** an organizer duplicates an existing event
- **THEN** a create form opens pre-filled with the source event's title,
  type, location, note, times, and other descriptive fields
- **AND** the date defaults to today (or the day chosen when initiating the
  duplicate) rather than the source event's own date
- **AND** submitting creates a new event with no attendance responses or
  comments

#### Scenario: Duplicate a series occurrence
- **WHEN** an organizer duplicates an occurrence that belongs to a
  recurring series
- **THEN** the resulting new event is a standalone, non-recurring event with
  no series association, regardless of the source occurrence's series

#### Scenario: Duplicate a multi-day event
- **WHEN** an organizer duplicates an event that spans multiple days
- **THEN** the pre-filled form's span length (the gap between `date` and
  `multiDayEndDate`) matches the source event's span, anchored to the new
  default date

#### Scenario: Read-only member cannot duplicate
- **WHEN** a caller without write permission on `events` attempts to
  duplicate an event
- **THEN** the action is not available/rejected, consistent with other
  event-authoring actions
