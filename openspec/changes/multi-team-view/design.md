## Context

Today the frontend is built around exactly one team:

- `AppState.activeTeamId: string | null` is the only team in scope;
  `activeTeam()`, `can(module, level)` and `isStaff()` all evaluate against it,
  and many call sites assert it non-null (`activeTeam()!` in `pages/Home.tsx`,
  `activeTeamId!` in `useRoleActions.ts`, …). `AppShell.tsx` renders nothing
  while no active team is set (`if (!team || !state.user) return null`).
- `state.roles` holds the roles of the active team only
  (`afterLoginLoad`/`refreshRoles` in `AppContext.tsx`); the event form reads
  it for role nominations, `canSeeComment` reads the active team's
  `reasonVisibilityRoles`.
- `SheetState` carries `eventId`/`membershipId` but no team; every sheet
  implicitly works on the active team.
- Query keys are already team-prefixed (`frontend/src/query/keys.ts`:
  `['teams', teamId, 'events']`, …), so caches of different teams live side by
  side and a team switch never mixes data. This is the foundation that makes a
  client-side fan-out cheap.
- `api.events.list(teamId, 'all')` walks every page (`serviceLayerReal.ts`),
  and `AppShell` always loads events, members and notifications of the active
  team (nav badge, detail sheets).
- The backend authorizes every team-scoped request against the single
  `{teamId}` in the path (`internal/middleware/authz.go`, generated
  `rbac_table.gen.go`), fail-closed for every method. The only multi-team
  exception is `cross-team-events` (`event_teams`):
  - reads, RSVP, comments, attendance and **status changes** work through any
    targeted team's URL (`eventScopedByAnyTargetTeam`);
  - **update and delete** only through the owning team
    (`UPDATE events … WHERE id = $1 AND team_id = $2`,
    `internal/events/repository.go`); the frontend mirrors this with
    `canEditOrDelete = canEdit && e.teamId === state.activeTeamId`
    (`EventDetailSheet.tsx`);
  - what a caller sees *depends on the team they read through*: `myStatus`
    folds in planned absences of that team only, attendees outside that team
    are redacted (name, avatar, status, badge — no profile, no reason), and
    the badge is computed relative to that team;
  - "event created/cancelled" notifications are enqueued **once per targeted
    team** (`events/service.go`).
- A person in several teams has one `users` row and one `memberships` row per
  team; roles, groups, titles, absences, push/reminder preferences, finances
  and stats thresholds are all per team. Absence reasons are visible to every
  member of the team the absence was entered in (`absences/service.go`,
  deliberately unprotected).
- Redacted shared-calendar events (`shared-calendar`) are only shown in the
  shared-calendar sheet on the Team page, never in the event list or
  calendar.

Typical multi-team users have 2–5 teams; a club officer may have 10+.

## Goals / Non-Goals

**Goals**
- Let a member look at the schedule, roster, news, polls and notifications of
  several of *their own* teams in one view.
- Zero change to who may see or do what: every read and write stays authorized
  per team exactly as today.
- Single-team mode (one team selected) looks and behaves as today for the
  user (data loading may change, D2) and is the degenerate case of the same
  code path, not a separate one.
- Make the team of every item visible at a glance, not by color alone.

**Non-Goals**
- Merging finances (separate cash boxes, team-specific penalty catalogs and
  fee groups) or statistics (team-specific rosters, roles and thresholds;
  quotas are not comparable across teams).
- Posting one news item / poll to several teams at once.
- A combined iCal feed across teams (the per-team feed stays; follow-up).
- Showing shared (redacted) calendars of non-member teams in the event list
  or calendar. They stay where they are today (Team page); integrating them
  would be a separate feature.
- Merging the attendance lists of a cross-team event across the selected
  teams (D4 offers a per-team switch instead; follow-up).
- Any server-side aggregate endpoint or RBAC change (see D2).

## Decisions

### D1 — Selection model: a set, no home team

```
viewTeams: { mode: 'all' } | { mode: 'some'; ids: string[] }   // new, source of truth
```

- `selectedTeamIds()` resolves `all` against `state.teams` at read time
  (alphabetical by team name), so a team the user joins later is
  automatically included. An explicit list ignores ids the user is no longer
  a member of.
- **All selected teams are equal.** There is no home, primary or default team
  in multi-team mode, and no ordering among the selected teams carries
  meaning beyond display.
- `isMultiTeam()` ⇔ resolved selection has ≥ 2 teams. All multi-team UI
  (badges, pickers, filter chips, per-team sections) is gated on it, so
  single-team screens look unchanged.
- **`activeTeamId`, `activeTeam()`, `can()`, `isStaff()` and `state.roles`
  are removed from the context.** They are replaced by
  `soleTeam(): TeamForUser | null` (the team in single-team mode, `null`
  otherwise) and team-explicit helpers (`canFor(teamId, module, level)`,
  `canAny(module, level)`, `useRolesQuery(teamId)`). Merely making
  `activeTeamId` null would not be caught by TypeScript — the field is
  already `string | null` and asserted non-null at many call sites — so the
  removal is what forces every one of the ~37 consumers to choose an explicit
  team source: the item it acts on (D3), an explicit user choice (D5, D6), a
  per-team loop, or `soleTeam()` where single-team-only behavior is intended.
  An ESLint rule forbids non-null assertions on `soleTeam()`.
- `AppShell` renders in multi-team mode (its `!team` early return becomes a
  "no teams at all" check).

**Selection lifecycle**

| Situation | Result |
|---|---|
| Single tap on a team in the switcher | single-team selection of that team, route → Home (today's `selectTeam`) |
| Confirming a multi-team selection | sheets close; current route kept if any selected team grants its read permission, else Home |
| Reducing the selection to one team | single-team mode for that team |
| Explicit selection loses teams (removed from team) | remaining teams stay; one left ⇒ single-team mode; none left ⇒ first team alphabetically, single-team (today's fallback) |
| Joining a team via invite | single-team view of the joined team (today: the invite is why the user is here) |
| Opening an invite for a team the user already belongs to (`alreadyMember`) | selection unchanged if that team is selected, otherwise single-team view of it |
| Creating a new team (`createTeam`) | single-team view of the new team |
| Team list on the Team page (`TeamPage.tsx`) | single-team mode: today's `selectTeam`; multi-team mode: switches the Team page's own picker (D6), never the selection |

**Persistence and URL**

- Persisted in localStorage (`tv_view_teams`, initialised from the existing
  `tv_active_team_id`).
- In multi-team mode **every** path carries the selection: `?teams=all` or
  `?teams=<id>,<id>`, detail paths included (D8). Precedence on load: `teams`
  in the URL → stored selection → today's single-team fallback. A URL without
  `teams` therefore never collapses a stored multi-team selection; old links
  keep their meaning for users who never selected several teams. Accepted
  side effect: the stored selection is shared by all tabs, so a tab opened
  on a URL without `teams` follows whatever selection another tab stored
  last.

*Alternative considered:* keeping `activeTeamId` as a "home team" inside the
selection, serving as default create target and default for single-team-only
screens. Rejected by product decision: a hidden default team makes it easy to
create or edit something in a team the user did not consciously pick, and it
makes one selected team look more important than the others.

### D2 — Aggregation: client-side fan-out over existing endpoints

Multi-team hooks (`useEventsMulti(teamIds, scope)`, `useMembersMulti`, …) use
React Query's `useQueries` with the **existing** per-team query keys and
fetchers, then merge.

- No new endpoint, no RBAC change, no OpenAPI change, no new attack surface.
- Cache is shared with single-team mode: switching from "All" to one team is
  instant and vice versa.
- Requests are only issued for teams where the caller has at least `read` on
  the module (`TeamForUser.myPerms`), so a team with `events: none` is simply
  not part of the events view — no guaranteed 403s.
- **Load only what is on screen.** Home and the nav pending-RSVP badge use
  the `upcoming` scope; the full `all` scope is loaded only by the events
  page's past tab and the calendar. Members are loaded only by screens that
  need them (member list, birthdays in the calendar, member detail) instead
  of unconditionally by `AppShell`. Notifications stay always-on (badge).
- **Scoped event keys.** Today `queryKeys.events(teamId)` has no scope and
  `useEventsQuery` always fetches `'all'`. The key becomes
  `queryKeys.events(teamId, scope)` = `['teams', teamId, 'events', scope]`;
  the prefix `['teams', teamId, 'events']` is unchanged, so every existing
  prefix invalidation keeps hitting all scopes (scope values cannot collide
  with the UUID segment of `eventDetail`). This also changes single-team
  mode: Home and the badge load `upcoming` there too — a loading change only,
  with no visible difference. The cache stays shared between single- and
  multi-team mode because both use the same scoped keys.
- **Known cost:** the calendar and the past tab load each team's full history
  (`all`, every page) — the events API has no date-range filter. Acceptable
  for typical team sizes; a date-range parameter would be an API change and
  is left to open question 4.
- **Cross-copy invalidation.** A mutation on a cross-team event invalidates
  the event list and detail of **every** team the event reaches (owner +
  `crossTeamIds`), not only the team it was sent through — today delete
  (`useEventMutations.ts`) invalidates only the owning team and RSVP/comment
  only the request's team, which would leave a deleted event or a stale
  `myStatus` visible through another selected team's cached copy. Status and
  save already invalidate all reaching teams.

*Alternative considered:* `GET /me/events?teamIds=…` and siblings. Rejected for
now: it would break the "one `{teamId}` per request" invariant the generated
RBAC table relies on and need hand-written per-team authorization like
`cross-team-events` — for every module. Revisit only if measurements with
10+ teams show the fan-out is too slow (open question 4).

### D3 — Per-item team context

Every merged item carries the team(s) it was obtained through.

- Single-team items (news, polls, absences, members' memberships, events of
  one team): `{ ...item, teamId }`. All actions use that team for the request
  path *and* the permission check (`canFor(item.teamId, …)`).
- Page-level "create" buttons are shown if the caller has `write` in **any**
  selected team (`canAny`).
- **Sheets carry their team.** `SheetState` gains a `teamId` for every
  team-scoped sheet (event detail/form/series action/comment, attendance,
  poll voters, member detail/edit, absence form, role editor, calendar
  export …), set by whoever opens it; sheet code never falls back to a global
  team. It is required for sheets that act on an existing item, and `null`
  only for create forms opened without a determined team — those keep the
  chosen target team in their form state (D7).
- Team-dependent reference data is loaded per team: roles via a new
  `queryKeys.roles(teamId)` / `useRolesQuery(teamId)` (replacing
  `state.roles`), reason visibility from `team(teamId).reasonVisibilityRoles`
  and the caller's roles in *that* team.

### D4 — Cross-team events

A cross-team event targeting several selected teams is returned by each of
those teams' lists, and the copies differ (D-Context). It is shown **once**,
keeping every per-team copy internally:

```
MergedEvent = { id, copies: Record<teamId, TeamEvent>, reachTeamIds: string[] }
```

- **Participation team** (`myStatus`, RSVP, comment, reminder state): chosen
  among the selected teams **whose copy was actually loaded** (a team with
  `events: none` yields no copy and would answer an RSVP with 403): the
  owning team if it qualifies, otherwise the alphabetically first one. This
  is also the default *view team* of the detail sheet. Since a person RSVPs
  once per event (`cross-team-events`), only a status *derived* from a
  planned absence can differ between teams; it follows the participation
  team's absences. Accepted consequence: an absence entered only in another
  selected team leaves the merged event "pending" (and counted in the
  pending badge) — exactly what the participation team itself reports.
- **Edit, delete, retarget:** offered only through the owning team, and only
  if the owning team is selected and the caller has `events:write` there —
  exactly the backend rule (`UPDATE/DELETE … AND team_id = $owner`). In
  single-team mode this is today's `canEditOrDelete` unchanged.
- **Duplicate** is a plain client-side create, not a backend operation on the
  event: offered whenever the caller has `events:write` in any selected team
  (today it follows `canEdit`, and `useEventFormActions.ts` explicitly
  supports duplicating through a non-owning team). The target team follows
  D7.
- **Cancel / reactivate (status):** offered if the caller has `events:write`
  in any selected team that reaches the event; sent through the owning team
  if it qualifies, otherwise the alphabetically first qualifying team.
- **Inside the detail**, everything that today hangs on `canEdit` — editing
  other members' attendance rows, nominations, RSVPing past the cutoff — uses
  `canFor(viewTeam, 'events', 'write')`. Consequence: RSVPing through the
  participation team respects that team's cutoff even if write permission in
  another reaching team would bypass it; switching the view team to that
  team makes the bypass available.
- **Detail view:** shows the event and attendance as read through the view
  team. When several selected teams reach the event, a switch ("view as
  team X / Y") changes the view team; attendance, redaction and badges are
  then those the backend returns for that team. No client-side merge of
  attendance lists.
- **Badge** in lists: the selected teams that reach the event (never the
  owning team's name unless the user is a member there — `state.teams` only
  knows the user's own teams).
- **Filter chips and color:** an event reaching several selected teams is
  shown as long as at least one of its reaching teams is visible in the
  filter; its accent color is that of the owning team if visible, otherwise
  of the alphabetically first visible reaching team; it shows the icons of
  all visible reaching teams.

### D5 — De-duplication (everything except events)

- **Notifications:** a cross-team event's `event_created` / `event_cancelled`
  notification exists once per targeted team (`events/service.go`). Only
  these two types are merged, by `(type, eventId, actorId)` with `createdAt`
  within 60 s — attendance notifications are never fanned out and must not be
  merged (a quick yes→no change would otherwise collapse). The merged entry
  takes its fields (incl. `actorMembershipId`) from the copy of the event's
  owning team if loaded, otherwise of the alphabetically first team holding a
  copy. The unread count is computed *after* merging. "Seen" is not per
  notification: `POST /teams/{teamId}/notifications/seen` moves one per-team
  `seen_at` timestamp. As today, opening the notifications sheet marks seen —
  now in every selected team **whose notifications were loaded**; a team that
  failed to load (D9) is not marked, so its unseen items are not swallowed.
  Clicking an entry opens the event through the participation team (D4).
- **Members:** the default list is alphabetical with **one row per person**
  (grouped by `userId`) and one badge per selected team they belong to.
  Per-team attributes (roles, group, title, joined date) are shown per team,
  never merged into one value. Profile fields (name, photo, phone, …) come
  from the same user and are shown once. An optional "group by team" view
  lists each selected team's roster as its own section; there a person in
  several teams appears in each of their teams' sections (it is a per-team
  roster view), with a hint naming their other selected teams.
- **Birthdays** synthesized from members (`synthesizeBirthdayEvents.ts`):
  once per `userId`.
- **Absences** are stored per team and are *not* merged even if identical;
  each shows its team badge (see D7 for creating).
- **Attendance reasons** (governed by the per-team `reasonVisibilityRoles`)
  are only shown in the context of the team that returned them. Absence
  reasons are team-wide by design and need no extra rule beyond D7.

### D6 — Module matrix

| Module / screen | Multi-team behavior |
|---|---|
| Home | aggregated (upcoming scope): next events, my pending RSVPs, latest news |
| Home hero & quick stats | hero shows the selected teams (icons + names) instead of one team photo/name; upcoming-event count from merged events; member counts per team from `TeamForUser.memberCount` (no member data loaded, no misleading sum across teams) |
| Nav pending-RSVP badge | sum over merged (deduplicated) upcoming events |
| Events — list | aggregated, team badge, team filter chips |
| Events — calendar | aggregated, team accent color **plus** team short label/icon on each entry; legend = filter chips |
| Events — calendar subscription (export sheet, opened from Events page) | **per team**: the sheet asks which team's feed to manage |
| Events — absences tab | aggregated read; create for several teams (D7) |
| Event detail | D4 (view team, switch) |
| Members | aggregated; A–Z with one row per person (default), optional per-team roster view (D5) |
| Member detail | one membership; switcher when the person is in several selected teams |
| Roles editor (opened from Members and Team page) | **single team** — team choice first |
| News, Polls | aggregated, chronological, team badge; pinned items of all teams first |
| Notifications | aggregated, deduped (D5) |
| Finances | **not merged** — team picker |
| Statistics | **not merged** — team picker |
| Team page (team info, roles, invites, calendar shares, shared calendars) | **single team** — team picker |
| Settings — team settings sheet (incl. stats group thresholds) | **single team** — team picker |
| Settings — per-membership personal settings (profile title, push category & reminder preferences) | one section **per selected team**, labelled with the team |
| Settings — purely personal (appearance, account, privacy, legal) | unchanged |

**Team picker** on single-team-only screens: lists only selected teams for
which the caller has the screen's read permission and does **not** change the
selection. Exactly one qualifying team ⇒ shown directly; several ⇒ the screen
first asks which team to open (team cards), then shows it with the picker in
its header; the choice is remembered per screen (per selection) so returning
doesn't ask again — a convenience local to that screen, never inherited by
other screens or forms; none ⇒ route hidden.

**Route gating** (`ROUTE_MODULE`, `RouteScreen`, nav, `ensureRouteData`): a
route is reachable if *any* selected team grants its read permission;
`ensureRouteData`'s `can()` early return becomes `canAny()` so its
notifications refresh still runs.

### D7 — Create flows

- Event, news, poll, … forms get a required "Team" field in multi-team mode,
  listing only selected teams with `write` on the module. **No team is
  pre-selected**: saving is blocked until the user picks one. If exactly one
  team qualifies, it is fixed and shown read-only.
- **Context pre-fill** where the origin determines the team, and only if the
  caller has `write` there (otherwise the field starts empty):
  calendar quick-create while the team filter shows exactly one team;
  creating from inside a single-team-only screen; **duplicating** an event
  pre-fills its owning team (no write in any selected team ⇒ duplicate not
  offered, D4).
- Fields that depend on the team (role nominations — default "all roles" *of
  the chosen team* —, the cross-team target picker, fee groups, penalty
  catalog, reason visibility) stay disabled until a team is picked and are
  reset when the team changes. The cross-team picker excludes the chosen
  target team (today it excludes `activeTeamId`).
- The event form's cross-team target picker stays as is. Multi-team mode does
  **not** silently turn a create into a cross-team event; *one event for
  several teams* and *separate events per team* stay visibly distinct.
- **Absences** are the one fan-out create: one period can be entered for
  several selected teams in one step (one `POST /teams/{id}/absences` per
  team). **No team is pre-selected.** Because an absence reason is visible to
  all members of each team it is entered in, the form shows
  "Reason visible to all members of X, Y" for the chosen teams before saving.
  Partial failures are reported per team; successful ones are kept.

### D8 — URL & deep links

- `?teams=…` on every path in multi-team mode (D1).
- Detail paths carry the team the detail is shown through:
  `/events/<id>?team=<viewTeamId>&teams=…`,
  `/members/<membershipId>?team=<teamId>&teams=…`.
- A detail link whose `team` is not in the current selection opens the detail
  anyway if the user is a member of that team, without changing the
  selection.
- An old `/events/<id>` link without `team` in multi-team mode: the event is
  requested through the selected teams with `events:read` (in parallel);
  once all requests have settled, the owning team's copy is shown if one was
  returned, otherwise the alphabetically first returned copy; if none
  returns it, the usual "not found" handling applies.

### D9 — Loading and partial failure

- A module renders once all its per-team queries have settled; while loading,
  the existing skeletons are shown.
- One failing team never blanks the page: merged results of the others are
  shown plus a dismissible notice "Team X could not be loaded — retry".
- A 403 for one team's **module data** (permissions changed mid-session)
  triggers the existing team refresh and drops that team from the module.
  403s that are normal today are excluded: the roles list requires
  `settings:read`, so `useRolesQuery` treats a 403 as "no roles" (as
  `afterLoginLoad` swallows it today) and never drops a team.

### D10 — Visual identity and accessibility

- `TeamBadge`: team logo/icon + short name (`Team.short`, falling back to
  `name`), compact on mobile (icon only, accessible name = team name).
- Calendar entries: team accent color *and* team icon/short label, so teams
  are distinguishable without color (WCAG 1.4.1); colors too close to each
  other fall back to a deterministic palette.
- App header: stacked team icons + "n Teams"/"All teams"; tapping opens the
  switcher. All selected teams are displayed equally.

## Risks / Trade-offs

- **Broad refactor** (~37 files read `activeTeamId`, ~31 `app.can()` call
  sites, `state.roles`, `SheetState`). → Removing the single-team API (D1)
  turns every site into a compile error that has to be resolved
  deliberately; migrate module by module (phases in tasks.md); single-team
  tests stay untouched as a regression net; a test renders every route in
  multi-team mode.
- **Wrong-team actions** (RSVP, delete, vote sent to another selected team).
  → Required team on merged items and sheets (typed), plus a test per action
  asserting the request path's team.
- **Cross-team event semantics** are subtle (D4). → Rules mirror the backend
  exactly (edit/delete owner-only, status any target); dedicated tests for
  owning team selected / not selected / no write.
- **Request fan-out** for users with many teams. → Only readable teams, only
  on-screen modules, `upcoming` scope for Home/badge, shared caches. The
  calendar and past tab still load every team's full history (no date-range
  filter in the API), so 10+ teams with long histories can be slow; measure
  before considering a date-range parameter or an aggregate endpoint.
- **Visual noise** from badges. → Badges only in multi-team mode; team filter
  chips narrow a view without changing the selection.
- **Extra click when creating** (no default team). → Accepted trade-off for
  never creating in an unintended team; mitigated by context pre-fill and the
  single-qualifying-team shortcut.
- **Bundle budget** (250 KB/chunk). → No new dependencies; `useQueries` is
  part of TanStack Query.

## Migration Plan

Frontend-only and additive. The stored single team (`tv_active_team_id`)
becomes the initial single-team selection, so nothing changes for existing
users until they pick a second team. Rollback = revert; no data migration.

## Open Questions

1. Should "All teams" become the default for users with several teams, or stay
   opt-in? (Proposal: opt-in.)
2. Should the attendance lists of a cross-team event be merged across the
   selected teams instead of switching the view team (D4)? Needs care with
   per-team reason visibility; proposed as a follow-up.
3. Should overlapping events of the same person across teams be highlighted
   ("schedule conflict")? Cheap once events are merged; follow-up.
4. Is a soft limit on the number of selected teams needed, or is the fan-out
   acceptable for club officers with 10+ teams? Would a date-range filter on
   the events list (API change) be worth it for the calendar? Decide after
   measuring.
5. Follow-ups worth a separate change: combined iCal feed for the selection,
   shared calendars in the main calendar, a personal cross-team "my open
   payments" summary, personal cross-team attendance statistics.
