## Why

`Finanzen -> Beiträge` rows are grouped into a fee period purely by matching
`(label, dueDate)` client-side (`FinancesContributions.groupKey`) — there is
no group entity in the database. Today, opening a single member's row (a
matrix cell with `paidAmount > 0`, or the list view's edit button) opens
`ContribFormSheet`, a fully editable form for that one row's `label`,
`amount`, `description`, and `dueDate`, plus per-row archive/delete. That is
exactly the wrong editing surface for this data shape:

- Renaming or re-dating a single row silently detaches it from its group
  (`groupKey` no longer matches), splitting one fee period into two without
  any warning.
- Changing one member's `amount` lets their fee silently diverge from
  everyone else's in the same period, with no visual indicator that
  something is now inconsistent.
- There is already a group-level bulk **archive** action
  (`archiveContribGroup`, fanning a PATCH out over every row in the group)
  specifically because "archive this whole fee period" is the actual
  Kassenwart use case, not archiving members one at a time — but no
  equivalent exists for *editing* name/amount/description/due date. The only
  way to change those today is the single-row form this proposal removes.

Separately, a matrix cell's click target today branches on `paidAmount`:
zero opens the transaction-booking form directly, anything above opens the
edit form. That means the same click target does two structurally different
things depending on state, and there's no detail view at all for a cell
that already has an open partial payment plus more still owed.

## What Changes

- **The single-contribution detail view becomes read-only.** Reusing the
  sheet reached from a matrix cell or the list view's row action, it always
  shows: the member's name, the paid/required amount, and every transaction
  linked to that row (`LinkedTransactionsList`, unchanged) — nothing else.
  `label`/`amount`/`description`/`dueDate` are no longer editable there, and
  the per-row archive/delete actions are removed from this view.
- **A "Beitrag erfassen" button replaces the old paid/unpaid click branch.**
  The detail view always opens (no more "unpaid cell skips straight to the
  transaction form" special case); a button inside it opens the existing
  transaction-booking form pre-linked to that contribution
  (`openTxFormForContribution`), covering both a first payment and any
  further partial payment.
- **New group-level "Bearbeiten" action**, next to the existing "Fälligkeitsperiode
  archivieren" action on the group summary card. Opens a form (label, amount,
  description, due date — prefilled from the group's current shared values)
  that, on save, fans the change out as a `PATCH` over every row in the
  group (same mechanism as `archiveContribGroup`), keeping every member's
  row in that fee period identical and still matched by `groupKey` after the
  edit.
- Per-row delete is dropped from the UI entirely (no replacement) — nothing
  in the request asks for it, and it becomes unreachable once the read-only
  detail view no longer exposes it.

## Capabilities

### Modified Capabilities

- `membership-fees`: the single-contribution view is read-only; editing
  name/amount/description/due date moves to a new group-level action.
- `finance-matrix-transactions`: matrix cell click always opens the
  read-only detail view instead of branching between the transaction form
  and the edit form.

## Impact

- **Frontend only** — reuses the existing
  `PATCH /teams/{teamId}/finances/contributions/{contributionId}` endpoint
  for both the (now group-fanned-out) edit and the existing archive
  fan-out; no OpenAPI/backend change.
- `frontend/src/features/finances/components/ContribFormSheet.tsx` (gutted
  to read-only, renamed `ContribDetailSheet.tsx`), new
  `ContribGroupEditSheet.tsx`, new `contribGroupEditFormSchema.ts`
  (replacing `contribFormSchema.ts`), `FinancesContributions.tsx` (group
  "Bearbeiten" button, row action re-pointed to the detail view),
  `ContribMatrixView.tsx` (cell click always opens the detail view),
  `hooks/useFinanceActions.ts` (`openContribForm`/`saveContrib`/
  `deleteContrib` replaced by `openContribDetail`,
  `openContribGroupEdit`/`editContribGroup`), `context/AppContext.tsx`
  (`SheetType` gains `contribDetail`/`contribGroupEdit`, drops
  `contribForm`), `features/finances/index.ts`, `sheets/index.tsx`
  (sheet titles), `i18n/{de,en}.ts`, and the relevant tests.
