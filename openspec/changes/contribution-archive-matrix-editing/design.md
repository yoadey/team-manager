## Context

`contributions` is one row per (fee-group, member) — a fee-group being the
(name, dueDate) pair `FinancesContributions.groupKey` already uses client-side
to cluster rows into chips. There is no group entity in the DB; "archive this
fee period" and "edit this fee period" are therefore always a fan-out over the
member rows that share a group key, exactly like `CreateContributions` already
fans one create form out to N rows. This change keeps that pattern rather than
inventing a `contribution_groups` table — the group is, and stays, a
client-computed projection over rows that share `(name, dueDate)`.

## Goals

- Archiving/un-archiving a fee period must be one user action, not N.
- Archived rows must disappear from every place that currently shows "what's
  outstanding" (list, matrix, linking picker, `ContribOpen`) without deleting
  data or breaking the transactions already linked to them.
- The contribution matrix and the linking-picker matrix must not drift from
  each other or from the list view — one derivation of paid/partial/open/
  overpaid, reused by all three.
- No new backend endpoint for the contribution matrix: `GetFinanceOverview`
  already returns the full (capped at `maxOverviewRows`) contributions list
  un-paginated, unlike attendance (which needed a dedicated, range-scoped,
  potentially large query). Building the grid client-side avoids a second
  round-trip and a second server-side aggregation to keep in sync.

## Decisions

- **`archived` and `description` on the row, not a new group table.** Simplest
  schema change; a fan-out PATCH (one request per row in the group, `Promise.all`)
  implements "archive this whole period" purely in the frontend action layer
  (`useFinanceActions.archiveContribGroup`), no new backend surface. Same
  reasoning `flexible-membership-fees` used for `CreateContributions`'s
  fan-out create.
- **Archived rows stay in `ListContributions`/`GetOverview`, filtered
  client-side.** An `archived` query param or a second endpoint would let the
  server drop them entirely, but then "show archived / restore" would need a
  *third* way to fetch them. One list, one `archived` boolean per row, is
  simpler and matches how `status` is already computed client-and-server from
  the same source of truth. `CountOpenContributions` (the aggregate used for
  the `ContribOpen` badge) does move server-side filtering: it already scans
  the full table independent of the capped display list, so `AND NOT
  c.archived` there is a one-line change, not a new endpoint.
- **A derived `contributionAmountStatus` helper, not per-component logic.**
  `{ status: 'open'|'partial'|'paid'|'overpaid', displayAmount, excess }`
  computed once in `features/finances/contributionStatus.ts` from
  `(amount, paidAmount)`, used by the list row, the matrix cell, and the
  linking-picker cell. `status` stays `open|partial|paid` at the API layer
  (unchanged wire contract — `overpaid` is a paid contribution, backend-wise,
  since `paidAmount >= amount` is still true); "overpaid" is a frontend-only
  presentation refinement layered on top of the existing `paid` status, not a
  new wire enum value (avoids a spec/API change for something purely about
  how much of the already-known `paidAmount` gets displayed).
- **Matrix columns = fee groups (not individual rows), rows = members with at
  least one row in the filtered (non-archived) set.** Mirrors
  `FinancesContributions`'s existing `groupKey`/`groupLabel` derivation instead
  of introducing a second grouping scheme.
- **Linking-picker matrix lives in a `Dialog`, contribution-tab only.** The
  sheet (`TxFormSheet`) is a narrow bottom sheet; a grid needs more horizontal
  room than fits alongside the amount/title fields. The penalty tab keeps
  `LinkedPaymentPicker`'s existing flat list — fines aren't period-shaped, so a
  grid buys nothing there and reworking it isn't part of this change's ask.
- **Transaction `note` is create+update settable, never list-rendered.**
  Mirrors `PenaltyAssignment.note` (`maxLength: 10000`, optional) exactly,
  including validation (`validate.MaxLen`). `FinancesTransactions`'s row stays
  title/category/date/amount only; the note only surfaces when a treasurer
  opens the row to edit it (`TxFormSheet`), matching "not shown directly in
  the overview."

## Risks

- **Fan-out archive/edit is N requests, not transactional.** A partial failure
  (network drop mid-fan-out) can leave a group half-archived. Accepted: the
  same risk already exists for `CreateContributions`'s failure mode from the
  *client* side (though that one is a single atomic backend transaction) and
  for any multi-row bulk UI action in this codebase; `Promise.allSettled` +
  reporting which rows failed (rather than `Promise.all` silently aborting after
  the first rejection) keeps a partial failure visibly a partial failure
  instead of an ambiguous one, and re-running the action is idempotent (PATCH
  `archived: true` on an already-archived row is a no-op).
- **`description` has no dedicated display slot in the compact list row.**
  Shown in `ContribFormSheet` (the edit form) and as a `title` tooltip on the
  matrix cell / list row when present; not added to the always-visible summary
  card to avoid widening it for what's an optional field most rows won't set.
