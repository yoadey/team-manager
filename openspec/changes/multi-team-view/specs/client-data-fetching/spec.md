## MODIFIED Requirements

### Requirement: Team-scoped cache prevents stale cross-team data
Query cache keys MUST be scoped by team id so that switching the active team,
or changing the set of selected teams, cannot surface data fetched for a team
that is no longer selected and cannot attribute one team's data to another
team.

#### Scenario: Rapid team switch
- **WHEN** the user switches from team A to team B while A's request is in flight
- **THEN** B's screen shows only B's data
- **AND** a late response for A does not overwrite B's cache

#### Scenario: Team removed from a multi-team selection
- **WHEN** teams A and B are selected and the user deselects B while B's
  request is in flight
- **THEN** the view shows only A's data, and the late response for B does
  not appear in it

#### Scenario: Shared cache between single- and multi-team views
- **WHEN** the user has viewed team A alone and then selects A and B
- **THEN** team A's cached data is reused and attributed to team A only
