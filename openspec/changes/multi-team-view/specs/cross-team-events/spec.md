## MODIFIED Requirements

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
