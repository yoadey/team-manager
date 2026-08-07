## Why

`Finanzen` already links records to each other — a transaction can pay a
contribution (`Transaction.contributionId`) or a penalty assignment
(`Transaction.penaltyAssignmentId`) — but none of the three detail views
surface that link once it exists:

1. **`TxFormSheet` (Umsätze detail).** `LinkedPaymentPicker`, the only UI that
   ever shows a contribution/penalty next to a transaction, is gated on
   `!edit && type === 'income'` (`TxFormSheet.tsx:90`). Reopening an existing,
   already-linked transaction for editing shows no trace of what it pays —
   the `contributionId`/`penaltyAssignmentId` are loaded into the form but
   never rendered.
2. **`FinancesPenalties` (Strafen).** Assignment rows aren't clickable at all
   — there is no detail view for a penalty assignment, so there's nowhere to
   show which transaction(s) paid it. Only an aggregate paid/open chip is
   shown (`FinancesPenalties.tsx:103-107`).
3. **`ContribFormSheet` (Beiträge detail).** The edit sheet shows the
   contribution's own fields (label, amount, description, due date) but never
   the transaction(s) that paid it — same aggregate-only gap as penalties.

A Kassenwart reconciling the books has no way, from any of these three detail
views, to answer "which entries is this linked to?" without leaving the sheet
and cross-referencing the flat Umsätze list by eye.

## What Changes

- **`TxFormSheet`**: when editing a transaction that has `contributionId` or
  `penaltyAssignmentId` set, show a read-only "linked to" line naming the
  contribution/penalty it pays (label + amount), instead of showing nothing.
- **`FinancesPenalties`**: assignment rows become clickable, opening a new
  read-only detail view (`PenaltyAssignSheet` gains a `view` mode) that lists
  every transaction linked to that assignment (title, date, amount).
- **`ContribFormSheet`**: add a "linked transactions" section listing every
  transaction linked to that contribution (title, date, amount).
- All three linked-entry lists are derived client-side from data the
  `FinanceOverview` query already loads (`transactions[].contributionId` /
  `.penaltyAssignmentId`) — no new backend endpoint or OpenAPI field, since a
  contribution/penalty can have 0..n paying transactions but a transaction
  links to at most one contribution/penalty, and the overview already
  fetches all four collections together.
- Clicking a linked entry in any of the three lists navigates to that
  entry's own detail sheet (a linked transaction opens `TxFormSheet` in edit
  mode; a linked contribution/penalty from a transaction's detail opens its
  own detail sheet), so the relationship is navigable both directions.

## Capabilities Impacted

- `finance-listing` (modified): the three finance detail views gain
  linked-entry visibility.

## Impact

- Frontend only: `frontend/src/features/finances/components/TxFormSheet.tsx`,
  `FinancesPenalties.tsx`, `PenaltyAssignSheet.tsx`, `ContribFormSheet.tsx`,
  `hooks/useFinanceActions.ts` (`openPenaltyAssign` gains an optional
  assignment argument for view mode), `sheets/index.tsx` (view-mode sheet
  title), i18n (`de.ts`/`en.ts`) for new labels.
- No backend, migration, or OpenAPI changes — `FinanceOverview` already
  carries every field these views need.
