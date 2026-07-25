## ADDED Requirements

### Requirement: Statistics use effective attendance
Attendance statistics MUST be computed from the effective attendance status (explicit response, else an overlapping absence → not attending, else an opt-out default → attending), identical to the status shown on the event summary.

#### Scenario: Opt-out event with no explicit responses
- **WHEN** an opt-out event has members who never responded
- **THEN** those members count as attending in the statistics, matching the event summary count

### Requirement: Reconcilable aggregations
Event-level and member-level attendance aggregations shown together MUST treat former members consistently, so their totals reconcile.

#### Scenario: After a member leaves
- **WHEN** a member leaves the team and statistics are viewed
- **THEN** the event-level and member-level counts apply the same membership filter
- **AND** the two figures do not contradict each other
