## Context

Events are single-team: `events.team_id` plus a strict `AND team_id = $N` invariant on every by-id query, and RBAC keys authorization on the path's `{teamId}`. Attendance rows are per-team. This feature deliberately introduces a **controlled** exception: an event may belong to a set of teams, and authorization/visibility must consider membership in *any* targeted team while still not leaking profiles across team boundaries.

## Goals / Non-Goals

**Goals:**
- One event → many teams; each targeted team's members see it.
- Merged attendance view with a per-person team badge, restricted to name/avatar/team/status (no profile access) for people outside the viewer's own team.
- A multi-team member is a single attendee with a single RSVP, labelled per the display rule below.
- Creation gated on write-on-events in **all** targeted teams.

**Display rule (per viewer, per attendee):**
- If the attendee is a member of the viewer's own (currently active) team, show **no team badge** — they read as an ordinary same-team attendee.
- Otherwise, intersect the event's targeted teams with the attendee's own memberships, and show the **alphabetically-first team name** (case-insensitive) from that intersection as the badge. This is a single label, not a list — it only needs to convey "this person is from elsewhere," not enumerate every team they happen to share with the event.

**Non-Goals:**
- Cross-team access to anything else (finances, news, polls, member profiles).
- Cross-*club* events; targets are teams within the same instance.

## Decisions

- **Model:** keep `events.team_id` as the "owning" team for backward compatibility, add an `event_teams(event_id, team_id)` join for all targeted teams (owning team included). Reads resolve visibility via "viewer is a member of any row in `event_teams`".
- **Authorization:** extend `authz` so an event route authorizes when the caller has the required permission in *any* targeted team for read/RSVP, and — for create/update of a cross-team event — in *all* targeted teams for `events:write`. Single-team events keep today's exact behavior (the invariant is unchanged for them).
- **Attendance dedup:** the `attendance` table is already `UNIQUE(event_id, user_id)` with no `team_id` column — a multi-team member already gets exactly one row and one RSVP with zero migration needed here. What's new is the *display* layer: for each attendee, compute their memberships intersected with the event's targeted teams, for the display rule above to pick from.
- **Restricted projection:** a `CrossTeamAttendee` view exposes only `{name, avatarColor, hasPhoto, status, teamName?}` (badge per the display rule; absent for viewer's-own-team attendees) and no membership/profile identifiers that would allow navigating to a foreign profile. The frontend disables profile navigation for attendees not in the viewer's own team.
- **Creation UI:** multi-team picker lists only teams where the creator has `events:write`; server re-validates.

## Risks / Trade-offs

- **This weakens the strongest invariant in the codebase** (`AND team_id = $N`). Mitigate by scoping the exception narrowly to event read/RSVP/attendance, re-validating on the server, and leaving all other modules strictly single-team. Document the exception in `CLAUDE.md`.
- RBAC generation (`cmd/genrbac`, `matchRBACRoute`) assumes one `{teamId}`; multi-team events need an authorization path that doesn't fit the generated single-team table — design the check explicitly and keep single-team routes on the existing fast path.
- Privacy: the merged view must not expose emails, phone, or profile links across teams — only name/avatar/team badge. Verify with tests that no foreign PII leaks.
- Large surface area; consider landing behind a flag and after `consistent-profile-photos` (shared avatar) and `event-cancellation-lead-time`.
