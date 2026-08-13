## 1. Database
- [ ] 1.1 `00027_event_stats_exclusion.sql`: `ALTER TABLE event_series ADD COLUMN
      exclude_from_stats boolean NOT NULL DEFAULT false;` and
      `ALTER TABLE events ADD COLUMN exclude_from_stats boolean NOT NULL DEFAULT false;`
- [ ] 1.2 `make migrate` locally if Docker is available; otherwise rely on CI's
      `backend-migration-rollback` (up→down→up) and `backend-migration-safety` gates

## 2. OpenAPI
- [ ] 2.1 `TeamEvent`, `CreateEventRequest`, `UpdateEventRequest`: add
      `excludeFromStats: boolean` (default `false`)
- [ ] 2.2 Series-template fields (wherever `cancelLeadMinutes` lives on the
      recurring-series request shape): add the same field
- [ ] 2.3 `cd backend && make generate` (commit `internal/gen/api.gen.go`)
- [ ] 2.4 repo-root `make generate-ts` (commit `frontend/src/api/types.gen.ts`)

## 3. Backend: events module
- [ ] 3.1 `model.go`: `ExcludeFromStats bool` on `EventRow`, `EventSeriesRow`,
      `CreateEventParams`, `UpdateEventParams`
- [ ] 3.2 `repository.go`: `CreateSeries` inserts `exclude_from_stats` into both
      the `event_series` row and every generated `events` row (same list as
      `cancel_lead_minutes`); `UpdateEvent`'s `scope == "series"` path updates
      `event_series.exclude_from_stats` and propagates to future
      (`date >= CURRENT_DATE`) occurrences; `scope == "single"` updates only
      the targeted occurrence

## 4. Backend: stats module
- [ ] 4.1 `stats/repository.go`: add `AND e.exclude_from_stats = false` to the
      event predicate in `MemberStats`, `EventStats`, `AbsenceStats`,
      `SingleMemberStats`, and the attendance-matrix queries
      (`matrixColumns`/`matrixCells`)

## 5. Backend: tests
- [ ] 5.1 `events/repository_test.go`: series creation seeds the flag onto all
      occurrences; scope=series edit updates future occurrences only; scope=single
      edit overrides one occurrence without touching the series/other occurrences
- [ ] 5.2 `stats/repository_test.go`: an excluded event contributes to no
      member quote, no matrix cell/column, no event-stats row, and no absence
      stats row, while a non-excluded event in the same range is unaffected

## 6. Frontend
- [ ] 6.1 `features/events/components/eventFormSchema.ts`: add
      `excludeFromStats: z.boolean().default(false)`
- [ ] 6.2 `features/events/components/EventFormSheet.tsx`: add a toggle
      "Nicht in Statistik werten" near `EventTypeSelector`; reuse the existing
      `SeriesEditSubmit` single-vs-series scope dialog on edit, no new UX
- [ ] 6.3 `api/map.ts`: map `excludeFromStats` both directions
- [ ] 6.4 `services/serviceLayerReal.ts`: pass the field through create/update
      event calls
- [ ] 6.5 `mocks/db.ts` + `mocks/handlers.ts`: seed field, honor it in the MSW
      demo backend's stats computations
- [ ] 6.6 `i18n/{de.ts,en.ts}`: label + helper text for the new toggle

## 7. Verification
- [ ] 7.1 `cd backend && make test && make lint`
- [ ] 7.2 `cd frontend && npm run typecheck && npm run lint && npm test`
- [ ] 7.3 `make generate && make generate-ts` produce no drift (CI's
      `backend-openapi-drift` gate)
- [ ] 7.4 Manual: create a series with the toggle on, confirm every generated
      occurrence is excluded from the stats page; flip one occurrence back on
      individually and confirm only that occurrence now counts
