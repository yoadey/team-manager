## Why

`Finanzen -> Beiträge` has three usability gaps a real Kassenwart hits immediately:

1. **No description, no archive.** A contribution can only be edited by opening one
   member's row at a time and changing name/amount/due date (`ContribFormSheet` +
   `PATCH .../contributions/{id}`) — there's no longer free-text description field,
   and once a fee period is no longer relevant (last season's dues, a cancelled
   event fee) there's no way to get it out of the way. It stays in the group chip
   list forever, and `ContribOpen`/the matrix (below) keeps counting it.
2. **The list view doesn't scale.** `FinancesContributions` shows one fee-group at a
   time (a chip row to switch groups, then a flat member list underneath). Seeing
   "who across all fee periods still owes money" means clicking through every
   chip. The stats page already solved exactly this shape (member × event grid,
   `openspec/changes/archive/2026-08-02-attendance-matrix-view`) — contributions
   need the same member × fee-group grid.
3. **Overpayment is invisible.** `contributionStatus` derives `open|partial|paid`
   from `paidAmount` vs `amount`, which is correct, but the UI then displays
   `fmtMoney(c.amount)` for a `paid` row instead of the actual `paidAmount` —
   a member who paid €60 against a €50 fee shows the same "50,00 €" as a member
   who paid exactly €50. The €10 credit is silently dropped from the display,
   even though the backend already has the real number.

Two adjacent rough edges the same area of the app has:

4. **The transaction-linking picker doesn't fit the mental model.** Booking an
   income transaction against a contribution uses `LinkedPaymentPicker`'s flat,
   searchable list (`name · label — amount`). For a club with e.g. 30 members ×
   4 open fee periods that's up to 120 rows to search through one at a time, when
   the natural picture is the same member × fee-group grid as the overview.
5. **No note field on transactions.** `PenaltyAssignment` already carries a
   free-text `note` (`internal/db/migrations/00008_penalty_assignment_note.sql`);
   `transactions` never got the same field, so there's nowhere to record
   "Bar erhalten von X, Quittung Nr. 12" against a booking without stuffing it
   into `title`/`category`, which are both shown directly in the transaction list.

## What Changes

- **`Contribution` gains `description` (optional free text, ≤2000 chars) and
  `archived` (bool, default `false`).** Both are patchable via the existing
  `PATCH /teams/{teamId}/finances/contributions/{contributionId}`.
  `ContribFormSheet` gains a description textarea and an "Archivieren" action
  alongside the existing delete action.
- **Archived contributions are hidden by default**, everywhere: the group chip
  list, the summary card, the row list, the new matrix view, the linked-payment
  picker, and `ContribOpen`/the finance overview's open-contribution count. A
  "Archivierte Beiträge anzeigen" toggle in `FinancesContributions` reveals them
  (with an "Archiviert" chip) so a treasurer can restore one (un-archive) if it
  was archived by mistake. A "Fälligkeitsperiode archivieren" bulk action on the
  group summary card archives every row in the selected group in one action
  (fan-out over the existing per-row PATCH, mirroring how `CreateContributions`
  already fans a single form out to one row per member) — archiving one whole
  fee period is the actual use case, not archiving members one at a time.
- **New matrix view for `FinancesContributions`.** A "Liste"/"Matrix" tab switch
  (matching `Stats`'s `quota`/`matrix` tabs); the matrix is a member × fee-group
  grid computed client-side from the already-fetched `FinanceOverview.contributions`
  (no new backend endpoint — unlike the attendance matrix, this data is already
  fully loaded and unpaginated). Cells show paid (✓ green) / partial (◐ primary) /
  open (🕐 orange) / **overpaid (＋ teal, new)** with the exact amount in the
  cell's `aria-label`/tooltip.
- **Overpayment becomes visible.** Both the list and matrix views show the actual
  `paidAmount` (not the capped `amount`) whenever `paidAmount > amount`, with a
  distinct "X,XX € zu viel bezahlt" treatment instead of collapsing into the same
  display as an exact payment.
- **`LinkedPaymentPicker`'s contribution tab becomes a matrix-with-checkboxes.**
  Selecting which fee a new transaction pays opens the same member × fee-group
  grid (open cells only; archived and fully-paid fee periods excluded, as today)
  in a popup dialog (space in the transaction sheet is tight for a grid); each
  open cell is a checkbox-styled single-select control showing the still-owed
  amount. The penalty ("Strafen") tab is unchanged — it stays a flat searchable
  list, since fines aren't naturally arranged by period.
- **`transactions` gains an optional `note`** (free text, ≤10000 chars, same
  limit as `PenaltyAssignment.note`). Settable on create and update. Shown only
  in `TxFormSheet` (the create/edit form) — never in `FinancesTransactions`'s
  list rows, matching the ask that it not appear directly in the overview.

## Capabilities

### Modified Capabilities

- `membership-fees`: contributions gain `description` and `archived`; archived
  contributions are excluded from display, matrix, linking, and the open-count
  aggregate by default.
- `finance-listing`: new matrix view for contributions; overpayment surfaced in
  both list and matrix; `LinkedPaymentPicker`'s contribution tab becomes a
  matrix picker; transactions gain an optional, list-hidden `note`.

## Impact

- **DB**: `backend/internal/db/migrations/00022_contribution_archive_description.sql`
  (+`00023_..._indexes.sql` for a partial index backing the "exclude archived"
  filter), `00024_transaction_note.sql`.
- **API**: `backend/openapi/openapi.yaml` — `Contribution` gains `description`,
  `archived`; `UpdateContributionRequest`/`CreateContributionRequest` gain
  `description`; `UpdateContributionRequest` gains `archived`; `Transaction`/
  `CreateTransactionRequest`/`UpdateTransactionRequest` gain `note`. Regenerate
  `internal/gen` (`make generate`) and `frontend/src/api` (`make generate-ts`).
- **Backend**: `internal/finances/{model,repository,service,handler}.go` (+ tests):
  `ContributionRow`/`ContributionPatch` gain `Description`/`Archived`;
  `CountOpenContributions` excludes archived rows; `TransactionRow`/
  `TransactionPatch` gain `Note`.
- **Frontend**: `features/finances/types.ts`, `api/map.ts`,
  `services/serviceLayerReal.ts`, `mocks/{db,handlers}.ts`,
  `features/finances/components/{ContribFormSheet,FinancesContributions,
  LinkedPaymentPicker,TxFormSheet}.tsx` (+ new `ContribMatrixView.tsx`,
  `ContribLinkMatrixDialog.tsx`), `hooks/{useFinanceQueries,useFinanceMutations,
  useFinanceActions}.ts`, `i18n/{de,en}.ts`, and the relevant tests.
