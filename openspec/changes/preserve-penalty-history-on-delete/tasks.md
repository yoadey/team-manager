## 1. Migration
- [x] 1.1 Add `00027_penalty_fk_set_null.sql`: `DROP NOT NULL` on `penalty_id`, drop the FK, re-add it `ON DELETE SET NULL` (NOT VALID + VALIDATE, per migration-safety rules); safe down migration (restores CASCADE, leaves column nullable to avoid a raw SET NOT NULL scan)
- [x] 1.2 `make generate` — sqlc regenerated `penalty_assignments.PenaltyID` as `*uuid.UUID`; committed generated output

## 2. Repository/service
- [x] 2.1 `finances/model.go` `PenaltyID uuid.UUID` → `*uuid.UUID`; scan sites already scan by address (null-tolerant); `toGenAssignment` passes the pointer through (openapi_types.UUID is an alias for uuid.UUID)
- [x] 2.2 Confirmed no read path joins `penalties` for display (snapshot columns only) and no `.PenaltyID` dereference exists; overview sums come from assignment snapshots, unaffected by a null penalty_id
- [x] 2.3 OpenAPI `PenaltyAssignment.penaltyId` made `nullable: true` and dropped from `required` → generated as `*openapi_types.UUID` / `penaltyId?: string | null`; frontend `map.ts` uses `?? null`; domain `PenaltyAssignment.penaltyId` widened to `string | null`

## 3. Tests
- [x] 3.1 `TestFinancesRepository_DeletePenalty_PreservesAssignments`: delete a penalty with paid + unpaid assignments → both survive with null penalty_id and intact snapshot label/amount
- [x] 3.2 Updated `service_test.go` mock rows to the pointer field; existing delete/toggle tests stay green

## 4. Verification
- [x] 4.1 `go test ./... -short` green; migration up→down→up + migration-safety lint confirmed by CI (no Docker locally)
- [x] 4.2 `golangci-lint run ./...` 0 issues; `make generate` + `make generate-ts` committed with no further drift
- [x] 4.3 Frontend `typecheck` + finance/map tests (146) green; `npm run lint` 0 errors
