## MODIFIED Requirements

### Requirement: Matrix range and authorization
The overview, matrix, absence-stats, and single-member statistics
endpoints MUST all apply the same date-range defaulting and clamping, and
MUST all require the same `events`-module read authorization; an
unauthenticated request MUST be rejected.

#### Scenario: Default range when unspecified
- **WHEN** any of the four statistics endpoints is requested without
  `from`/`to`
- **THEN** the same default window (last 3 months) is used

#### Scenario: Single-member view honors an explicit range
- **WHEN** the single-member statistics endpoint is requested with
  explicit `from`/`to`
- **THEN** the returned statistics are computed for that range, not the
  default

#### Scenario: Unauthenticated request
- **WHEN** any of the four statistics endpoints is requested without a
  valid session
- **THEN** the request is rejected with 401
