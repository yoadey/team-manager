## Why

The finance "Beiträge" (contributions) tab defaults to a flat per-fee-group list, even though the matrix view (member x fee-group grid) gives the treasurer the fastest overview and is where most people land first. The matrix is also read-only today — there's no way to book or inspect a payment from it, forcing a detour through the list view. Separately, the contribution-linking matrix embedded in the "record a new transaction" flow carries a member photo and generous spacing that make it cramped on a phone, and linking a transaction to a fee or penalty is a two-step disclosure (an inner "Beiträge"/"Strafen" toggle) instead of two direct actions. Transactions also lack a way to set the booking date from the UI (the API already accepts one) and can't be filtered by category once a team accumulates many.

## What Changes

- The contributions tab's matrix view becomes the default (was: list).
- Clicking a populated matrix cell opens: the new-transaction form pre-linked to that contribution when it has no payment yet (`paidAmount === 0`), or the contribution's detail/edit sheet (which already lists linked transactions) when it has at least one.
- `ContribLinkMatrixDialog` (the picker matrix used from the transaction form) becomes more compact: no member photo, tighter column padding, no rounded corners on the interior selection cells.
- `LinkedPaymentPicker` (the transaction form's "Verknüpfen mit" section) drops its collapsed two-step disclosure in favor of a heading plus two direct buttons ("Beiträge" / "Strafen"), each opening its own popup (matrix dialog for fees, a new list-style dialog for penalties) — mirroring the always-visible "Betrag" field above it instead of hiding behind a toggle.
- The transaction form (`TxFormSheet`) gains a date field, defaulting to today on create and to the transaction's existing date on edit.
- The transactions ("Umsätze") list gains a category filter.

## Capabilities

### New Capabilities
- `finance-matrix-transactions`: contribution-matrix-as-default-view, click-to-book/inspect from the matrix, the compacted link-picker matrix, single-step fee/penalty linking, a transaction date field, and category filtering on the transaction list.

## Impact

- Frontend only — the backend already accepts/returns `date` on transactions (see `finance-listing`'s "Client-settable transaction date" requirement) and category is already a plain string field, so no OpenAPI/backend change is needed.
- `frontend/src/features/finances/components/FinancesContributions.tsx` (default view), `ContribMatrixView.tsx` (cell click), `ContribLinkMatrixDialog.tsx` (compact styling), `LinkedPaymentPicker.tsx` (single-step buttons, new `PenaltyLinkDialog.tsx`), `TxFormSheet.tsx` + `txFormSchema.ts` (date field), `FinancesTransactions.tsx` (category filter), `hooks/useFinanceActions.ts` / `hooks/useFinanceMutations.ts` (date plumbing, new "open tx form linked to a contribution" action), plus i18n keys (de/en) and tests for all of the above.
- No API/spec change beyond this new capability; CI: frontend lint/typecheck/test/build + bundle budget.
