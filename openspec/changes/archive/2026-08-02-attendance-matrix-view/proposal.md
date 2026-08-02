## Why

The statistics page answers "how present is each member?" and "how full was each event?" only as **aggregates** (a per-member quote, a per-event yes-count). It cannot answer the concrete question a trainer actually has: *for this specific training, who exactly was there?* — and, across a season, *which trainings were fully attended and which had gaps?* Reconstructing that today means opening every event's detail sheet one at a time.

## What Changes

- Add a second view ("Matrix") to the statistics page: a member × event grid. Rows are members, columns are events (dated), each cell shows that member's **effective** attendance for that event (✓ ja / ? vielleicht / ✗ nein / – unbekannt), reusing the same effective-status definition as the overview so the two never disagree.
- Rows are sorted by attendance frequency (most `yes` first), so the most reliable members surface at the top.
- Columns are filterable by event type (training / auftritt / event) via checkboxes, defaulting to all types shown.
- New read-only backend endpoint `GET /teams/{teamId}/stats/attendance-matrix` returning the grid for the selected date range (same `from`/`to`/clamping as the overview), gated by the existing `events` RBAC module.

## Capabilities

### New Capabilities
<!-- none -->

### Modified Capabilities
- `attendance-statistics`: extend with a per-member-per-event attendance matrix, computed from the same effective-status definition as the existing quotes.

## Impact

- API: `backend/openapi/openapi.yaml` — new path `/teams/{teamId}/stats/attendance-matrix` (operationId `getAttendanceMatrix`, `x-rbac-module: events`) + schemas `AttendanceMatrix`, `AttendanceMatrixColumn`, `AttendanceMatrixRow`, `AttendanceMatrixStatus`. Regenerate `internal/gen` (`make generate`) and `frontend/src/api` (`make generate-ts`).
- Backend: `internal/stats/{model,repository,service,handler}.go` (+ tests). Reuses `attendance.EffectiveStatusExpr`; no migration (read-only over existing tables).
- Frontend: `types/index.ts`, `api/map.ts` (mapper), `services/serviceLayerReal.ts` (new `stats.attendanceMatrix`), `query/keys.ts`, `pages/hooks/useStatsQueries.ts`, `pages/Stats.tsx` (tab + table + type filter), `i18n/{de,en}.ts`, `mocks/handlers.ts` (MSW), and the relevant tests.
