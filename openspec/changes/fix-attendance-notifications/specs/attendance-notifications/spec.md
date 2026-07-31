## ADDED Requirements

### Requirement: An attendance response generates a team notification
The system MUST create an `attendance` notification when a member's
attendance status is set to an actual response (`yes`, `no`, or `maybe`),
attributed to the member the response belongs to, and MUST NOT create one
when the status is set to `pending`.

#### Scenario: Member responds "yes" to an event
- **WHEN** a member sets their own attendance status to `yes` for an event
- **THEN** an `attendance` notification is created for that team, with the
  responding member as the actor, the event's id/title/date, and status
  `yes`

#### Scenario: Organizer sets another member's attendance
- **WHEN** a caller with `events:write` sets a different member's attendance
  status to `no`
- **THEN** the resulting notification's actor is the member whose
  attendance changed, not the caller

#### Scenario: Attendance reset to pending
- **WHEN** an attendance status is set to `pending`
- **THEN** no notification is created

### Requirement: Attendance notifications are visible to the same audience as other event notifications
An `attendance` notification MUST be gated by the `events` module the same
way `event_created`/`event_cancelled`/etc. already are — no separate
visibility rule.

#### Scenario: Viewer has events:read
- **WHEN** a team member with at least `read` on the `events` module views
  the team's notification feed
- **THEN** `attendance` notifications appear alongside other event
  notifications

#### Scenario: Viewer has events:none
- **WHEN** a team member with `none` on the `events` module views the
  team's notification feed
- **THEN** `attendance` notifications are omitted, exactly like other
  `events`-module notification types
