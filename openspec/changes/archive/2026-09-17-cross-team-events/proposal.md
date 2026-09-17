## Why

**Status: ACTIVE.** Previously deferred pending `consistent-profile-photos` and `event-cancellation-lead-time` — both have since shipped and archived, so the stated blockers are cleared. It remains the largest, highest-risk change of the set (it relaxes the codebase's strongest invariant, `AND team_id = $N`), so implementation still needs to sequence carefully and keep single-team events on their existing, unchanged fast path.

Clubs run joint sessions (multiple teams training/performing together). Today an event belongs to exactly one team — the whole backend enforces `AND team_id = $N` on every by-id query — so there's no way to invite several teams to one event, and members of each team can't see one shared attendance picture.

## What Changes

- Allow an event to target **multiple teams**. Members of any targeted team see the event and its attendance.
- In the **event view only**, show all attendees across the targeted teams with a **team label** per person, per this display rule: a person who belongs to the **viewer's own (currently active) team** shows **no team badge**; otherwise the badge shows the **alphabetically-first** (by team name) team, among the event's targeted teams, that the person actually belongs to. No access to other teams' member profiles from here — name + team badge + avatar + attendance status, nothing clickable through to a profile.
- A person who belongs to **several of the targeted teams** appears **once**, RSVPs **once** across all of them.
- Creating a cross-team event is only allowed for a user who has **write on `events` in every targeted team**.

## Capabilities

### New Capabilities
- `cross-team-events`: events shared across multiple teams with a merged, profile-restricted attendance view.

## Impact

- Spec/backend: `openapi.yaml` event schemas gain a set of target team ids; create/update validate write-on-events across all targets; a join table `event_teams` (+ migration; attendance already dedups naturally — `attendance` is `UNIQUE(event_id, user_id)` with no `team_id` column, so no attendance migration is needed); event read/list, attendance assembly, and RBAC (`internal/middleware/authz.go`, generated RBAC table) all reworked to accept multi-team membership instead of a single `team_id`; a restricted cross-team member projection (name, avatar, single team badge per the display rule above — **no** profile route). Regenerate clients; extensive tests.
- Frontend: event create form (multi-team picker gated by permission), event detail attendance grouped/badged by team, dedup of multi-team members, no profile navigation for cross-team attendees; `frontend/src/i18n/{de,en}.ts`; MSW.
- Docs: RBAC section in `CLAUDE.md` (team-scoping invariant now has a controlled multi-team exception).
- CI: openapi-drift, migration gates, backend + frontend gates. **Large, API + migration + RBAC-affecting — sequence carefully.**
