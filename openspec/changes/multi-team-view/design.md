## Context

Today the frontend is built around exactly one team:

- `AppState.activeTeamId: string | null` is the only team in scope;
  `activeTeam()`, `can(module, level)` and `isStaff()` all evaluate against it.
- Query keys are already team-prefixed (`frontend/src/query/keys.ts`:
  `['teams', teamId, 'events']`, …), so caches of different teams live side by
  side and a team switch never mixes data. This is the foundation that makes a
  client-side fan-out cheap.
- The backend authorizes every team-scoped request against the single
  `{teamId}` in the path (`internal/middleware/authz.go`, generated
  `rbac_table.gen.go`), fail-closed for every method. The only multi-team
  exception is `cross-team-events` (`event_teams`), checked by hand in
  `events.Service`.
- A person in several teams has one `users` row and one `memberships` row per
  team; roles, groups, titles, absences, reason visibility, finances and stats
  thresholds are all per team.

Typical multi-team users have 2–5 teams; a club officer may have 10+.

## Goals / Non-Goals

**Goals**
- Let a member look at the schedule, roster, news, polls and notifications of
  several of *their own* teams in one view.
- Zero change to who may see or do what: every read and write stays authorized
  per team exactly as today.
- Single-team mode (one team selected) behaves exactly as today and is the
  degenerate case of the same code path, not a separate one.
- Make the team of every item visible at a glance.

**Non-Goals**
- Merging finances (separate cash boxes, team-specific penalty catalogs and
  fee groups) or statistics (team-specific rosters, roles and thresholds;
  quotas are not comparable across teams).
- Posting one news item / poll to several teams at once.
- A combined iCal feed across teams (the per-team feed stays; possible
  follow-up).
- Any server-side aggregate endpoint or RBAC change (see D2).
- Seeing teams the user is not a member of (that is `shared-calendar`).

## Decisions

### D1 — Selection model: a set, no home team

```
viewTeams: { mode: 'all' } | { mode: 'some'; ids: string[] }   // new, source of truth
activeTeamId: string | null   // derived: the team iff exactly one is selected, else null
```

- `selectedTeamIds()` resolves `all` against `state.teams` at read time, so a
  team the user joins later is automatically included in "All teams". An
  explicit list ignores ids the user is no longer a member of.
- **All selected teams are equal.** There is no home, primary or default team
  in multi-team mode, and no ordering among the selected teams carries
  meaning beyond display (alphabetical by team name).
- `activeTeamId` / `activeTeam()` remain as *derived* values: the single
  selected team in single-team mode (so today's code keeps working there), and
  `null` in multi-team mode. Every call site that needs one team must
  therefore obtain it explicitly — from the item it acts on (D3), from an
  explicit team choice (D5, D6), or by running per team. Making it `null`
  rather than "some team" lets TypeScript and tests surface every place that
  silently assumed a single team.
- Selecting a single team is today's `selectTeam`. Reducing a multi-team
  selection to one team makes that team the active one.
- `isMultiTeam()` ⇔ resolved selection has ≥ 2 teams. All multi-team UI
  (badges, pickers, filter chips) is gated on it, so single-team screens look
  unchanged.
- Persisted in localStorage (`tv_view_teams`, initialised from the existing
  `tv_active_team_id`); reflected in the URL as `?teams=all` or
  `?teams=<id>,<id>` on top-level routes (absent ⇒ single team, keeping
  existing links valid).

*Alternative considered:* keeping `activeTeamId` as a "home team" inside the
selection, serving as default create target and default for single-team-only
screens. Rejected by product decision: a hidden default team makes it easy to
create or edit something in a team the user did not consciously pick, and it
makes one selected team look more important than the others.

### D2 — Aggregation: client-side fan-out over existing endpoints

Multi-team hooks (`useEventsMulti(teamIds)`, `useMembersMulti`, …) use React
Query's `useQueries` with the **existing** per-team query keys and fetchers,
then merge.

- No new endpoint, no RBAC change, no OpenAPI change, no new attack surface.
- Cache is shared with single-team mode: switching from "All" to one team is
  instant and vice versa.
- Requests are only issued for teams where the caller has at least `read` on
  the module (`TeamForUser.myPerms`), so a team with `events: none` is simply
  not part of the events view — no guaranteed 403s.

*Alternative considered:* `GET /me/events?teamIds=…` and siblings. Rejected for
now: it would break the "one `{teamId}` per request" invariant the generated
RBAC table relies on and need hand-written per-team authorization like
`cross-team-events` — for every module. Revisit only if measurements show the
fan-out is too slow for users with many teams (tracked as an open question,
not built speculatively).

### D3 — Per-item team context ("via team")

Every merged item is wrapped as `{ ...item, viaTeamId }`, the team whose
endpoint returned it.

- All actions on the item use `viaTeamId` for the request path *and* for the
  permission check: `canForTeam(team(viaTeamId), module, level)`. Item-level
  code must not call `app.can()` (active team, `null` in multi-team mode); the
  refactor replaces those call sites with an item-aware helper
  (`app.canFor(item, module, level)`), which in single-team mode yields the
  same result as today.
- Page-level "create" buttons are shown if the caller has `write` in **any**
  selected team.
- Detail sheets (`openEventDetail`, member detail) receive the item's
  `viaTeamId` and fetch through it, instead of `state.activeTeamId`
  (`AppShell.tsx` currently passes `state.activeTeamId` to
  `useEventDetailQuery`).

### D4 — De-duplication

- **Events:** a cross-team event targeting two selected teams is returned by
  both teams' lists. Deduplicate by event id. Its `viaTeamId` is the team with
  the highest `events` permission (write > read), ties broken by the owning
  team first, then team name — so edit/cancel buttons appear whenever *any*
  path would allow them. The badge shows the owning team plus "+n".
- **Shared-calendar events** (redacted, from a team the viewer is not a member
  of): dropped when the same event id is already present in full through a
  selected team.
- **Members:** grouped by `userId`. One row per person with one badge per
  selected team they belong to. Per-team attributes (roles, group, title,
  joined date) are shown per team, never merged into one value. Profile fields
  (name, photo, phone, …) come from the same user and are shown once.
- **Birthdays** synthesized from members (`synthesizeBirthdayEvents.ts`):
  once per `userId`.
- **Absences** are stored per team and are *not* merged even if identical;
  each shows its team badge (see D6 for the create side).
- **Reasons** (attendance/absence reasons, governed by the per-team
  `reasonVisibilityRoles`) are only ever shown in the context of the team that
  returned them — never carried over to another team's row.

### D5 — Module matrix

| Module / screen | Multi-team behavior |
|---|---|
| Home | aggregated: next events, my pending RSVPs, latest news |
| Events — list | aggregated, team badge, team filter chips |
| Events — calendar | aggregated, entries colored by team, legend = filter chips |
| Events — absences tab | aggregated read; create can target several teams (D6) |
| Event detail | single event via its `viaTeamId`; attendance as today |
| Members | aggregated, deduped by person, grouped by team (default) or A–Z |
| Member detail | one membership; switcher when the person is in several selected teams |
| News, Polls | aggregated, chronological, team badge; pinned items of all teams first |
| Notifications | aggregated feed, unread count summed, "mark seen" fans out per team |
| Finances | **not merged** — team picker (selected teams with `finances:read`) |
| Statistics | **not merged** — team picker (selected teams with `stats:read`) |
| Team page, Settings (team parts) | **single team** — team picker; personal settings unaffected |

A team picker on a single-team-only screen shows only selected teams for which
the caller has the screen's read permission and does **not** change the
selection. Since there is no home team, it has no global default:

- exactly one qualifying team ⇒ that team is shown directly;
- several ⇒ the screen first asks which team to open (a short list of team
  cards), then shows that team with the picker in its header;
- the team last picked *on that screen* is remembered for that screen only
  (per screen, per selection) so that returning to finances doesn't ask again.
  This is a convenience local to the screen, not a team that other screens or
  create actions inherit;
- none ⇒ the route is hidden, as today (D5 route gating).

### D6 — Create flows

- Event, news, poll, … forms get a required "Team" field in multi-team mode,
  listing only selected teams with `write` on the module. **No team is
  pre-selected**: the user has to pick one, and saving is blocked until they
  do. If exactly one team qualifies, it is fixed and the field is shown
  read-only (so the user still sees where the item goes).
- Creating from a context that already implies a team pre-fills it: e.g.
  "new event" from a calendar day while the team filter shows only one team,
  or creating from inside a single-team-only screen (finances of team A ⇒
  team A).
- Fields that depend on the team (role nominations, fee groups, penalty
  catalog, reason visibility) stay disabled until a team is picked, and are
  reset when the team changes.
- The event form's existing cross-team target picker stays as is. The two
  concepts are kept explicit in the UI: *one event for several teams*
  (cross-team, merged attendance) vs. *separate events per team*. Multi-team
  mode does **not** silently turn a create into a cross-team event.
- **Absences** are the one deliberate fan-out create: "Absent from … to …" can
  be entered for several selected teams at once (default: all selected teams),
  issuing one `POST /teams/{id}/absences` per team; partial failures are
  reported per team.

### D7 — Visual identity

- `TeamBadge` component: team logo/icon + short name (`Team.short`, falling
  back to `name`), compact on mobile (icon only).
- Calendar entries use the team's `iconBg` as accent; colors that are too close
  fall back to a deterministic palette so teams stay distinguishable.
- App header: stacked team icons + "3 Teams"/"All teams" instead of one team
  name; tapping opens the switcher.

### D8 — URL & deep links

- `?teams=…` as in D1. Detail paths gain the via team:
  `/events/<id>?team=<viaTeamId>`, `/members/<membershipId>?team=<teamId>`.
- On load, the selection from the URL is intersected with the user's
  memberships; unknown ids are ignored. A detail link whose `team` is not in the
  current selection opens the detail anyway (if the user is a member) without
  changing the selection.

### D9 — Loading and partial failure

- A module renders once all its per-team queries have settled; while loading,
  the existing skeletons are shown.
- One failing team never blanks the page: merged results of the others are
  shown plus a dismissible notice "Team X could not be loaded — retry".
- A 403 for one team (permissions changed mid-session) triggers the existing
  team refresh and drops that team from the module.

### D10 — Interplay with `cross-team-events`

The server computes the attendee team badge relative to the `{teamId}` in the
URL — here the event's `viaTeamId`. In multi-team mode, attendees from other
*selected* teams therefore still carry a badge. This is kept unchanged (it is
informative and requires no API change); see open questions.

## Risks / Trade-offs

- **Broad refactor** (~37 files read `activeTeamId`, ~31 `app.can()` call
  sites). → Introduce `selectedTeamIds()` / `canFor(item, …)` first with
  single-team behavior identical to today, migrate module by module behind
  the selection (phases in tasks.md), and keep single-team tests untouched as
  a regression net.
- **Wrong-team actions** (e.g. RSVP or delete sent to another selected team
  instead of the item's team). → `viaTeamId` is mandatory on merged item types (enforced
  by TypeScript), plus tests per action that the request path uses the item's
  team.
- **Request fan-out** for users with many teams (N teams × modules). → Only
  readable teams are fetched, caches are shared, React Query de-duplicates;
  measure with 10+ teams before considering D2's aggregate endpoint.
- **Visual noise** from badges on every row. → Badges only in multi-team mode;
  team filter chips allow narrowing without changing the selection.
- **Bundle budget** (250 KB/chunk). → No new dependencies; `useQueries` is
  already part of TanStack Query.

- **Call sites assuming a single team** (`activeTeamId!`, `activeTeam()!`,
  `S().activeTeamId === teamId` race guards in `AppContext.tsx`). → These
  become `null` in multi-team mode by design (D1); each is migrated to an
  explicit team, and a test runs every page in multi-team mode to catch any
  missed non-null assertion.
- **Extra click when creating** (no default team). → Accepted trade-off for
  never creating in an unintended team; mitigated by context pre-fill and the
  single-qualifying-team shortcut (D6).

## Migration Plan

Frontend-only and additive. The stored single team (`tv_active_team_id`)
becomes the initial single-team selection, so nothing changes for existing
users until they pick a second team. Rollback = revert; no data migration.

## Open Questions

1. Should "All teams" become the default for users with several teams, or stay
   opt-in? (Proposal: opt-in.)
2. Should the cross-team attendee badge be suppressed for attendees of *any*
   selected team (needs an API parameter), or stay relative to `viaTeamId`
   (D10)?
3. Should overlapping events of the same person across teams be highlighted
   ("schedule conflict")? Cheap once events are merged; proposed as a follow-up.
4. Is a soft limit on the number of selected teams needed, or is the fan-out
   acceptable for club officers with 10+ teams? Decide after measuring.
5. Follow-ups worth a separate change: combined iCal feed for the selection,
   a personal cross-team "my open payments" summary, personal cross-team
   attendance statistics.
