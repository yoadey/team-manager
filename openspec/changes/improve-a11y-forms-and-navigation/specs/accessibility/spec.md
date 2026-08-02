## ADDED Requirements

### Requirement: Tabbed navigation follows the WAI-ARIA APG tabs pattern
A tabbed interface MUST implement roving `tabindex` (only the active tab
is a Tab stop), arrow-key switching between tabs, and
`role="tabpanel"`/`aria-controls` wiring content to its triggering tab.

#### Scenario: Keyboard user switches tabs
- **WHEN** a user focuses a tablist and presses an arrow key
- **THEN** focus and selection move to the adjacent tab
- **AND** only the selected tab is reachable via a single further Tab
  press

### Requirement: Form inputs have a persistent accessible label
A form input MUST have an accessible label that persists after the user
types a value, not only placeholder text.

#### Scenario: Poll option input after typing
- **WHEN** a user types into a poll option input
- **THEN** the input's accessible name (via `aria-label` or an
  associated `<label>`) is still available to assistive technology

### Requirement: The calendar's day grid is keyboard-navigable
A calendar month view MUST let a keyboard user move focus between day
cells using arrow keys, not only via mouse/tap or the month prev/next
controls.

#### Scenario: Keyboard user browses days
- **WHEN** a user focuses a day cell in the calendar grid and presses an
  arrow key
- **THEN** focus moves to the adjacent day in that direction
