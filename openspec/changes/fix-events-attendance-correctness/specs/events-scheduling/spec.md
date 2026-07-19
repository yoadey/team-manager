## ADDED Requirements

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
