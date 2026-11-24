## 1. Foundation: selection model (no visible change yet)

- [ ] 1.1 `context/AppContext.tsx`: add `viewTeams` (`all` | explicit ids),
      `selectedTeamIds()`, `isMultiTeam()`, `setViewTeams()`; keep
      `activeTeamId` as home team (always within the selection);
      `selectTeam(id)` = single-team selection as today.
- [ ] 1.2 Persist selection (`tv_view_teams`), migrate from
      `tv_active_team_id`; ignore ids the user is no longer a member of.
- [ ] 1.3 `context/urlState.ts`: `?teams=all|<ids>` on top-level routes;
      `?team=<id>` on `/events/<id>` and `/members/<id>`; parse + build +
      round-trip tests; old URLs keep their meaning.
- [ ] 1.4 Item-aware permissions: `canFor(item | teamId, module, level)` and
      `canAny(module, level)` over the selection; unit tests proving
      single-team results equal today's `can()`.
- [ ] 1.5 Merged item typing: `WithTeam<T> = T & { viaTeamId: string }`.

## 2. Shared UI

- [ ] 2.1 `components/TeamBadge.tsx` (logo/icon + short name, compact mode)
      and deterministic team color helper (distinct fallback palette).
- [ ] 2.2 `TeamsSheet` (`features/team/components/NavSheets.tsx`):
      multi-select, "All teams", home-team marker; single tap keeps today's
      quick switch.
- [ ] 2.3 `layouts/AppShell.tsx` header: stacked icons + "n Teams" in
      multi-team mode.
- [ ] 2.4 `TeamFilterChips` (hide/show a team inside a view without changing
      the selection) and `TeamPicker` (single-team-only screens, D5).
- [ ] 2.5 Partial-failure notice with retry (D9).
- [ ] 2.6 i18n keys (de/en).

## 3. Events, calendar, Home (phase 1)

- [ ] 3.1 `useEventsMulti(teamIds)` via `useQueries` over existing
      `queryKeys.events`; only teams with `events:read`; dedupe by id with
      via-team choice (D4); drop redacted shared-calendar duplicates.
- [ ] 3.2 `EventsPage` list + `EventCalendar` (team colors, legend/filter) +
      `pages/Home.tsx` (next events, pending RSVPs).
- [ ] 3.3 Event detail: `openEventDetail` / `useEventDetailQuery` use the
      item's `viaTeamId` instead of `state.activeTeamId`.
- [ ] 3.4 `useEventActions`: RSVP, comment, nominate, status, delete use
      `viaTeamId` and `canFor`; tests asserting the request team per action.
- [ ] 3.5 `EventFormSheet`: target-team field (writable teams only, default
      home team); cross-team picker unchanged and clearly separated.
- [ ] 3.6 Birthdays deduped by `userId`.

## 4. Members and absences (phase 2)

- [ ] 4.1 `useMembersMulti`: group by `userId`, per-team attributes kept
      per team; list grouped by team (default) or A–Z.
- [ ] 4.2 Member detail opened per membership, switcher for people in
      several selected teams; edit gated by `members:write` in that team.
- [ ] 4.3 Absences tab aggregated; reasons only in their own team's entry.
- [ ] 4.4 `AbsenceFormSheet`: multi-team target, fan-out create, per-team
      failure report.

## 5. News, polls, notifications (phase 3)

- [ ] 5.1 News and polls aggregated (pinned first), team badge, create with
      team field, vote via `viaTeamId`.
- [ ] 5.2 Notifications aggregated, summed unread count, "mark seen" fans out
      per team; notification click opens the item through its team.

## 6. Single-team-only screens (phase 3)

- [ ] 6.1 Finances, Statistics, Team page and team-scoped Settings use
      `TeamPicker` (selected teams with the screen's read permission, default
      home team); the picker does not alter the selection.
- [ ] 6.2 Route gating (`ROUTE_MODULE` / `RouteScreen`): a route is reachable
      if *any* selected team grants its read permission.

## 7. Demo backend and tests

- [ ] 7.1 `mocks/db.ts`: demo user in ≥ 2 teams with overlapping members,
      differing permissions per team, and a cross-team event.
- [ ] 7.2 `services/serviceContract.test.ts` / page tests for every scenario
      in `specs/multi-team-view/spec.md`.
- [ ] 7.3 e2e: select two teams, see merged events, RSVP to the other team's
      event, open finances with picker.

## 8. Verification

- [ ] 8.1 `openspec validate multi-team-view --strict`
- [ ] 8.2 Frontend: lint, typecheck, test with coverage (80/65/75/80)
- [ ] 8.3 Frontend: build + `check:bundle` (250 KB/chunk, 600 KB total gzipped)
- [ ] 8.4 e2e suite
- [ ] 8.5 Backend gates unaffected (no backend change): openapi-drift, lint,
      test, govulncheck stay green
- [ ] 8.6 Independent code review by a fresh agent before merge
