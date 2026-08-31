# attendance-statistics Specification

## Purpose
Defines how attendance statistics are computed and shaped: overview quotes, the per-member-per-event matrix, and the single-member view all derive from the same effective-attendance status (explicit response, else a covering absence counts as not attending, else an opt-out event defaults to attending) so no view contradicts another; event- and member-level aggregations apply a consistent membership filter across former members; the matrix orders rows by attendance frequency and columns chronologically; and all three endpoints share the same date-range defaulting/clamping and require `stats`-module read authorization.

## Requirements

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

### Requirement: Per-member-per-event attendance matrix
The statistics capability MUST expose an attendance matrix for a team over a date range: one row per current member, one column per active event in range, and a cell giving that member's effective attendance for that event. Cell status MUST be derived from the same effective-attendance definition as the per-member quotes (explicit response, else a covering absence → not attending, else an opt-out default → attending, else pending), so a cell never contradicts the overview.

#### Scenario: Cell reflects effective status
- **WHEN** an opt-out event in range has a member who never explicitly responded
- **THEN** that member's cell for that event is "yes" (attending), matching the overview quote and event summary

#### Scenario: No response on a normal event
- **WHEN** a normal (opt-in) event in range has a member with no explicit response, no covering absence
- **THEN** that member's cell for that event is "pending" (unbekannt)

#### Scenario: Row aggregate reconciles with the matrix
- **WHEN** the matrix is returned
- **THEN** each row's reported `yes` count equals the number of that member's "yes" cells across the columns

### Requirement: Matrix ordering
The matrix MUST order rows by attendance frequency (most attending first) and columns chronologically, so the grid reads consistently with the overview.

#### Scenario: Rows sorted by attendance
- **WHEN** two members have different "yes" counts in range
- **THEN** the member with more "yes" appears in an earlier row

#### Scenario: Columns sorted by date
- **WHEN** the matrix has multiple events
- **THEN** the columns are ordered by event date ascending

### Requirement: Matrix range and authorization
The overview, matrix, and single-member statistics endpoints MUST all
apply the same date-range defaulting and clamping, and MUST all require
the same `stats`-module read authorization; an unauthenticated request
MUST be rejected.

#### Scenario: Default range when unspecified
- **WHEN** any of the three statistics endpoints is requested without
  `from`/`to`
- **THEN** the same default window (last 3 months) is used

#### Scenario: Single-member view honors an explicit range
- **WHEN** the single-member statistics endpoint is requested with
  explicit `from`/`to`
- **THEN** the returned statistics are computed for that range, not the
  default

#### Scenario: Unauthenticated request
- **WHEN** any of the three statistics endpoints is requested without a
  valid session
- **THEN** the request is rejected with 401
