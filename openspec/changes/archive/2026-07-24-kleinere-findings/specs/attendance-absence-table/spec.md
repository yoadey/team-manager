## ADDED Requirements

### Requirement: A per-member, per-event absence table is available alongside the quota view
The statistics page MUST offer a table view, separate from the existing per-person attendance-quota view, listing every member absence with the event and date it occurred.

#### Scenario: Viewing the absence table
- **WHEN** a user with stats-read permission opens the absence table tab
- **THEN** each row shows a member, the event they missed, and the event's date
- **AND** the table covers the same date range as the quota view's active filter

#### Scenario: No absences in the selected range
- **WHEN** no member has any absence in the selected date range
- **THEN** the absence table shows an empty state instead of an empty table
