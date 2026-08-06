## 1. API contract

- [x] 1.1 Add `/teams/{teamId}/stats/attendance-matrix` GET to `openapi.yaml` (operationId `getAttendanceMatrix`, `x-rbac-module: events`, `from`/`to` query params like the overview)
- [x] 1.2 Add schemas `AttendanceMatrixColumn`, `AttendanceMatrixRow` (cells as `additionalProperties` map), `AttendanceMatrix`. Cell type reuses the existing `AttendanceStatus` enum rather than a new one — a colliding 4-value enum forces oapi-codegen to globally prefix every enum constant (breaks hand-written `gen.NotNominated` etc.)
- [x] 1.3 `cd backend && make generate`; repo-root `make generate-ts`; commit regenerated `internal/gen` + `frontend/src/api`

## 2. Backend

- [x] 2.1 `stats/model.go`: `MatrixColumnRow`, `MatrixCellRow`
- [x] 2.2 `stats/repository.go`: `AttendanceMatrix(ctx, teamID, from, to)` — own RepeatableRead/ReadOnly tx, columns query + cells query reusing `attendance.EffectiveStatusExpr`
- [x] 2.3 `stats/service.go`: `GetAttendanceMatrix` — default/clamp range, assemble rows (cells map, yes/counted), sort rows by yes desc (stable → name), columns by date
- [x] 2.4 `stats/handler.go`: `GetAttendanceMatrix` — auth guard, map service result to gen response; wired through `server.go`
- [x] 2.5 Extend `statsRepo` + `statsService` interfaces and their unit-test mocks with the new method

## 3. Frontend

- [x] 3.1 `types/index.ts`: `AttendanceCellStatus`, `AttendanceMatrixColumn`, `AttendanceMatrixRow`, `AttendanceMatrix`
- [x] 3.2 `api/map.ts`: `mapAttendanceMatrix` (folds not_nominated → pending)
- [x] 3.3 `services/serviceLayerReal.ts`: `stats.attendanceMatrix(teamId, range)`
- [x] 3.4 `query/keys.ts` + `pages/hooks/useStatsQueries.ts`: `statsMatrix` key + `useAttendanceMatrixQuery` (lazy `enabled` gate on the open tab)
- [x] 3.5 `pages/Stats.tsx`: Übersicht/Matrix tab toggle; matrix table (sticky member column, per-event columns, yes-total column); event-type filter checkboxes; cell glyphs ✓/?/✗/–
- [x] 3.6 `i18n/{de,en}.ts`: tab labels, matrix headers, type filter, cell aria labels
- [x] 3.7 `mocks/handlers.ts`: MSW handler for the matrix endpoint (effective status, same ordering)

## 4. Tests

- [x] 4.1 Backend: service unit tests (assembly, sort, row/yes reconciliation, nil-event placeholder, default range) + handler auth/success + repository integration (opt-out default, covering absence, roster-driven, column/row ordering)
- [x] 4.2 Frontend: `Stats.test.tsx` (tab switch, grid render, type filter, loading), `useStatsQueries.test.ts` (matrix hook enable/fetch), `map.test.ts` (mapper), `serviceLayerReal.test.ts` (method forwards range + maps), `handlers.test.ts` (MSW end-to-end grid); a11y hook mock updated

## 5. Verification

- [x] 5.1 `make generate` + `make generate-ts` idempotent — regeneration produces only additive diffs (CI `backend-openapi-drift`)
- [x] 5.2 Backend: `go test ./internal/stats/...` green (unit + integration), `golangci-lint run` clean on changed packages
- [x] 5.3 Frontend: `npm run lint` (0 errors), `npm run typecheck`, affected test files green, `npm run build` + `check:bundle` within budget (254.4 KB total)
- [x] 5.4 `openspec validate attendance-matrix-view --strict` passes
