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
The event view MUST show attendees across all targeted teams with at most one team badge per person, without exposing foreign members' profiles or profile-level PII.

#### Scenario: Foreign attendee
- **WHEN** a viewer sees an attendee who is not in the viewer's own team
- **THEN** that attendee is shown with name, avatar, attendance status, and a team badge only, with no way to open their profile and no email/phone exposed

### Requirement: Team badge follows the viewer's own team, then alphabetical order
An attendee who belongs to the viewer's own (currently active) team MUST be shown with no team badge. An attendee who does not MUST be shown with a single badge naming the alphabetically-first (by team name) team, among the event's targeted teams, that the attendee belongs to.

#### Scenario: Attendee shares the viewer's team
- **WHEN** a viewer's own team is among an attendee's memberships in the event's targeted teams
- **THEN** that attendee is shown with no team badge

#### Scenario: Attendee is only in other targeted teams
- **WHEN** an attendee belongs to targeted teams "Bravo" and "Alpha" but not the viewer's own team
- **THEN** the attendee is shown with the badge "Alpha" (alphabetically first), not "Bravo"

### Requirement: Multi-team member counts once
A person who belongs to several of the event's targeted teams MUST appear once and RSVP once across all of them.

#### Scenario: Member of two targeted teams
- **WHEN** a person is in both targeted teams A and B (and also team C, not targeted)
- **THEN** they appear once with a single RSVP, and their badge is computed only from A and B (never C)
