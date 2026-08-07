## 1. Database
- [x] 1.1 `00018_flexible_membership_fees.sql`: rename `contributions.label`→`name`
      (backfill NULLs), add `contributions.due_date`, drop `contributions.month`,
      drop `contributions.status`, add `transactions.contribution_id UUID
      REFERENCES contributions(id) ON DELETE SET NULL`
- [x] 1.2 `00019_flexible_membership_fees_indexes.sql` (`NO TRANSACTION`):
      `idx_contributions_team_due_date`, partial `idx_transactions_contribution_id
      WHERE contribution_id IS NOT NULL`, both `CONCURRENTLY`
- [x] 1.3 `make migrate` locally if Docker is available; otherwise rely on CI's
      `backend-migration-rollback` (up→down→up) and `backend-migration-safety`
      gates

## 2. OpenAPI
- [x] 2.1 `Contribution`: `name` (was `label`, now required), `dueDate`
      (nullable date), `paidAmount` (int64, required), `status` gains `partial`;
      drop `month`
- [x] 2.2 `CreateContributionRequest` (new): `name`, `amount`, `dueDate?`,
      `userIds` (`minItems: 1`, `maxItems`); `POST
      /teams/{teamId}/finances/contributions` returns `array<Contribution>`
- [x] 2.3 `UpdateContributionRequest`: `name` (was `label`), `amount`, `dueDate`
- [x] 2.4 `DELETE /teams/{teamId}/finances/contributions/{contributionId}`
- [x] 2.5 Remove `PUT
      /teams/{teamId}/finances/contributions/{contributionId}/paid`
- [x] 2.6 `Transaction` + `CreateTransactionRequest` gain `contributionId`
      (nullable uuid)
- [x] 2.7 `cd backend && make generate` (commit `internal/gen/api.gen.go`)
- [x] 2.8 repo-root `make generate-ts` (commit `frontend/src/api/{types.gen.ts,zod.gen.ts}`)

## 3. Backend: finances module
- [x] 3.1 `model.go`: `ContributionRow` (`Name`, `DueDate *time.Time`,
      `PaidAmount int64`, drop `Month`/`Status`), `ContributionPatch` (`Name`,
      `Amount`, `DueDate`), `TransactionRow`/`TransactionPatch` gain
      `ContributionID *uuid.UUID`
- [x] 3.2 `repository.go`: `ListContributions`/`getContributionByID` join a
      `LATERAL` subquery summing linked income transactions for `PaidAmount`;
      `CreateContributions` (fan-out, one tx, atomic per-row membership
      re-check mirroring `CreateAssignment`); `DeleteContribution`;
      `CountContributions`; `ContributionBelongsToTeam`; `CountOpenContributions`
      counts `paidAmount < amount`; `CreateTransaction` takes
      `contributionID *uuid.UUID`
- [x] 3.3 `service.go`: derive `status` from `paidAmount`/`amount`;
      `CreateContributions` (validates `maxContributionsPerTeam`);
      `DeleteContribution`; `CreateTransaction` validates
      `contributionId` implies `type: income` and belongs to the team;
      `maxContributionsPerTeam` constant + `ErrTooManyContributions`;
      `ErrContributionNotInTeam`
- [x] 3.4 `handler.go` + `internal/server/server.go`: wire
      `CreateContribution`(s)/`DeleteContribution`, remove
      `SetContributionPaid`

## 4. Backend: tests
- [x] 4.1 `repository_test.go`: `CreateContributions` fan-out + atomic
      membership race, `DeleteContribution` unlinks (not cascades) linked
      transactions, `ListContributions`/`getContributionByID` paidAmount sum
      across multiple linked transactions, `CountOpenContributions`
      open+partial
- [x] 4.2 `service_test.go`: status derivation (open/partial/paid),
      `maxContributionsPerTeam`, `CreateTransaction` contributionId/type
      validation
- [x] 4.3 `handler_test.go`: new/changed endpoints, removed paid-toggle route
      returns 404

## 5. Frontend
- [x] 5.1 `features/finances/types.ts`: `Contribution` (`label` now always
      the fee name, `dueDate`, `paidAmount`, `status` incl. `partial`, drop
      `month`), `Transaction.contributionId`; `ContribCreateFormValues`
      (new); `ContribFormValues` gains `dueDate`
- [x] 5.2 `api/map.ts`: `mapContribution`/`mapTransaction` updates
- [x] 5.3 `services/serviceLayerReal.ts`: `finances.createContributions`,
      `finances.deleteContribution`, `updateContribution` gains `dueDate`,
      `addTransaction` gains `contributionId`; remove `setContributionPaid`
- [x] 5.4 `mocks/db.ts` + `mocks/handlers.ts`: seed data without `month`,
      handlers for create/delete contribution + paidAmount computed from
      linked mock transactions
- [x] 5.5 `features/finances/hooks/{useFinanceQueries.ts,useFinanceMutations.ts}`:
      mutations for create/delete contribution; drop set-paid mutation
- [x] 5.6 `features/finances/hooks/useFinanceActions.ts` +
      `context/AppContext.tsx`: `openContribCreate`/`saveContribCreate`/
      `deleteContrib`; drop `setContributionPaid`; `SheetType` gains
      `contribCreate`
- [x] 5.7 `features/finances/components/ContribCreateSheet.tsx` (new): name,
      amount, optional due date, member multi-select (incl. "select all")
- [x] 5.8 `features/finances/components/ContribFormSheet.tsx`: due date field,
      delete action
- [x] 5.9 `features/finances/components/FinancesContributions.tsx`: group by
      fee name (soonest due date first) instead of month; show paid/total
      progress; "new fee" button (`canFin`)
- [x] 5.10 `features/finances/components/TxFormSheet.tsx`: optional "applies
      to this fee" contribution picker (income only)
- [x] 5.11 `sheets/index.tsx` + `features/finances/index.ts`: register
      `contribCreate` sheet + title
- [x] 5.12 `i18n/{en.ts,de.ts}`: new/changed `finances.*` keys

## 6. Frontend: tests
- [x] 6.1 `FinancesContributions.test.tsx`, `ContribFormSheet.test.tsx`,
      new `ContribCreateSheet.test.tsx`
- [x] 6.2 `hooks/useFinanceQueries.test.ts`, `hooks/useFinanceActions.test.ts`
- [x] 6.3 `services/serviceContract.test.ts`, `api/map.test.ts`

## 7. Docs
- [x] 7.1 Check `CLAUDE.md`/`docs/` for stale references to monthly
      contributions

## 9. Database (penalty assignment linking)
- [x] 9.1 `00020_penalty_assignment_linked_payment.sql`: add
      `transactions.penalty_assignment_id UUID REFERENCES
      penalty_assignments(id) ON DELETE SET NULL`, drop
      `penalty_assignments.paid`
- [x] 9.2 `00021_penalty_assignment_linked_payment_indexes.sql`
      (`NO TRANSACTION`): partial `idx_transactions_penalty_assignment_id
      WHERE penalty_assignment_id IS NOT NULL`, `CONCURRENTLY`

## 10. OpenAPI (penalty assignment linking)
- [x] 10.1 `PenaltyAssignment`: add `paidAmount` (int64, required); `paid`
      stays boolean, documented as derived (`paidAmount >= amount`)
- [x] 10.2 `Transaction` + `CreateTransactionRequest`: add
      `penaltyAssignmentId` (nullable uuid), mutually exclusive with
      `contributionId`
- [x] 10.3 Remove `PUT
      /teams/{teamId}/finances/penalty-assignments/{assignmentId}/paid` and
      the now-unused `SetPaidRequest` schema
- [x] 10.4 `cd backend && make generate` / repo-root `make generate-ts`

## 11. Backend: penalty assignment linking + tests
- [x] 11.1 `model.go`: `PenaltyAssignmentRow` drops `Paid bool`, gains
      `PaidAmount int64`; `TransactionRow`/`TransactionPatch` gain
      `PenaltyAssignmentID *uuid.UUID`
- [x] 11.2 `repository.go`: `ListAssignments`/`GetAssignmentByID` join the
      same `LATERAL` sum pattern as contributions; drop
      `SetAssignmentPaid`; `PenaltyAssignmentBelongsToTeam`;
      `CreateTransaction` takes `penaltyAssignmentID *uuid.UUID`
- [x] 11.3 `service.go`: derive `paid` from `paidAmount`/`amount`; drop
      `SetPenaltyPaid`; `CreateTransaction` validates at most one of
      `contributionId`/`penaltyAssignmentId` set, income-only, belongs to
      team
- [x] 11.4 `handler.go` + `internal/server/server.go`: remove
      `SetPenaltyPaid`
- [x] 11.5 `repository_test.go`/`service_test.go`/`handler_test.go`:
      derived-paid coverage, mutual-exclusivity validation, removed route
      returns 404

## 12. Frontend: penalty assignment linking + searchable picker
- [x] 12.1 `features/finances/types.ts`: `PenaltyAssignment.paidAmount`;
      `Transaction.penaltyAssignmentId`
- [x] 12.2 `api/map.ts`, `services/serviceLayerReal.ts`: map/pass through
      new fields; remove `setPenaltyPaid`
- [x] 12.3 `mocks/db.ts` + `mocks/handlers.ts`: derive penalty `paid` from
      linked mock transactions; remove the paid-toggle handler; validate
      mutual exclusivity in the create-transaction handler
- [x] 12.4 `features/finances/hooks/{useFinanceMutations.ts,useFinanceActions.ts}`
      + `context/AppContext.tsx`: remove `setPenaltyPaid`
- [x] 12.5 `features/finances/components/FinancesPenalties.tsx`: replace the
      paid/open toggle button with a static derived chip (mirrors
      `FinancesContributions`)
- [x] 12.6 New `features/finances/components/LinkedPaymentPicker.tsx`:
      collapsed-by-default, Beitrag/Strafe type toggle, text search over
      fee/penalty label + member name, scrollable result list; used by
      `TxFormSheet.tsx` in place of the flat `<select>`
- [x] 12.7 `i18n/{en.ts,de.ts}`: new/changed `finances.*` keys
- [x] 12.8 Tests: `FinancesPenalties.test.tsx`, new
      `LinkedPaymentPicker.test.tsx`, `TxFormSheet.test.tsx` updates,
      `useFinanceActions.test.ts`, `serviceContract.test.ts`

## 13. Verification
- [x] 13.1 `openspec validate flexible-membership-fees --strict`
- [x] 13.2 `cd backend && make generate` / repo-root `make generate-ts` — no diff
- [x] 13.3 `cd backend && make lint`
- [x] 13.4 `cd backend && make test`
- [x] 13.5 `cd frontend && npm run lint && npm run typecheck && npm test && npm run build`
