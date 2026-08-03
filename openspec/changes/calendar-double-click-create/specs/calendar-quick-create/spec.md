## ADDED Requirements

### Requirement: Double-click a calendar day to create a pre-dated event
A member with `events:write` permission MUST be able to open the event-creation sheet with a specific day pre-selected as the event date by double-clicking that day's cell in the calendar month view.

#### Scenario: Double-click an in-month day
- **WHEN** a member with `events:write` double-clicks an in-month day cell that has no events
- **THEN** the event-creation sheet opens
- **AND** the date field is pre-filled with that day's date

#### Scenario: Double-click without permission
- **WHEN** a member without `events:write` double-clicks a day cell
- **THEN** no sheet opens

#### Scenario: Double-click an event chip inside a day
- **WHEN** a member double-clicks an event chip rendered inside a day cell
- **THEN** the day's create sheet does not open
- **AND** the chip's own single-click behavior (opening that event's detail) is unaffected
