## MODIFIED Requirements

### Requirement: Statistics use effective attendance
Attendance statistics MUST be computed from the effective attendance status (explicit response, else an overlapping absence → not attending, else an opt-out default → attending), identical to the status shown on the event summary. For a multi-day event (`end_date` set), "overlapping" MUST be evaluated against the event's full span (`date` through `end_date`, inclusive), not only its start date — a planned absence intersecting any part of the span counts as covering the event.

#### Scenario: Opt-out event with no explicit responses
- **WHEN** an opt-out event has members who never responded
- **THEN** those members count as attending in the statistics, matching the event summary count

#### Scenario: Absence covers only the later part of a multi-day event
- **WHEN** a multi-day event spans `date` through `end_date` and a member has a planned absence whose range overlaps the span but does not cover `date` itself (e.g. it starts after the event's first day)
- **THEN** that member's effective attendance for the event resolves to "no" (not attending), not pending or attending

### Requirement: Per-member-per-event attendance matrix
The statistics capability MUST expose an attendance matrix for a team over a date range: one row per current member, one column per active event in range, and a cell giving that member's effective attendance for that event. Cell status MUST be derived from the same effective-attendance definition as the per-member quotes (explicit response, else a covering absence → not attending, else an opt-out default → attending, else pending), so a cell never contradicts the overview. For a multi-day event, "covering absence" uses the same full-span overlap as the effective-attendance definition, not just the event's start date.

#### Scenario: Cell reflects effective status
- **WHEN** an opt-out event in range has a member who never explicitly responded
- **THEN** that member's cell for that event is "yes" (attending), matching the overview quote and event summary

#### Scenario: No response on a normal event
- **WHEN** a normal (opt-in) event in range has a member with no explicit response, no covering absence
- **THEN** that member's cell for that event is "pending" (unbekannt)

#### Scenario: Row aggregate reconciles with the matrix
- **WHEN** the matrix is returned
- **THEN** each row's reported `yes` count equals the number of that member's "yes" cells across the columns
