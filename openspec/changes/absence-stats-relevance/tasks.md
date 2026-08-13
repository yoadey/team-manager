## 1. Database
- [x] 1.1 `00028_absence_stats_relevance.sql`: `ALTER TABLE absences ADD COLUMN
      not_relevant_for_stats boolean NOT NULL DEFAULT false;` and
      `ALTER TABLE absences ADD COLUMN not_relevant_set_by uuid REFERENCES users(id);`
- [ ] 1.2 `make migrate` locally if Docker is available; otherwise rely on CI's
      `backend-migration-rollback`/`backend-migration-safety` gates (no Docker
      in this dev sandbox -- deferred to CI)

## 2. OpenAPI
- [x] 2.1 `Absence`: add `notRelevantForStats: boolean`
- [x] 2.2 New path `/teams/{teamId}/absences/{absenceId}/stats-relevance`
      (`PATCH`, body `{ notRelevantForStats: boolean }`, `x-rbac-module: public`)
- [x] 2.3 `cd backend && make generate` (commit `internal/gen/api.gen.go`)
- [x] 2.4 repo-root `make generate-ts` (commit `frontend/src/api/types.gen.ts`)

## 3. Backend: absences module
- [x] 3.1 `model.go`: `NotRelevantForStats bool`, `NotRelevantSetBy *uuid.UUID`
      on `AbsenceRow`
- [x] 3.2 `repository.go`: `GetOwner(ctx, id, teamID uuid.UUID) (uuid.UUID, error)`
      (used by the service to decide self-vs-other before writing) and
      `SetStatsRelevance(ctx, id, teamID uuid.UUID, notRelevant bool, setBy
      uuid.UUID) (*AbsenceRow, error)` (team-scoped update, unconditional --
      the service is responsible for authorizing the caller first)
- [x] 3.3 `service.go`: `permChecker` interface (`GetPermissions`, mirroring
      `notifications.Service`'s dependency) + `hasEventsWritePermission`
      local helper; `SetStatsRelevance` reads the owner via `GetOwner`, and
      when `ownerID != callerID` requires `hasEventsWritePermission`, else
      `ErrForbiddenStatsRelevance`
- [x] 3.4 `handler.go` + `internal/server/server.go`: wire the new
      `PATCH .../stats-relevance` route (server.go needs an explicit
      delegation method, not just struct embedding -- `Absences` is a named
      field, not anonymously embedded)

## 4. Backend: attendance/stats
- [x] 4.1 `internal/attendance/sql.go`: new `NotRelevantAbsenceCoversExpr`
      constant, added alongside (not replacing) `EffectiveStatusExpr` --
      `EffectiveStatusExpr` itself is left unchanged since it's shared with
      `internal/events`' attendance-summary queries, where a 6th status value
      would break the Yes+No+Maybe+Pending+NotNominated == Total invariant
      (see design.md's revised "Decisions")
- [x] 4.2 `stats/repository.go`: `MemberStats`, `EventStats`, `AbsenceStats`,
      `SingleMemberStats` each wrap `EffectiveStatusExpr` in
      `CASE WHEN a.status IS NULL AND NotRelevantAbsenceCoversExpr THEN
      'excluded' ELSE EffectiveStatusExpr END` -- `'excluded'` never leaves
      SQL as a wire value in these four queries, so introducing it is safe
- [x] 4.3 `stats/repository.go`'s `matrixCells`: same wrapping, but mapping to
      `'pending'` instead of `'excluded'`, since `MatrixCellRow.Eff` IS cast
      directly to the wire `gen.AttendanceStatus` enum (`yes/no/maybe/pending/
      not_nominated`, no "excluded" member) -- already correctly excluded from
      the per-row Yes/Counted aggregation in `stats/service.go`

## 5. Backend: tests
- [x] 5.1 `absences/service_test.go`: self can always set the flag; a non-owner
      without `events:write` is rejected; a non-owner with `events:write`
      succeeds; not-found absence returns an error
- [x] 5.2 `stats/repository_test.go`: a not-relevant absence's covered date is
      excluded from `counted` entirely (not counted as "no") in
      `MemberStats`/`EventStats`/`SingleMemberStats`, absent from
      `AbsenceStats`'s rows, and the matrix cell reads `'pending'`; a normal
      absence still counts as "no" (regression)

## 6. Frontend
- [x] 6.1 `features/events/components/EventAbsences.tsx`: add a "nicht
      relevant" checkbox/chip, visible when `isMe` or the viewer holds
      `events:write`
- [x] 6.2 New mutation hook alongside `useAbsence{Queries,Mutations,Actions}.ts`
      for the `PATCH .../stats-relevance` call
- [x] 6.3 `api/map.ts`, `services/serviceLayerReal.ts`,
      `mocks/{db.ts,handlers.ts}`: wire the new field/endpoint through
- [x] 6.4 `i18n/{de.ts,en.ts}`: label + helper text

## 7. Verification
- [x] 7.1 `cd backend && make test && make lint` (integration tests skip: no
      Docker in this sandbox; unit tests + lint green)
- [x] 7.2 `cd frontend && npm run typecheck && npm run lint && npm test`
- [x] 7.3 `make generate && make generate-ts` produce no drift
- [ ] 7.4 Manual: flag a colleague's absence as not-relevant with and without
      `events:write`, confirm the permission boundary and the resulting quote
      change
