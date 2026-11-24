## Why

A user who belongs to several teams (a player in two squads, a coach of the
youth and the senior team, a club officer) can only ever look at **one** team
at a time. `AppState.activeTeamId` (`frontend/src/context/AppContext.tsx`) is a
single id, every page reads exactly that team
(`useEventsQuery(app.api, state.activeTeamId)` in `pages/Home.tsx`,
`features/events/EventsPage.tsx`, `features/members/MembersPage.tsx`,
`layouts/AppShell.tsx`, …), and `selectTeam` resets the route to Home on every
switch. To answer "what do I have on this week?" or "is anyone in both squads
double-booked on Saturday?" the user has to switch back and forth and keep the
other team's schedule in their head.

The existing cross-team mechanisms do not cover this: `cross-team-events` lets
*one event* target several teams, and `shared-calendar` exposes a redacted
schedule of a team the viewer is *not* a member of. Neither gives a member a
combined view of the teams they already belong to.

## What Changes

- **Team selection becomes a set.** The team switcher (`TeamsSheet` in
  `features/team/components/NavSheets.tsx`) gains a multi-select mode and an
  "All teams" shortcut. Selecting exactly one team is today's behavior,
  unchanged. The selection is persisted (localStorage) and reflected in the URL.
- **A "home team" stays.** `activeTeamId` is kept as the *home team* of the
  selection: the default target for create actions and the team the
  single-team-only screens open on.
- **Aggregated modules** show the union of the selected teams, each item marked
  with a team badge (and, in the calendar, the team's color):
  Home, Events (list, calendar, absences), Members, News, Polls,
  Notifications.
- **Single-team-only modules** are deliberately *not* merged and show a team
  picker restricted to the selection instead: Finances, Statistics, the Team
  page and every team-scoped part of Settings (roles, invites, calendar shares,
  group thresholds, calendar feed, push preferences).
- **Every item keeps its team context.** Each merged item carries the team it
  was read through ("via team"); actions on it (RSVP, comment, edit, delete,
  vote, open detail) and their permission checks use that team, never the home
  team.
- **Create actions ask for a team** (pre-selected: home team if writable),
  offering only teams in which the caller has `write` on the module.
- **No backend or API change in the first iteration.** Aggregation is a
  client-side fan-out over the existing per-team endpoints, so RBAC stays
  exactly as it is: every request is still authorized against a single
  `{teamId}`.

## Capabilities

### New Capabilities
- `multi-team-view`: selecting several teams at once, how each module behaves
  in that mode (aggregated vs. single-team-only), per-item team context and
  permission evaluation, de-duplication of items and people that appear through
  more than one team, create-target selection, URL/persistence of the
  selection, and partial-failure handling.

### Modified Capabilities
- None. `cross-team-events`, `shared-calendar`, `authorization` and
  `state-based-routing` keep their requirements; this change builds on them
  (see design.md for the interplay).

## Impact

- Frontend state & routing: `frontend/src/context/AppContext.tsx`
  (`activeTeamId`, `selectTeam`, `can`, `activeTeam`),
  `frontend/src/context/urlState.ts` (selection + detail team in the URL),
  `frontend/src/features/team/components/NavSheets.tsx` (multi-select),
  `frontend/src/layouts/AppShell.tsx` (header indicator, detail sheet data).
- Data layer: new multi-team query hooks next to
  `features/*/hooks/use*Queries.ts`, reusing `frontend/src/query/keys.ts`
  per-team keys so single- and multi-team views share one cache.
- Pages: `pages/Home.tsx`, `features/events/*` (list, `EventCalendar.tsx`,
  `EventAbsences.tsx`, `EventFormSheet.tsx`, `useEventActions.ts`),
  `features/members/*`, `features/news/*`, `features/polls/*`,
  `features/notifications/*`; team picker for `features/finances/*`,
  `pages/Stats.tsx`, `features/team/*`, `features/settings/*`.
- Shared UI: a team badge / team color component in `components/`.
- i18n: new keys (de/en) in `frontend/src/i18n`.
- MSW demo backend (`frontend/src/mocks/`): demo user needs ≥2 teams with
  overlapping members and a cross-team event so the mode is demoable/testable.
- Backend / `backend/openapi/openapi.yaml`: unchanged in this change. A
  server-side aggregate endpoint is explicitly deferred (design.md, D2).
- Scale of the refactor: ~37 non-test frontend files read `activeTeamId`,
  ~31 call sites use `app.can(...)` against the active team.
