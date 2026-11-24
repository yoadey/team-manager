## 1. Foundation: selection model (no visible change yet)

- [ ] 1.1 `context/AppContext.tsx`: add `viewTeams` (`all` | explicit ids) as
      source of truth, `selectedTeamIds()`, `isMultiTeam()`,
      `setViewTeams()`, `soleTeam()`; `selectTeam(id)` = single-team
      selection as today. Selection lifecycle per design D1 (confirm keeps
      route if reachable, shrink, removal, invite join).
- [ ] 1.2 Remove `activeTeamId`, `activeTeam()`, `can()`, `isStaff()` and
      `state.roles` from the context; add `canFor(teamId, …)`,
      `canAny(…)`; resolve every resulting compile error with an explicit
      team source (item, explicit choice, per-team loop, or `soleTeam()`);
      ESLint rule against non-null assertions on `soleTeam()`. Unit tests:
      single-team results equal today's `can()`.
- [ ] 1.3 `queryKeys.roles(teamId)` + `useRolesQuery(teamId)` replacing
      `state.roles` / `refreshRoles`; reason visibility from the item's team.
- [ ] 1.4 `SheetState.teamId` required for every team-scoped sheet; openers
      set it; sheets never fall back to a global team.
- [ ] 1.5 `layouts/AppShell.tsx`: render without a sole team; members query
      only where needed; nav badge from merged upcoming events.
- [ ] 1.6 Persistence (`tv_view_teams`, migrated from `tv_active_team_id`) and
      `context/urlState.ts`: `teams` on every path in multi-team mode,
      `team` on detail paths, precedence URL → stored → fallback; parse/build
      round-trip tests; old URLs keep their meaning.
- [ ] 1.7 Route gating (`ROUTE_MODULE`, `RouteScreen`, nav, `ensureRouteData`)
      via `canAny`.

## 2. Shared UI

- [ ] 2.1 `components/TeamBadge.tsx` (icon/logo + short name, compact mode
      with accessible name) and deterministic team color helper.
- [ ] 2.2 `TeamsSheet` (`features/team/components/NavSheets.tsx`):
      multi-select and "All teams", all teams shown equally; single tap
      keeps today's quick switch.
- [ ] 2.3 App header: stacked icons + "n Teams"/"All teams".
- [ ] 2.4 `TeamFilterChips` (hide/show a team in a view without changing the
      selection) and `TeamPicker` (single-team-only screens: asks first when
      several teams qualify, remembers the choice per screen).
- [ ] 2.5 `TargetTeamField` for create forms: required, no pre-selection,
      fixed when one team qualifies, context pre-fill, dependent fields
      disabled until chosen and reset on change.
- [ ] 2.6 Partial-failure notice with retry.
- [ ] 2.7 i18n keys (de/en).

## 3. Events, calendar, Home (phase 1)

- [ ] 3.1 `useEventsMulti(teamIds, scope)` via `useQueries` over existing
      `queryKeys.events`; only teams with `events:read`; `upcoming` scope for
      Home/badge; merge into `MergedEvent` with per-team copies (D4).
- [ ] 3.2 `EventsPage` list + filter chips, `EventCalendar` (color + icon per
      team, legend), `pages/Home.tsx`.
- [ ] 3.3 Cross-team rules (D4): participation team, owner-only
      edit/delete/duplicate/retarget, status via any qualifying team,
      detail view-team switch; badge from selected reaching teams.
- [ ] 3.4 `useEventActions`: every action uses the item's/sheet's team and
      `canFor`; tests asserting the request team per action.
- [ ] 3.5 `EventFormSheet` / `useEventFormActions`: `TargetTeamField`,
      nominations from `useRolesQuery(chosenTeam)` defaulting to all of that
      team's roles; quick-create and duplicate pre-fill; cross-team picker
      unchanged and clearly separated.
- [ ] 3.6 `CalExportSheet`: team choice in multi-team mode.
- [ ] 3.7 Birthdays deduped by `userId`.
- [ ] 3.8 Old `/events/<id>` links without `team` resolved through the
      selected teams (D8).

## 4. Members and absences (phase 2)

- [ ] 4.1 `useMembersMulti`: group by `userId`, per-team attributes kept per
      team; list grouped by team (default) or A–Z.
- [ ] 4.2 Member detail per membership with switcher; edit gated by
      `members:write` in that team; roles editor (from Members and Team page)
      via team choice.
- [ ] 4.3 Absences tab aggregated with team badges.
- [ ] 4.4 `AbsenceFormSheet`: multi-team target, no pre-selection,
      "reason visible to all members of …" notice, fan-out create, per-team
      failure report.

## 5. News, polls, notifications (phase 3)

- [ ] 5.1 News and polls aggregated (pinned first), team badge, create with
      `TargetTeamField`, vote and voters sheet via the poll's team.
- [ ] 5.2 Notifications aggregated and deduped (D5), unread count after
      merging, "mark seen" per holding team, click opens via participation
      team.

## 6. Single-team-only screens and settings (phase 3)

- [ ] 6.1 Finances, Statistics, Team page, team settings sheet use
      `TeamPicker`; the picker does not alter the selection.
- [ ] 6.2 `ProfilePanel` (title) and `NotificationsPanel` (push category and
      reminder preferences): one section per selected team.

## 7. Demo backend and tests

- [ ] 7.1 `mocks/db.ts`: add a cross-team event and more members shared by the
      two existing demo teams (the demo user is already in both with
      different roles).
- [ ] 7.2 Page/hook tests for every scenario in
      `specs/multi-team-view/spec.md` and the modified scenarios in
      `specs/cross-team-events`, `specs/state-based-routing`,
      `specs/client-data-fetching`; a test rendering every route in
      multi-team mode.
- [ ] 7.3 e2e: select two teams, merged events, RSVP to the other team's
      event, cross-team event edit rules, finances with picker; extend
      `e2e/accessibility.spec.ts` to the multi-team calendar.

## 8. Verification

- [ ] 8.1 `openspec validate multi-team-view --strict`
- [ ] 8.2 Frontend: lint, typecheck, test with coverage (80/65/75/80)
- [ ] 8.3 Frontend: build + `check:bundle` (250 KB/chunk, 600 KB total gzipped)
- [ ] 8.4 e2e suite
- [ ] 8.5 Backend gates unaffected (no backend change): openapi-drift, lint,
      test, govulncheck stay green
- [ ] 8.6 Independent code review by a fresh agent before merge
