# shared-calendar Specification

## Purpose
TBD - created by archiving change shared-team-calendar. Update Purpose after archive.
## Requirements
### Requirement: Grant read-only calendar visibility to another team
A team MUST be able to grant, and revoke, read-only visibility of its calendar to another team; only a member with settings write on the sharing team may manage grants.

#### Scenario: Grant created
- **WHEN** a settings-authorized member of team A grants calendar visibility to team B
- **THEN** members of team B can read team A's shared calendar

#### Scenario: Grant revoked
- **WHEN** the grant from team A to team B is revoked
- **THEN** team B members can no longer read team A's calendar

### Requirement: Shared calendar exposes only schedule and location
A shared calendar MUST expose only each event's time, location, title, and type — never attendance, participants, comments, or notes.

#### Scenario: Viewing a shared event
- **WHEN** a grantee team member views a shared calendar event
- **THEN** they see its time, location, title, and type
- **AND** they do not see attendance, participants, comments, or notes

#### Scenario: No grant
- **WHEN** a team member's team has no grant from the owning team
- **THEN** they cannot read that team's calendar

