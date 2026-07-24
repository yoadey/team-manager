## ADDED Requirements

### Requirement: Event notifications never render a missing value as text
A notification's secondary line MUST omit any data segment (title, date, note, actor) that is absent, and MUST NEVER render the literal string "undefined" in its place.

#### Scenario: Event-created notification with only an event title and date
- **WHEN** an `event_created` notification carries an event title and event date but no separately-set notification title
- **THEN** the notification's secondary line shows the event title and date
- **AND** the text "undefined" does not appear anywhere in the rendered notification

#### Scenario: Event notification with all optional fields present
- **WHEN** an event notification carries a title, date, note, and actor name
- **THEN** all four are shown, separated consistently, in a stable order
