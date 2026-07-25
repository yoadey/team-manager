## Context

Events are single-team and their full payload includes attendance, comments, notes. Sharing a calendar must expose strictly less: a scheduling overlay. This is deliberately weaker than `cross-team-events` — no participation, no merged attendance, no profile exposure.

## Goals / Non-Goals

**Goals:**
- A team controls which other teams may see its calendar.
- Grantees see only time, location, title, and type per event.

**Non-Goals:**
- Any participation/RSVP by the grantee (that's `cross-team-events`).
- Exposing attendance, comments, notes, or member data.
- Bidirectional sharing implied by a single grant.

## Decisions

- **Model:** `calendar_shares(owner_team_id, viewer_team_id, created_at)`, unique per pair. Managing grants requires `settings:write` on `owner_team_id`.
- **Redacted projection:** a dedicated `SharedCalendarEvent` schema (id, title, type, date, startTime, endTime, location) — assembled by a query that selects *only* those columns; the full event serializer is never reused, so new sensitive fields can't leak by default.
- **Authorization:** a read endpoint (e.g. `GET /teams/{viewerTeamId}/shared-calendars` or `.../calendar-shares/{ownerTeamId}/events`) authorizes when the caller is a member of `viewerTeamId` and a `calendar_shares` row exists for (owner, viewer). No RBAC module gate beyond membership + the grant.
- **Frontend:** settings screen lists/edits share grants (pick target teams); a shared-calendar view renders the redacted events distinctly (e.g. muted overlay) and never renders attendance/comment UI for them.

## Risks / Trade-offs

- Leak-by-projection is the main risk: enforce redaction at the **query/serializer** level, not by hiding fields in the UI. Test that the endpoint returns none of attendance/comments/notes/participants.
- Revocation must be immediate — a revoked grant makes the read endpoint 403/404 at once.
- Interaction with `cross-team-events`: keep the two features distinct; a shared-calendar viewer is never an attendee.
- Location can itself be sensitive; it's explicitly in scope per the request, but document that sharing exposes event locations to the grantee team.
