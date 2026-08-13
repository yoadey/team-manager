## 1. Database
- [ ] 1.1 `00028_absence_stats_relevance.sql`: `ALTER TABLE absences ADD COLUMN
      not_relevant_for_stats boolean NOT NULL DEFAULT false;` and
      `ALTER TABLE absences ADD COLUMN not_relevant_set_by uuid REFERENCES users(id);`
- [ ] 1.2 `make migrate` locally if Docker is available; otherwise rely on CI's
      `backend-migration-rollback`/`backend-migration-safety` gates

## 2. OpenAPI
- [ ] 2.1 `Absence`: add `notRelevantForStats: boolean`
- [ ] 2.2 New path `/teams/{teamId}/absences/{absenceId}/stats-relevance`
      (`PATCH`, body `{ notRelevantForStats: boolean }`, `x-rbac-module: public`)
- [ ] 2.3 `cd backend && make generate` (commit `internal/gen/api.gen.go`)
- [ ] 2.4 repo-root `make generate-ts` (commit `frontend/src/api/types.gen.ts`)

## 3. Backend: absences module
- [ ] 3.1 `model.go`: `NotRelevantForStats bool`, `NotRelevantSetBy *uuid.UUID`
      on `AbsenceRow`
- [ ] 3.2 `repository.go`: `SetStatsRelevance(ctx, absenceID, teamID uuid.UUID,
      notRelevant bool, setBy uuid.UUID) error` (team-scoped update)
- [ ] 3.3 `service.go`: `SetStatsRelevance` — locate `teams`'s permission-lookup
      dependency (the interface `notifications.Service` already depends on,
      `GetPermissions(ctx, teamID, userID) (teams.PermissionsJSON, error)`) and
      add a local write-permission helper for the `events` module mirroring
      `notifications.HasReadAccess`/`middleware/authz.go`'s `hasWritePermission`
      fail-closed logic; when `absence.UserID != caller.ID`, require it
- [ ] 3.4 `handler.go` + `internal/server/server.go`: wire the new
      `PATCH .../stats-relevance` route

## 4. Backend: attendance/stats
- [ ] 4.1 `internal/attendance/sql.go`: `AbsenceCoversExpr` gains an
      `AND not_relevant_for_stats = false` variant (or a second constant) used
      by the new `'excluded'` branch; `EffectiveStatusExpr` gains the
      `'excluded'` branch, evaluated before the opt-out fallback
- [ ] 4.2 Audit every current caller of `EffectiveStatusExpr`/`AbsenceCoversExpr`
      outside `stats/repository.go` (e.g. event attendance-summary queries) and
      confirm `'excluded'` renders sensibly there (expected: same treatment as
      no response)
- [ ] 4.3 `stats/repository.go`: confirm `'excluded'` is naturally dropped by
      the existing `FILTER (WHERE eff IN ('yes','no','maybe'))` allowlists in
      `MemberStats`, `SingleMemberStats`, and that `AbsenceStats` skips
      `'excluded'` rows

## 5. Backend: tests
- [ ] 5.1 `absences/service_test.go`: self can always set the flag; a non-owner
      without `events:write` is rejected; a non-owner with `events:write`
      succeeds
- [ ] 5.2 `attendance`/`stats` tests: a not-relevant absence's covered date is
      excluded from `counted` entirely (not counted as "no"); a normal absence
      still counts as "no" (regression)

## 6. Frontend
- [ ] 6.1 `features/events/components/EventAbsences.tsx`: add a "nicht
      relevant" checkbox/chip, visible when `isMe` or the viewer holds
      `events:write`
- [ ] 6.2 New mutation hook alongside `useAbsence{Queries,Mutations,Actions}.ts`
      for the `PATCH .../stats-relevance` call
- [ ] 6.3 `api/map.ts`, `services/serviceLayerReal.ts`,
      `mocks/{db.ts,handlers.ts}`: wire the new field/endpoint through
- [ ] 6.4 `i18n/{de.ts,en.ts}`: label + helper text

## 7. Verification
- [ ] 7.1 `cd backend && make test && make lint`
- [ ] 7.2 `cd frontend && npm run typecheck && npm run lint && npm test`
- [ ] 7.3 `make generate && make generate-ts` produce no drift
- [ ] 7.4 Manual: flag a colleague's absence as not-relevant with and without
      `events:write`, confirm the permission boundary and the resulting quote
      change
