## MODIFIED Requirements

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
