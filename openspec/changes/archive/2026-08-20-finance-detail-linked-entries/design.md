## Context

`useFinanceOverviewQuery` fetches one `FinanceOverview` per team containing
`transactions`, `penalties`, `assignments`, and `contributions` together
(capped at `maxOverviewRows` = 1000 rows per collection, same cap the
existing `paidAmount` aggregation already lives with). Every finance sheet
component already reads this query. The link itself already exists as data:
`Transaction.contributionId` / `Transaction.penaltyAssignmentId` (nullable
strings, set at creation, mutually exclusive, see `flexible-membership-fees`
design.md). `PenaltyAssignment`/`Contribution` only expose the derived
`paidAmount`, never the transaction id(s) that sum to it.

## Goals

- Every detail view answers "what is this linked to?" without a second
  round-trip or leaving the sheet.
- Reuse the already-loaded `FinanceOverview` — no new endpoint, OpenAPI
  field, or backend query.
- Keep the existing forward-only linking rule (set at creation, immutable
  after) unchanged; this change is display-only, not new editing surface.

## Decisions

- **Derive linked lists client-side by filtering `finances.transactions`.**
  A contribution's/assignment's linked transactions are
  `finances.transactions.filter(t => t.contributionId === c.id)` (or
  `penaltyAssignmentId`). This mirrors the existing `paidAmount` computation
  (same source rows, sum vs. list) and needs no backend change. The known
  limitation is the same one `paidAmount` already has: a team with more than
  `maxOverviewRows` transactions can have older linked rows fall outside the
  window. Accepted for this change, consistent with how the rest of the
  overview already behaves — not a new gap introduced here.
- **`PenaltyAssignSheet` gains a `view` mode instead of a new sheet type.**
  The sheet already renders member/penalty/date/note; view mode reuses that
  layout read-only and appends the linked-transactions list, rather than
  duplicating the layout in a second component. `openPenaltyAssign(a?)`
  takes an optional existing assignment: omitted → `create` (unchanged
  behavior), passed → `view`.
- **`ContribFormSheet` keeps its existing edit form and appends a
  linked-transactions section below the existing fields** — the section is
  additive, not a mode switch, since the contribution form was already
  editable and stays editable.
- **`TxFormSheet` shows a read-only linked-entry line in edit mode**, sourced
  from the same `finances.contributions`/`finances.assignments` collections
  already loaded for the (create-only) `LinkedPaymentPicker`, matched by
  `contributionId`/`penaltyAssignmentId`. No new query.
- **Linked-entry rows are clickable and navigate to that entry's own detail
  sheet** (`app.openTxForm(tx)` for a transaction row; a contribution/penalty
  row from `TxFormSheet` opens `app.openContribForm(c)` /
  `app.openPenaltyAssign(a)` in view mode) so the relationship reads both
  directions without extra chrome.

## Risks

- **List staleness**: `FinanceOverview` isn't real-time; a transaction linked
  by another user seconds ago may not appear until the query refetches.
  Pre-existing limitation of every other value on this page (balances,
  `paidAmount`, list order) — not new here.
