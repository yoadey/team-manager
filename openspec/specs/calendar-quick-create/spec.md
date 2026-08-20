# calendar-quick-create Specification

## Purpose
TBD - created by archiving change calendar-double-click-create. Update Purpose after archive.

## Requirements

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

#### Scenario: Double-click an absence or birthday chip inside a day
- **WHEN** a member double-clicks an absence or birthday entry rendered inside a day cell
- **THEN** the day's create sheet does not open

### Requirement: Keyboard equivalent for calendar quick-create
A member with `events:write` permission MUST be able to trigger the same pre-dated event creation via keyboard, not only by double-click.

#### Scenario: Focus and activate a day cell via keyboard
- **WHEN** a member with `events:write` tabs to an in-month day cell and presses Enter or Space
- **THEN** the event-creation sheet opens with that day's date pre-filled, same as a double-click

#### Scenario: No keyboard target without permission
- **WHEN** a member without `events:write`, or an out-of-month cell, is considered
- **THEN** the cell is not focusable and carries no button role or label
