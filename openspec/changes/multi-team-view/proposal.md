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
  unchanged. The selection is persisted (localStorage) and carried in the URL
  of every page, detail pages included.
- **No home team.** With several teams selected, no team is privileged and
  there is no implicit default team. The single-team context API
  (`activeTeamId`, `activeTeam()`, `can()`, `isStaff()`, `state.roles`) is
  replaced by team-explicit helpers; anything that needs one team takes it
  from the item it acts on or from an explicit user choice.
- **Aggregated modules** show the union of the selected teams, each item
  marked with a team badge (text/icon, not only color): Home, Events (list,
  calendar, absences), Members, News, Polls, Notifications.
- **Single-team-only screens** are deliberately *not* merged and use a team
  picker restricted to the selection, asking first when several teams
  qualify: Finances, Statistics, the Team page (team info, roles, invites,
  calendar shares, shared calendars), the roles editor, the team settings and
  the calendar subscription. Per-membership personal settings (profile title,
  push and reminder preferences) are shown as one section per selected team.
- **Every item and sheet keeps its team.** Actions and permission checks use
  the item's team. Cross-team events appear once and follow the backend's
  rules: edit/delete only through the owning team, status changes through any
  targeted team, a per-team view switch in the detail.
- **Duplicates are merged:** cross-team events, people in several teams,
  birthdays, and the per-team copies of cross-team event notifications.
- **Create actions require choosing a team** among teams with `write`, with
  no pre-selection unless only one team qualifies or the origin determines
  it. Absences can be entered for several teams at once, with no team
  pre-selected and a notice that the reason is visible to all members of the
  chosen teams.
- **No backend or API change.** Aggregation is a client-side fan-out over the
  existing per-team endpoints, loading only what is on screen, so RBAC stays
  exactly as it is: every request is still authorized against a single
  `{teamId}`.

## Capabilities

### New Capabilities
- `multi-team-view`: selecting several teams at once, how each module behaves
  in that mode (aggregated vs. single-team-only), per-item team context and
  permission evaluation, cross-team event handling, de-duplication of items,
  people and notifications, create-target selection, URL/persistence of the
  selection, and partial-failure handling.

### Modified Capabilities
- `cross-team-events`: the "viewer's own team" behind the attendee badge is
  defined for multi-team mode (the detail's view team).
- `state-based-routing`: detail URLs additionally carry the detail's team and
  the selection while several teams are selected.
- `client-data-fetching`: the team-scoped cache guarantee is extended from
  switching the active team to changing the set of selected teams.

## Impact

- Frontend state & routing: `frontend/src/context/AppContext.tsx`
  (`activeTeamId`, `activeTeam`, `can`, `isStaff`, `state.roles`,
  `selectTeam`, `afterLoginLoad`, `ensureRouteData`, `SheetState`),
  `frontend/src/context/urlState.ts` (selection + detail team in the URL),
  `frontend/src/features/team/components/NavSheets.tsx` (multi-select),
  `frontend/src/layouts/AppShell.tsx` (early return, header, nav badge,
  always-on queries, detail sheet data), `pages/index.tsx` (`RouteScreen`
  gating), and every other consumer of the removed single-team API
  (~37 files, ~31 `app.can(...)` call sites).
- Data layer: multi-team query hooks next to `features/*/hooks/use*Queries.ts`
  reusing `frontend/src/query/keys.ts` per-team keys; new
  `queryKeys.roles(teamId)`.
- Pages: `pages/Home.tsx`; `features/events/*` (list, `EventCalendar.tsx`,
  `EventAbsences.tsx`, `EventDetailSheet.tsx`, `EventFormSheet.tsx`,
  `AbsenceFormSheet.tsx`, `CalExportSheet.tsx`, `useEventActions.ts`,
  `useEventFormActions.ts`); `features/members/*`; `features/news/*`;
  `features/polls/*`; `features/notifications/*`; team picker for
  `features/finances/*`, `pages/Stats.tsx`, `features/team/*`,
  `features/settings/*` (`ProfilePanel.tsx`, `NotificationsPanel.tsx`
  per-team sections).
- Shared UI: team badge, team picker, team filter chips, target-team field in
  `components/`.
- i18n: new keys (de/en) in `frontend/src/i18n`.
- MSW demo backend (`frontend/src/mocks/`): the demo user already belongs to
  two teams with different roles; add a cross-team event and more overlapping
  members.
- Backend / `backend/openapi/openapi.yaml`: unchanged. A server-side aggregate
  endpoint is explicitly deferred (design.md, D2).
