## 1. Shared linked-entries UI

- [x] 1.1 Add a small presentational list component (e.g.
      `components/LinkedTransactionsList.tsx`) rendering title/date/amount
      rows for a `Transaction[]`, each clickable to `app.openTxForm(tx)`, with
      an empty state when the list is empty.

## 2. `FinancesPenalties` / `PenaltyAssignSheet` (Strafen detail)

- [x] 2.1 `FinancesPenalties.tsx`: make each assignment row clickable
      (`app.openPenaltyAssign(a)`), keeping the existing delete button
      working via `stopPropagation`.
- [x] 2.2 `useFinanceActions.ts`: `openPenaltyAssign` accepts an optional
      `PenaltyAssignment` — omitted keeps today's `create` sheet; passed opens
      `mode: 'view'` with the assignment's id/userId/penaltyId/date/note as
      `formInitial`.
- [x] 2.3 `PenaltyAssignSheet.tsx`: in `view` mode render the existing
      member/penalty/date/note read-only (no submit button) plus
      `LinkedTransactionsList` filtered from `finances.transactions` by
      `penaltyAssignmentId === assignment.id`.
- [x] 2.4 `sheets/index.tsx`: `sheetMeta` title for `penaltyAssign` in view
      mode (e.g. reuse existing title, no back-nav needed).

## 3. `ContribFormSheet` (Beiträge detail)

- [x] 3.1 `ContribFormSheet.tsx`: fetch `useFinanceOverviewQuery`, add a
      "linked transactions" section (label + `LinkedTransactionsList`
      filtered by `contributionId === contribution.id`) below the existing
      form fields.

## 4. `TxFormSheet` (Umsätze detail)

- [x] 4.1 `TxFormSheet.tsx`: in edit mode, when `contributionId` or
      `penaltyAssignmentId` is set, render a read-only "linked to" row
      (label + amount) resolved from `finances.contributions` /
      `finances.assignments`, clickable to open that entry's own detail
      sheet (`app.openContribForm` / `app.openPenaltyAssign` view mode).

## 5. i18n

- [x] 5.1 Add German + English strings: linked-transactions section title,
      empty state, "linked to" label for `TxFormSheet`.

## 6. Verification

- [x] 6.1 `cd frontend && npm run lint`
- [x] 6.2 `cd frontend && npm run typecheck`
- [x] 6.3 `cd frontend && npm test` (existing + new coverage for the above
      components/hooks)
- [x] 6.4 `cd frontend && npm run build`
