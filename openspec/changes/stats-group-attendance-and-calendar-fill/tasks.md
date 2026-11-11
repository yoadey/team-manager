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

## 8. Verification

- [ ] 8.1 `openspec validate stats-group-attendance-and-calendar-fill
      --strict`
- [ ] 8.2 Frontend: `npm run lint`, `npm run typecheck`, `npm test`
      (coverage thresholds 80/65/75/80), `npm run build` +
      `npm run check:bundle` (250 KB/chunk, 600 KB total gzipped)
- [ ] 8.3 Backend: `make generate` + `make generate-ts` leave no diff
      (openapi-drift), `go build ./...`, `make lint` (+ tool-pin-sync
      unaffected), `go test ./...` (note whether Docker-backed tests ran or
      skipped), coverage ≥ 35 % business logic, `govulncheck`
- [ ] 8.4 Migration gates: rollback (up → down → up) and migration-safety
      lint on `00037`
- [ ] 8.5 e2e suite unaffected or updated
- [ ] 8.6 Independent review by a fresh subagent; findings addressed
