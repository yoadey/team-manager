## MODIFIED Requirements

### Requirement: Merged attendance without profile access
The event view MUST show attendees across all targeted teams with at most one team badge per person, without exposing foreign members' profiles or profile-level PII. A foreign member is one who is not in the viewer's own team, which is the team through which the event is being viewed: the single selected team in single-team mode, and the detail's current view team in multi-team mode.

#### Scenario: Foreign attendee
- **WHEN** a viewer sees an attendee who is not in the viewer's own team
- **THEN** that attendee is shown with name, avatar, attendance status, and a team badge only, with no way to open their profile and no email/phone exposed

#### Scenario: Foreign relative to the detail's view team
- **WHEN** teams A and B are selected, a cross-team event targets both, the
  detail is viewed as A, and an attendee belongs only to B
- **THEN** that attendee is shown as foreign (no profile, no email/phone),
  and after switching the view to B they are shown as a member of the
  viewer's own team

### Requirement: Team badge follows the viewer's own team, then alphabetical order
An attendee who belongs to the viewer's own team MUST be shown with no team
badge. An attendee who does not MUST be shown with a single badge naming the
alphabetically-first (by team name) team, among the event's targeted teams,
that the attendee belongs to. The viewer's own team is the team through which
the event is being viewed: the single selected team in single-team mode, and
the detail's current view team in multi-team mode.

#### Scenario: Attendee shares the viewer's team
- **WHEN** a viewer's own team is among an attendee's memberships in the event's targeted teams
- **THEN** that attendee is shown with no team badge

#### Scenario: Attendee is only in other targeted teams
- **WHEN** an attendee belongs to targeted teams "Bravo" and "Alpha" but not the viewer's own team
- **THEN** the attendee is shown with the badge "Alpha" (alphabetically first), not "Bravo"

#### Scenario: Multi-team mode uses the detail's view team
- **WHEN** teams "Alpha" and "Bravo" are selected, a cross-team event targets
  both, and the detail is viewed as "Bravo"
- **THEN** attendees who belong to "Bravo" are shown with no team badge
