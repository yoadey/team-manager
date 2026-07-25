## Context

Events are single-team: `events.team_id` plus a strict `AND team_id = $N` invariant on every by-id query, and RBAC keys authorization on the path's `{teamId}`. Attendance rows are per-team. This feature deliberately introduces a **controlled** exception: an event may belong to a set of teams, and authorization/visibility must consider membership in *any* targeted team while still not leaking profiles across team boundaries.

## Goals / Non-Goals

**Goals:**
- One event → many teams; each targeted team's members see it.
- Merged attendance view with per-person team labels, restricted to name/avatar/team (no profile access) for people outside the viewer's own team.
- A multi-team member is a single attendee with a single RSVP, labelled with the subset of their memberships that are targeted by the event.
- Creation gated on write-on-events in **all** targeted teams.

**Non-Goals:**
- Cross-team access to anything else (finances, news, polls, member profiles).
- Cross-*club* events; targets are teams within the same instance.

## Decisions

- **Model:** keep `events.team_id` as the "owning" team for backward compatibility, add an `event_teams(event_id, team_id)` join for all targeted teams (owning team included). Reads resolve visibility via "viewer is a member of any row in `event_teams`".
- **Authorization:** extend `authz` so an event route authorizes when the caller has the required permission in *any* targeted team for read/RSVP, and — for create/update of a cross-team event — in *all* targeted teams for `events:write`. Single-team events keep today's exact behavior (the invariant is unchanged for them).
- **Attendance dedup:** key attendance by `user_id` at the event level (not per-team), so a multi-team member has exactly one row and one RSVP. The row carries the list of that user's memberships **intersected with the event's targeted teams** for labelling.
- **Restricted projection:** a `CrossTeamAttendee` view exposes only `{name, avatarColor, hasPhoto, teams[]}` and no membership/profile identifiers that would allow navigating to a foreign profile. The frontend renders team badges and disables profile navigation for attendees not in the viewer's own team.
- **Creation UI:** multi-team picker lists only teams where the creator has `events:write`; server re-validates.

## Risks / Trade-offs

- **This weakens the strongest invariant in the codebase** (`AND team_id = $N`). Mitigate by scoping the exception narrowly to event read/RSVP/attendance, re-validating on the server, and leaving all other modules strictly single-team. Document the exception in `CLAUDE.md`.
- Attendance keyed by user instead of (user, team) is a data-model change — migration must reconcile existing per-team attendance without loss.
- RBAC generation (`cmd/genrbac`, `matchRBACRoute`) assumes one `{teamId}`; multi-team events need an authorization path that doesn't fit the generated single-team table — design the check explicitly and keep single-team routes on the existing fast path.
- Privacy: the merged view must not expose emails, phone, or profile links across teams — only name/avatar/team badge. Verify with tests that no foreign PII leaks.
- Large surface area; consider landing behind a flag and after `consistent-profile-photos` (shared avatar) and `event-cancellation-lead-time`.
