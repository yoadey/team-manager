## 1. Database
- [x] 1.1 `00027_event_stats_exclusion.sql`: `ALTER TABLE event_series ADD COLUMN
      exclude_from_stats boolean NOT NULL DEFAULT false;` and
      `ALTER TABLE events ADD COLUMN exclude_from_stats boolean NOT NULL DEFAULT false;`
- [ ] 1.2 `make migrate` locally if Docker is available; otherwise rely on CI's
      `backend-migration-rollback` (up→down→up) and `backend-migration-safety` gates
      (no Docker in this dev sandbox -- deferred to CI)

## 2. OpenAPI
- [x] 2.1 `TeamEvent`, `CreateEventRequest`, `UpdateEventRequest`: add
      `excludeFromStats: boolean` (default `false`)
- [x] 2.2 Series-template fields (wherever `cancelLeadMinutes` lives on the
      recurring-series request shape): add the same field
- [x] 2.3 `cd backend && make generate` (commit `internal/gen/api.gen.go`)
- [x] 2.4 repo-root `make generate-ts` (commit `frontend/src/api/types.gen.ts`)

## 3. Backend: events module
- [x] 3.1 `model.go`: `ExcludeFromStats bool` on `EventRow`, `EventSeriesRow`,
      `CreateEventParams`, `UpdateEventParams`
- [x] 3.2 `repository.go`: `CreateSeries` inserts `exclude_from_stats` into both
      the `event_series` row and every generated `events` row (same list as
      `cancel_lead_minutes`); `buildEventUpdateSets` gains the field so
      `UpdateEvent`'s `scope == "series"` path (`updateSeriesEvents`) updates
      every occurrence of the series (no date filter, mirroring
      `cancel_lead_minutes`'s existing behavior) and `scope == "single"`
      updates only the targeted occurrence

## 4. Backend: stats module
- [x] 4.1 `stats/repository.go`: add `AND e.exclude_from_stats = false` to the
      event predicate in `MemberStats`, `EventStats`, `AbsenceStats`,
      `SingleMemberStats`, and the attendance-matrix queries
      (`matrixColumns`/`matrixCells`)

## 5. Backend: tests
- [x] 5.1 `events/repository_test.go`: series creation seeds the flag onto all
      occurrences; scope=series edit applies to every occurrence; scope=single
      edit overrides one occurrence without touching the series/other occurrences
- [x] 5.2 `stats/repository_test.go`: an excluded event contributes to no
      member quote, no matrix cell/column, no event-stats row, while a
      non-excluded event in the same range is unaffected

## 6. Frontend
- [x] 6.1 `features/events/components/eventFormSchema.ts`: add
      `excludeFromStats: z.boolean().default(false)`
- [x] 6.2 `features/events/components/EventFormSheet.tsx`: add a toggle
      "Nicht in Statistik werten" near `EventTypeSelector`; reuse the existing
      `SeriesEditSubmit` single-vs-series scope dialog on edit, no new UX
- [x] 6.3 `api/map.ts`: map `excludeFromStats` both directions
- [x] 6.4 `services/serviceLayerReal.ts`: pass the field through create/update
      event calls
- [x] 6.5 `mocks/db.ts` + `mocks/handlers.ts`: seed field, honor it in the MSW
      demo backend's stats computations
- [x] 6.6 `i18n/{de.ts,en.ts}`: label + helper text for the new toggle

## 7. Verification
- [x] 7.1 `cd backend && make test && make lint` (integration tests skip: no
      Docker in this sandbox; unit tests + lint green)
- [x] 7.2 `cd frontend && npm run typecheck && npm run lint && npm test`
- [x] 7.3 `make generate && make generate-ts` produce no drift (CI's
      `backend-openapi-drift` gate)
- [ ] 7.4 Manual: create a series with the toggle on, confirm every generated
      occurrence is excluded from the stats page; flip one occurrence back on
      individually and confirm only that occurrence now counts
