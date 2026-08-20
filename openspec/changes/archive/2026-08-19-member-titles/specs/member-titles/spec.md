## ADDED Requirements

### Requirement: Self-service title on a membership
A member MUST be able to set or clear a short, free-text, non-RBAC "title"
(e.g. "Orgaente", "Witzbeauftragter") on their own membership without
holding `members:write`, as long as they have at least `members: read` in
that team (a team that sets `members: none` for a member hides this the
same as any other members-module read).

#### Scenario: Setting your own title
- **WHEN** a member with only the default `members: read` permission calls
  `PUT /teams/{teamId}/members/{myMembershipId}/title` with `{"title":
  "Witzbeauftragter"}`
- **THEN** the request succeeds
- **AND** the membership's title is now "Witzbeauftragter"

#### Scenario: Clearing your own title
- **WHEN** a member submits `{"title": ""}` to their own title endpoint
- **THEN** the title is cleared (stored as no title)

#### Scenario: Title too long
- **WHEN** a member submits a title longer than 40 characters
- **THEN** the request is rejected and the stored title is unchanged

### Requirement: A member cannot set another member's title via the self-service endpoint
The self-service title endpoint MUST reject a request targeting a
membership other than the caller's own, regardless of the caller's
permission level.

#### Scenario: Attempting to set someone else's title
- **WHEN** a member (with or without `members:write`) calls
  `PUT /teams/{teamId}/members/{otherMembershipId}/title` for a membership
  that is not their own
- **THEN** the request is rejected
- **AND** the other member's title is unchanged

### Requirement: Admins can manage any member's title
A caller with `members:write` MUST be able to set or clear any member's
title through the existing member-update path, the same way they manage
that member's `group`.

#### Scenario: Moderating a member's title
- **WHEN** a caller with `members:write` submits `PATCH
  /teams/{teamId}/members/{membershipId}` with a `title` field for another
  member
- **THEN** that member's title is updated to the submitted value

### Requirement: Title is cosmetic and carries no permissions
A title MUST NOT be interpreted by RBAC, notifications, or any other
system behavior — it is display-only text.

#### Scenario: Title does not affect permissions
- **WHEN** a member sets any title text, including text matching a role or
  permission name
- **THEN** their effective permissions are unaffected

### Requirement: Title displayed in small, secondary text alongside the member's name
Where a member's name is shown with enough room — the member list and an
event's attendance list — their title, if set, MUST be shown nearby in
visually smaller, secondary text, not equal emphasis to the name.

#### Scenario: Member list shows the title
- **WHEN** a member with a title set is shown in the team's member list
- **THEN** their title is displayed in small text near their name
- **AND** a member with no title shows no title text

#### Scenario: Attendance list shows the title
- **WHEN** an event's attendance list is shown and a listed member has a
  title set
- **THEN** their title is displayed in small text in that row
