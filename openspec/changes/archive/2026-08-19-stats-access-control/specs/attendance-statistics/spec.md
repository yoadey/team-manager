## MODIFIED Requirements

### Requirement: Matrix range and authorization
The matrix endpoint MUST apply the same date-range defaulting and clamping as the stats overview, and MUST require the same `stats`-module read authorization; an unauthenticated request MUST be rejected.

#### Scenario: Default range when unspecified
- **WHEN** the matrix is requested without `from`/`to`
- **THEN** the same default window as the overview (last 3 months) is used

#### Scenario: Unauthenticated request
- **WHEN** the matrix is requested without a valid session
- **THEN** the request is rejected with 401
