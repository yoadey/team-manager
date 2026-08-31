# notification-clarity Specification

## Purpose
Defines how a notification's secondary line handles missing data: any absent segment (title, date, note, actor) is omitted entirely rather than rendered, the literal string "undefined" must never appear in the output, and when all optional fields are present they are shown together in a stable, consistently separated order.
## Requirements
### Requirement: Event notifications never render a missing value as text
A notification's secondary line MUST omit any data segment (title, date, note, actor) that is absent, and MUST NEVER render the literal string "undefined" in its place.

#### Scenario: Event-created notification with only an event title and date
- **WHEN** an `event_created` notification carries an event title and event date but no separately-set notification title
- **THEN** the notification's secondary line shows the event title and date
- **AND** the text "undefined" does not appear anywhere in the rendered notification

#### Scenario: Event notification with all optional fields present
- **WHEN** an event notification carries a title, date, note, and actor name
- **THEN** all four are shown, separated consistently, in a stable order

