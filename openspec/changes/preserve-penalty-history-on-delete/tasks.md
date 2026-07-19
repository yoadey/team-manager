## 1. Migration
- [ ] 1.1 Add `00027_penalty_fk_set_null.sql`: drop the `penalty_assignments.penalty_id` FK, make the column nullable, re-add the FK as `ON DELETE SET NULL`; write a safe down migration
- [ ] 1.2 Run `cd backend && make generate` (sqlc picks up the now-nullable column); commit generated output

## 2. Repository/service
- [ ] 2.1 Audit read/aggregate paths for `penalty_id` dereferences; treat it as optional, display from the snapshot `label`/`amount`
- [ ] 2.2 Confirm the finance overview open/paid sums come from assignment snapshots, unaffected by a null `penalty_id`

## 3. Tests
- [ ] 3.1 Test: deleting a penalty with paid + unpaid assignments keeps all assignments with their snapshot values
- [ ] 3.2 Test: an assignment whose penalty was deleted still lists correctly (null catalog reference)

## 4. Verification
- [ ] 4.1 `make test` green; migration up→down→up green; migration-safety lint green
- [ ] 4.2 `make lint` + coverage gate green; `make generate` produces no diff
