## Why

**Status: DEFERRED (lowest priority).** Intentionally parked — do not start until the smaller frontend/API changes have landed. It is the largest, highest-risk change of the set (it relaxes the codebase's strongest invariant, `AND team_id = $N`) and depends on `consistent-profile-photos` and `event-cancellation-lead-time`. Kept as a validated proposal for later.

Clubs run joint sessions (multiple teams training/performing together). Today an event belongs to exactly one team — the whole backend enforces `AND team_id = $N` on every by-id query — so there's no way to invite several teams to one event, and members of each team can't see one shared attendance picture.

## What Changes

- Allow an event to target **multiple teams**. Members of any targeted team see the event and its attendance.
- In the **event view only**, show all attendees across the targeted teams with a **team label** per person. No access to other teams' member profiles from here — name + team badge + avatar, nothing clickable through to a profile.
- A person who belongs to **several of the targeted teams** appears **once**, RSVPs **once**, and is shown with the list of their memberships **limited to the event's targeted teams**.
- Creating a cross-team event is only allowed for a user who has **write on `events` in every targeted team**.

## Capabilities

### New Capabilities
- `cross-team-events`: events shared across multiple teams with a merged, profile-restricted attendance view.

## Impact

- Spec/backend: `openapi.yaml` event schemas gain a set of target team ids; create/update validate write-on-events across all targets; a join table `event_teams` (+ migration); event read/list, attendance assembly, and RBAC (`internal/middleware/authz.go`, generated RBAC table) all reworked to accept multi-team membership instead of a single `team_id`; attendance keyed by user (dedup across teams); a restricted cross-team member projection (name, avatar, team badges ⊆ targeted teams — **no** profile route). Regenerate clients; extensive tests.
- Frontend: event create form (multi-team picker gated by permission), event detail attendance grouped/badged by team, dedup of multi-team members, no profile navigation for cross-team attendees; `frontend/src/i18n/{de,en}.ts`; MSW.
- Docs: RBAC section in `CLAUDE.md` (team-scoping invariant now has a controlled multi-team exception).
- CI: openapi-drift, migration gates, backend + frontend gates. **Large, API + migration + RBAC-affecting — sequence carefully.**
