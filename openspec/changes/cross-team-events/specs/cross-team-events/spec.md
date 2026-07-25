## ADDED Requirements

### Requirement: Event may target multiple teams
An event MUST be able to target more than one team, and members of any targeted team MUST see the event and its attendance.

#### Scenario: Member of a targeted team
- **WHEN** an event targets teams A and B and a member of team B opens it
- **THEN** they see the event and its cross-team attendance

### Requirement: Create restricted to write-in-all-targets
Creating (or changing the targets of) a cross-team event MUST require write permission on events in every targeted team.

#### Scenario: Missing permission in one target
- **WHEN** a user has events:write in team A but not team B and tries to create an event targeting A and B
- **THEN** the request is rejected

#### Scenario: Permission in all targets
- **WHEN** a user has events:write in both team A and team B
- **THEN** they may create an event targeting A and B

### Requirement: Merged attendance without profile access
The event view MUST show attendees across all targeted teams with a team label per person, without exposing foreign members' profiles or profile-level PII.

#### Scenario: Foreign attendee
- **WHEN** a viewer sees an attendee who is not in the viewer's own team
- **THEN** that attendee is shown with name, avatar, and team badge only, with no way to open their profile and no email/phone exposed

### Requirement: Multi-team member counts once
A person who belongs to several of the event's targeted teams MUST appear once, RSVP once, and be labelled with their memberships limited to the event's targeted teams.

#### Scenario: Member of two targeted teams
- **WHEN** a person is in both targeted teams A and B (and also team C, not targeted)
- **THEN** they appear once with a single RSVP, labelled with teams A and B, and not team C
