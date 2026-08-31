# event-cancellation Specification

## Purpose
Defines an event's RSVP/cancellation cutoff as a labeled hours-and-minutes lead time before its start — not an absolute timestamp — with each recurring series occurrence deriving its own cutoff from its own start, and enforces it by rejecting a member's own attendance change after the cutoff while still allowing a caller with write permission on events to override it.
## Requirements
### Requirement: Cancellation deadline as a lead time before start
An event's RSVP/cancellation cutoff MUST be configurable as a duration (hours and minutes) before the event's start, not as an absolute timestamp. This is the only cutoff mechanism for events (no absolute-timestamp alternative). The hours and minutes inputs MUST each be clearly labeled with their unit so the pair reads as a duration, not two unlabeled numbers.

#### Scenario: Organizer sets a lead time
- **WHEN** an organizer sets the cutoff to 24 hours before start
- **THEN** the event's effective cutoff for each occurrence is that occurrence's start minus 24 hours

#### Scenario: Series occurrence
- **WHEN** a recurring series has a cancellation lead time
- **THEN** every occurrence derives its own absolute cutoff from its own start

#### Scenario: Labeled duration input
- **WHEN** an organizer opens the event form's cancellation lead time field
- **THEN** the hours input and minutes input are each visibly labeled with their unit

### Requirement: Enforce the cutoff for self-service changes
Once an event's cutoff has passed, a member's own attendance change MUST be rejected, while a caller with write permission on events MUST still be able to change it.

#### Scenario: Member declines after the cutoff
- **WHEN** a member tries to change their own attendance after the effective cutoff
- **THEN** the request is rejected with a problem+json error and the stored attendance is unchanged

#### Scenario: Organizer overrides after the cutoff
- **WHEN** a caller with write permission on events changes attendance after the cutoff
- **THEN** the change is accepted

#### Scenario: No cutoff set
- **WHEN** an event has no cancellation lead time
- **THEN** attendance changes are accepted at any time before the event

