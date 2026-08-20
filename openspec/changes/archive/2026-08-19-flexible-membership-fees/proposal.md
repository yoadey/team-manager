## Why

`contributions` today is a rigid "one row per member per calendar month" model:
every row carries a `month` (`YYYY-MM`), an optional `label` override, and a
boolean `paid`/`open` `status` the treasurer flips by hand. There is no
`createContribution` endpoint at all — the only rows that have ever existed
are seeded demo data (`frontend/src/mocks/db.ts`'s `CO()` helper), six months
deep per member. A real treasurer (Kassenwart) has no way to create a fee in
production today, and the shape assumes membership fees are always monthly
and always the same for everyone, which doesn't hold for clubs charging
one-off fees (tournament fee, kit fee, annual fee with a specific due date)
under a free-text name.

Separately, "paid" is a single boolean with no notion of a partial payment —
a member paying half a fee now in installments has no way to record that,
and the amount actually received is never linked back to the club's own
income bookkeeping (`transactions`): the existing demo data double-books the
same money as both a `contributions` row *and* a separately-entered
`transactions` "Mitgliedsbeiträge Mai" income row, with nothing tying the two
together.

## What Changes

- **Free-text fee definitions, not months.** A contribution row gains a
  required free-text `name` (replacing `month`+`label`) and an optional
  `dueDate`. There is no month field and no automatic monthly generation —
  a recurring fee (e.g. "Mitgliedsbeitrag Januar 2026", "Mitgliedsbeitrag
  Februar 2026") is created by hand, once per instance, exactly as the
  Kassenwart requested.
- **New `POST /teams/{teamId}/finances/contributions`.** Creates a fee for
  one or more members in a single call (`name`, `amount`, optional
  `dueDate`, `userIds`) — the Kassenwart picks the affected members once and
  gets one row per member back, rather than repeating the whole form per
  person.
- **New `DELETE /teams/{teamId}/finances/contributions/{contributionId}`**,
  completing create/update/delete symmetry with the rest of the finances
  module (transactions, penalties) now that contributions are genuinely
  user-created instead of only ever seeded.
- **Partial payments via linked transactions, not a boolean.** A `paid`
  boolean can't represent "half paid." Instead, `transactions` gains an
  optional `contributionId`: booking an income transaction against a
  contribution (in full or in part, and any number of times over time) is
  how a payment gets recorded. A contribution's `paidAmount` is the live
  `SUM()` of its linked income transactions, and `status`
  (`open`/`partial`/`paid`) is derived from comparing `paidAmount` to
  `amount` — never stored, so it can't drift from the transactions that are
  its source of truth. This also fixes the double-booking problem: money
  received now only ever exists once, in `transactions`, optionally tagged
  with which fee it pays.
- **Removed:** `PUT /teams/{teamId}/finances/contributions/{contributionId}/paid`
  (manual paid toggle) — superseded by linking a transaction. `month` and
  `status` columns are dropped from `contributions`.
- **Frontend:** `FinancesContributions` groups by fee name (soonest due date
  first) instead of by month; a new create sheet lets the Kassenwart name a
  fee, set its amount and optional due date, and multi-select the members it
  applies to; recording a payment happens via the existing transaction form,
  now with an optional "applies to this fee" picker.

### Penalty assignments get the same treatment

The same double-booking / manual-toggle problem exists for penalty
assignments (Strafen): `paid` is a hand-flipped boolean, unrelated to
whether the fine was actually booked as income anywhere.

- **`transactions` also gains `penaltyAssignmentId`** (mutually exclusive
  with `contributionId` on the same transaction — a booking pays at most one
  thing). A penalty assignment's `paidAmount` is the live sum of its linked
  income transactions, and `paid` (still a boolean — a fine isn't expected
  to be paid in installments) is derived as `paidAmount >= amount`, exactly
  mirroring contributions. A new assignment defaults to unpaid, as before,
  but now that's simply "no linked transaction yet" rather than a stored
  default.
- **Removed:** `PUT /teams/{teamId}/finances/penalty-assignments/{assignmentId}/paid`
  (manual paid toggle) and the now-fully-unused `SetPaidRequest` schema.
  `penalty_assignments.paid` is dropped from the table.
- **Frontend:** `FinancesPenalties`'s paid/open toggle button becomes a
  static, derived status chip (same treatment `FinancesContributions`
  already got).

### A searchable link picker, not a flat cross-product list

`TxFormSheet`'s "applies to this fee" field was a single flat `<select>`
listing every open *contribution row* — one option per (member × fee)
combination. For a club with 40 members and 20 open fees that's up to 800
options in one dropdown, and adding penalty assignments as a second linkable
target would make it worse, not better. It's replaced with a collapsed-by-
default picker: a Beitrag/Strafe type toggle, a text search (filters by fee
name, penalty label, and member name at once), and a scrollable result list
— the same interaction shape as `ContribCreateSheet`'s member multi-select,
single-select and searchable instead.

## Capabilities

### New Capabilities
- `membership-fees`: free-text, manually-created membership fee definitions
  with optional due dates, multi-member fan-out creation, and paid tracking
  derived from linked income transactions (supporting partial/installment
  payments). Extended to cover penalty assignments' paid tracking the same
  way (single boolean, no partial payments) and the shared searchable link
  picker.

### Modified Capabilities
- `finance-listing`: removes the "Idempotent paid-state changes"
  requirement entirely — neither a contribution's nor a penalty assignment's
  paid state is a settable boolean anymore, so idempotent-toggle semantics
  no longer apply to either.

## Impact

- Database: new migrations `backend/internal/db/migrations/00018_flexible_membership_fees.sql`
  (rename `contributions.label`→`name`, add `due_date`, drop `month`/`status`,
  add `transactions.contribution_id`),
  `00019_flexible_membership_fees_indexes.sql` (`due_date` index, partial
  `contribution_id` index, both `CONCURRENTLY`),
  `00020_penalty_assignment_linked_payment.sql` (add
  `transactions.penalty_assignment_id`, drop `penalty_assignments.paid`),
  `00021_penalty_assignment_linked_payment_indexes.sql` (partial
  `penalty_assignment_id` index, `CONCURRENTLY`).
- API contract: `backend/openapi/openapi.yaml` — `Contribution`,
  `CreateContributionRequest` (new), `UpdateContributionRequest`,
  `ContributionStatus` (gains `partial`), `PenaltyAssignment` (gains
  `paidAmount`), `Transaction`, `CreateTransactionRequest` gain
  `contributionId`/`penaltyAssignmentId`; removes both paid-toggle paths and
  the `SetPaidRequest` schema. Regenerated `internal/gen/api.gen.go`,
  `frontend/src/api/types.gen.ts`, `frontend/src/api/zod.gen.ts`.
- Backend: `internal/finances/{model.go,repository.go,service.go,handler.go}`,
  `internal/server/server.go`.
- Frontend: `features/finances/{types.ts,FinancesPage.tsx}`,
  `features/finances/components/{FinancesContributions.tsx,FinancesPenalties.tsx,ContribFormSheet.tsx,ContribCreateSheet.tsx (new),TxFormSheet.tsx,LinkedPaymentPicker.tsx (new),contribFormSchema.ts,contribCreateFormSchema.ts (new),txFormSchema.ts}`,
  `features/finances/hooks/{useFinanceQueries.ts,useFinanceMutations.ts,useFinanceActions.ts}`,
  `context/AppContext.tsx`, `sheets/index.tsx`, `api/map.ts`,
  `services/serviceLayerReal.ts`, `mocks/{db.ts,handlers.ts}`,
  `services/serviceContract.test.ts`, `i18n/{en.ts,de.ts}`.
