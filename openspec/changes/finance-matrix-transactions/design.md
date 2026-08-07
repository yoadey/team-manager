## Context

`FinancesContributions.tsx` currently toggles between a `'list'` and `'matrix'` local view state, defaulting to `'list'`. `ContribMatrixView.tsx` renders a static (non-interactive) member x fee-group grid. `LinkedPaymentPicker.tsx` (used from `TxFormSheet.tsx`) starts collapsed behind a "Mit Beitrag oder Strafe verknüpfen (optional)" toggle button; once expanded it shows a "Beiträge"/"Strafen" kind switch, and only the "Beiträge" side opens a dialog (`ContribLinkMatrixDialog.tsx`) — the "Strafen" side is an inline search+list within the same expanded panel. `TxFormSheet.tsx` has no date field even though `CreateTransactionRequest`/`UpdateTransactionRequest` already accept one (see `finance-listing`'s "Client-settable transaction date" requirement).

## Goals / Non-Goals

- Goal: matrix-first contributions tab, with the matrix itself usable for booking/inspecting, not just viewing.
- Goal: bring the transaction form's linking UI to the same one-tap-then-popup pattern for both fee and penalty linking.
- Goal: expose the already-supported transaction date on the form.
- Non-Goal: changing the OpenAPI contract — `date` and `category` are already client-settable/filterable client-side; no backend change.
- Non-Goal: server-side category filtering/pagination — the transactions list rendered here is the overview's capped display list (see `finance-listing`), so the category filter is a client-side filter over what's already fetched, consistent with how `contribGroup` filtering already works.

## Decisions

### Matrix cell click routes by paid state, not by "does a row exist"
Every populated cell already corresponds to an existing `Contribution` row (matrix rows are built from the contributions the treasurer created for those members). "Kein Beitrag erfasst" in the request means no *payment* has been booked yet, i.e. `paidAmount === 0`. Empty cells (no `Contribution` row for that member in that fee group at all) stay non-interactive, unchanged from today — there's no contribution id to act on.

### Reuse `ContribFormSheet` as the matrix's "view existing bookings" popup
`ContribFormSheet` already renders `LinkedTransactionsList` (transactions linked to that contribution, clickable through to each one) alongside the edit fields. Rather than build a second read-only popup, a matrix cell with `paidAmount > 0` opens the same sheet via `app.openContribForm(c)`. This also satisfies the separate "click a matrix entry to edit/link" ask as the same code path.

### A new `openTxFormForContribution` action, not an overloaded `openTxForm`
`openTxForm(tx?)` toggles between create/edit based on whether a transaction is passed. Booking a fresh payment from the matrix is a third case: create-mode, but pre-linked and pre-titled. A dedicated `useFinanceActions` callback keeps `openTxForm`'s signature simple and makes the pre-fill (title = contribution label, amount = amount still owed, `contributionId`, `type: 'income'`, today's date) explicit at the call site.

### `PenaltyLinkDialog` mirrors `ContribLinkMatrixDialog`'s dialog shell
The penalty side becomes a `Dialog` with the same search input, listing open assignments (already-filtered, client-side, matching `LinkedPaymentPicker`'s existing inline list), instead of an inline expanding panel. This is a new small component rather than generalizing `ContribLinkMatrixDialog`, since one is a grid (member x fee) and the other a flat list (member + fine) — forcing a shared abstraction would cost more than the duplication it avoids.

### Compacting `ContribLinkMatrixDialog` only, not `ContribMatrixView`
Only the transaction-form picker dialog gets the compact/no-photo/no-inner-radius treatment the request asks for; the contributions tab's own matrix (`ContribMatrixView`) keeps its current denser-but-labeled styling (with avatars) since it's a primary view, not a transient picker.

## Risks / Trade-offs

- Changing the contributions tab's default view will shift several existing list-view tests that relied on the implicit default; they're updated to switch to list view first where they test list-only behavior.
- A client-side category filter over the overview's capped transaction list won't reflect categories that only exist on older, paged-out transactions — acceptable since it mirrors the existing `contribGroup` filter's same scoping, and the full history stays reachable via pagination for anyone who needs it.
