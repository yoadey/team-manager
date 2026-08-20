## Context

Contributions are one row per (fee-group, member); the fee-group is a
client-computed `(label, dueDate)` projection (`FinancesContributions.groupKey`),
not a database entity (see `contribution-archive-matrix-editing`'s design.md,
which made the same call for the existing group-archive action). This change
extends that same "group is a client-side fan-out, not a new table" decision
to editing, and narrows the per-row view down to what it can safely show
without letting a row drift out of its group.

## Goals

- A member's own contribution row can never be edited into inconsistency
  with its group (different label/dueDate splitting the group; different
  amount than siblings) via the single-row detail view, because that view
  no longer edits those fields at all.
- Editing name/amount/description/due date for a whole fee period is one
  user action, mirroring `archiveContribGroup`'s existing fan-out pattern
  exactly — not a new backend endpoint, not a `contribution_groups` table.
- The detail view is reachable through exactly one path regardless of
  payment state (list row action, matrix cell) instead of the matrix
  branching between two different sheets depending on `paidAmount`.
- Booking a payment (first or additional) from the detail view uses the
  already-existing `openTxFormForContribution` flow — no new transaction
  form variant.

## Decisions

- **Group edit fans a `PATCH` out over every row in the group, exactly like
  `archiveContribGroup`.** Same `Promise.allSettled` + partial-failure
  toast pattern, same reasoning: no new backend endpoint, no transactional
  guarantee beyond what the archive action already accepts as a risk (see
  below).
- **The group edit form is prefilled from an arbitrary row in the group**
  (the group's rows are expected to share `label`/`amount`/`description`/
  `dueDate` — that's the definition of being in the same group before this
  change ships; a group with rows already diverged in `amount` before this
  change ships from manual per-row edits under the old behavior picks one
  row's amount as the new shared value, same as `ContribCreateSheet`
  already applies one shared amount across every member it creates rows
  for).
- **Per-row `archived`/`delete` are dropped from the UI, not from the API.**
  The backend `PATCH`/`DELETE` endpoints are unchanged and untouched;
  removing the frontend affordance is a UI-surface decision only, consistent
  with the request that editing (which includes archive state) happen
  exclusively at the group level. `archiveContribGroup` (group-level
  archive) is unaffected by this change and keeps using the same
  fan-out `PATCH { archived }`.
- **The matrix cell click loses its `paidAmount` branch.** Previously:
  `paidAmount === 0` opened the transaction form directly, `paidAmount > 0`
  opened the edit form. Now every click opens the read-only detail view,
  and the detail view's "Beitrag erfassen" button is the single path into
  the transaction form. This costs one extra click for a first-ever payment
  (cell → detail → erfassen, was cell → transaction form) in exchange for
  a single, predictable click target instead of state-dependent behavior —
  accepted, since the request is explicit that the detail view must be the
  one thing that opens.
- **Sheet type renamed `contribForm` → `contribDetail`, new `contribGroupEdit`
  added**, matching the existing `eventDetail`/`memberDetail` naming
  convention for read-only-detail sheets rather than keeping a `*Form` name
  on a sheet that no longer contains a form.

## Risks

- **Group edit fan-out is N requests, not transactional** — same accepted
  risk as `archiveContribGroup` (see that change's design.md); `Promise.allSettled`
  + a partial-failure toast keeps a partial failure visible instead of silent,
  and re-running is safe (each row's `PATCH` is idempotent for the same
  target values).
- **One more click to book a first payment** (see decision above) — accepted
  as the direct consequence of "the detail view really only opens the detail
  view," which is the explicit ask.
