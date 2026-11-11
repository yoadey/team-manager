## ADDED Requirements

### Requirement: Calendar fills the available viewport
The month calendar view MUST use the full width of the content area and,
when the viewport is taller than the calendar's minimum height, MUST stretch
its week rows so the calendar fills the remaining height. When the viewport
is too small, the calendar MUST keep its minimum row height and scroll
rather than shrink.

#### Scenario: Wide desktop viewport
- **WHEN** the calendar is shown in a content area wider than 820 px
- **THEN** the calendar grid spans the full content width

#### Scenario: Tall viewport
- **WHEN** the content area is taller than the calendar's minimum height
- **THEN** the six week rows grow equally so the calendar reaches the bottom
  of the content area without leaving empty space below it

#### Scenario: Small viewport
- **WHEN** the content area is shorter than the calendar's minimum height
- **THEN** each week row keeps its minimum height and the page scrolls

### Requirement: Day cells show as many entries as fit
A calendar day cell MUST show at least the previously fixed number of
entries (3 events and 2 absences on desktop, 2 events and 1 absence on
mobile) and MUST show more when its measured height has room for them;
entries that still do not fit MUST be summarised by a "+n" marker.

#### Scenario: Stretched cell shows more entries
- **WHEN** a day has five events and its cell is tall enough for five event
  chips
- **THEN** all five events are shown and no "+n" marker appears

#### Scenario: Minimum-height cell keeps the previous limit
- **WHEN** a day has five events and its cell is at the minimum height on
  desktop
- **THEN** three events are shown followed by a "+2" marker

#### Scenario: Measurement unavailable
- **WHEN** the browser cannot measure the grid (no `ResizeObserver`)
- **THEN** the previous fixed limits apply
