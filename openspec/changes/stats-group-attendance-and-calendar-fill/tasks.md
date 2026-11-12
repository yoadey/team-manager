## 1. API contract and schema

- [x] 1.1 `backend/openapi/openapi.yaml`: add `EventGroupStat`,
      `StatsThresholdMode`, `StatsGroupThreshold`,
      `StatsGroupThresholdList`, `SetStatsGroupThresholdsRequest`; add
      required `groups` to `EventStat`; add `GET`/`PUT
      /teams/{teamId}/stats/group-thresholds` (`x-rbac-module: settings`).
- [x] 1.2 Migration `00037_role_stats_thresholds.sql` (table keyed by
      `role_id`, FK cascade, CHECKs; `goose Down` drops it).
- [x] 1.3 Regenerate: `cd backend && make generate` (oapi-codegen, genrbac,
      sqlc) and repo-root `make generate-ts`; commit generated output.

## 2. Backend

- [x] 2.1 `internal/stats/model.go`: `EventGroupStatRow`,
      `GroupThresholdRow`, default constants, `ErrRoleNotInTeam`,
      `ErrInvalidThreshold`.
- [x] 2.2 `internal/stats/repository.go`: `EventGroupStats` (same roster /
      effective status as `EventStats`, joined to roles + thresholds),
      `ListGroupThresholds`, `SetGroupThresholds` (single transaction,
      team-scoped through `roles`); extend `OverviewReader`.
- [x] 2.3 `internal/stats/service.go`: attach groups to each `EventStat`
      inside the existing `WithReadTx`; threshold validation;
      `ListGroupThresholds` / `SetGroupThresholds`.
- [x] 2.4 `internal/stats/handler.go` + `internal/server/server.go`:
      `ListStatsGroupThresholds`, `SetStatsGroupThresholds` (400 / 422
      mapping).
- [x] 2.5 Tests: service (group assembly, validation), handler (status
      mapping, 401), repository integration (group counts, multi-role
      member, pending, upsert, foreign role rejected atomically, cascade).

## 3. Frontend: data layer

- [x] 3.1 `types/index.ts`, `api/map.ts`: `EventGroupStat`,
      `StatsGroupThreshold`, `EventStat.groups`.
- [x] 3.2 `services/apiContract.ts`, `services/serviceLayerReal.ts`:
      `stats.groupThresholds`, `stats.setGroupThresholds`.
- [x] 3.3 `mocks/db.ts`, `mocks/handlers.ts`: threshold store, the two
      routes with the same validation/RBAC, `groups` on the overview.

## 4. Frontend: statistics page

- [x] 4.1 `pages/Stats.tsx`: `events` tab ("Termine"), per-group
      indicators (check / percent / dash, red below threshold), quota tab
      without the events list.
- [x] 4.2 `pages/Stats.tsx`: matrix count column directly after the name.
- [x] 4.3 `i18n/de.ts`, `i18n/en.ts`: new keys; drop keys that become
      unused.
- [x] 4.4 `pages/Stats.test.tsx`: tab, indicators, red marking, column
      order.

## 5. Frontend: team settings

- [x] 5.1 `teamSettingsSchema.ts`, `TeamSheets.tsx`: per-role threshold
      rows (mode toggle + number input) with validation.
- [x] 5.2 `useTeamActions.ts`: save thresholds with the settings, only when
      changed; query hook for loading them.
- [x] 5.3 Tests: `TeamSheets.test.tsx`, `useTeamActions.test.ts`,
      `serviceContract.test.ts` / `serviceLayerReal.test.ts`.

## 6. Frontend: calendar

- [x] 6.1 `EventsPage.tsx`, `EventCalendar.tsx`: full width, rows stretch
      to the available height, measured per-cell entry limits with the old
      limits as minimum/fallback.
- [x] 6.2 `EventCalendar.test.tsx`: no width cap, fallback limits,
      increased limits with a mocked `ResizeObserver`.

## 7. Docs

- [x] 7.1 `docs/end-user/statistik.md`, `docs/end-user/termine.md`.
- [x] 7.2 `CLAUDE.md`: RBAC note for the settings-gated threshold route.

## 8. Independent-review follow-ups

- [x] 8.1 Calendar: one busy day inflated all six week rows (`1fr` rows
      against an indefinite height). Rows are now `minmax(<measured>, auto)`
      so only the busy week grows; line-height estimate rounded up for
      absence/birthday chips; e2e regression test in `e2e/events.spec.ts`.
- [x] 8.2 Team settings: a failed threshold save after a successful team
      save is reported as a partial save, the team list still refreshes and
      the threshold list reloads.
- [x] 8.3 Statistics: a too-low group carries a warning icon, not colour
      alone.
- [x] 8.4 Matrix: count column pinned next to the pinned name column on
      desktop; inaccurate code comment removed.
- [x] 8.5 API: `SetStatsGroupThresholdsRequest.items.maxItems` 500 → 200 to
      match the handler's cap; regenerated.
- [x] 8.6 Backend: upserts in role-id order (no deadlock between concurrent
      saves), FK violation from a concurrently deleted role → 422, audit
      record `stats.thresholds_update`.
- [x] 8.7 Demo backend: comment at the group breakdown corrected; the
      pre-existing raw-vs-effective drift of the mock overview documented
      (design.md, Risks) rather than widened into this change.

## 9. Missing responses count as not present (owner decision)

- [x] 9.1 API: required `pending` on `EventStat` and `EventGroupStat`;
      regenerated.
- [x] 9.2 Backend: `EventStats` / `EventGroupStats` report `pending`
      (effective status `pending`) next to `counted`; `nominated` / `pct` /
      `enough` and the quotas unchanged; tests incl. not-nominated members.
- [x] 9.3 Frontend: group indicator and the event line measure against
      `counted + pending`; tests.
- [x] 9.4 Demo backend: per-event line-up moved to effective attendance
      (agrees with its matrix and the real backend); contract tests updated.
- [x] 9.5 Spec delta, design decision and end-user docs updated.
- [x] 9.6 Independent review of 9.1–9.5, findings fixed: members of roles
      the event does not nominate are not expected (the events module only
      stores `nominated_role_ids`, so the stats queries check it themselves);
      a stats-excluded member's silence is not pending; the demo backend
      treats a not-relevant absence as not expected; design formula and
      field descriptions corrected. Each new backend test verified to fail
      without its fix.
- [x] 9.7 Second independent review (of 9.6): SQL confirmed correct; the
      remaining demo-vs-backend edge cases all stem from the events module
      ignoring `nominated_role_ids` (pre-existing) and are documented in
      design.md (Risks) as a follow-up, not fixed here.

## 10. Verification

- [x] 10.1 `openspec validate stats-group-attendance-and-calendar-fill
      --strict`
- [x] 10.2 Frontend: `npm run lint`, `npm run typecheck`, `npm test`
      (coverage thresholds 80/65/75/80), `npm run build` +
      `npm run check:bundle` (250 KB/chunk, 600 KB total gzipped)
- [x] 10.3 Backend: `make generate` + `make generate-ts` leave no diff
      (openapi-drift), `go build ./...`, `make lint` (+ tool-pin-sync
      unaffected), `go test ./...` (Docker-backed tests ran, none
      skipped), business-logic coverage 74 % (gate: ≥ 35 %). `govulncheck`
      cannot run in the sandbox (vuln.go.dev unreachable); CI runs it and it
      is red there for reasons unrelated to this change (no dependency was
      added — see the PR discussion).
- [x] 10.4 Migration gates: rollback (up → down → up) and migration-safety
      lint on `00037`
- [x] 10.5 e2e suite: 18/18 pass, including the new calendar layout test
- [x] 10.6 Independent review by a fresh subagent; findings addressed (see
      group 8). The product question it raised -- whether members who gave
      no response count against a group being "complete" -- was decided by
      the owner and implemented in group 9.
