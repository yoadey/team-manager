## ADDED Requirements

### Requirement: Inline RSVP from the events overview
A member MUST be able to set their own attendance for an event directly from the events overview using compact icon controls, without opening the event detail.

#### Scenario: Accept from the list
- **WHEN** a member taps the accept control on an event row
- **THEN** their attendance is set to accepted and the row reflects the new status

#### Scenario: Current status shown
- **WHEN** the overview renders an event the member has already responded to
- **THEN** the member's current status is highlighted in the inline controls

#### Scenario: Past the cutoff
- **WHEN** an event's cancellation cutoff has passed
- **THEN** the inline controls are disabled and the member cannot change their attendance from the overview
