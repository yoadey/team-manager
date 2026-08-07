## 1. Contribution matrix: default view + click-to-book/inspect

- [x] 1.1 `FinancesContributions.tsx`: default the `view` state to `'matrix'`
- [x] 1.2 `useFinanceActions.ts`: new `openTxFormForContribution(c: Contribution)` action — opens the transaction form in create mode pre-filled with `type: 'income'`, `title: c.label`, `amount` = amount still owed, `contributionId: c.id`, `date` = today
- [x] 1.3 `AppContext.tsx`: wire `openTxFormForContribution` through
- [x] 1.4 `ContribMatrixView.tsx`: accept `app`, make populated cells clickable — `paidAmount === 0` calls `openTxFormForContribution(c)`, `paidAmount > 0` calls `openContribForm(c)`; empty cells stay non-interactive
- [x] 1.5 `FinancesContributions.tsx`: pass `app` down to `ContribMatrixView`

## 2. Compact link-picker matrix

- [x] 2.1 `ContribLinkMatrixDialog.tsx`: remove the `<Av>` avatar from the member column
- [x] 2.2 `ContribLinkMatrixDialog.tsx`: reduce header/cell padding and column min-widths
- [x] 2.3 `ContribLinkMatrixDialog.tsx`: remove border-radius from the interior cell selection button

## 3. Single-step fee/penalty linking

- [x] 3.1 New `PenaltyLinkDialog.tsx` — a `Dialog` with a search input and a list of open penalty assignments (mirrors the search/list already inline in `LinkedPaymentPicker`), selecting one calls `onSelect(id)` and closes
- [x] 3.2 `LinkedPaymentPicker.tsx`: replace the collapsed toggle + kind-switch panel with an always-visible "Verknüpfen mit" heading and two buttons ("Beiträge" opens `ContribLinkMatrixDialog`, "Strafen" opens `PenaltyLinkDialog`); keep the existing selected-summary display (change/remove) once something is linked
- [x] 3.3 New i18n keys for `PenaltyLinkDialog` (title, search placeholder, empty state) in de/en; reuse `linkedPickerKindContrib`/`linkedPickerKindPenalty` for the two button labels

## 4. Transaction date field

- [x] 4.1 `txFormSchema.ts`: add required `date: z.string()`
- [x] 4.2 `TxFormSheet.tsx`: add a `Field` with a `TextInput type="date"` for `date`
- [x] 4.3 `useFinanceActions.ts`: `openTxForm` defaults `date` to `todayStr()` on create, `tx.date` on edit; `saveTx` includes `date` in the mutation payload
- [x] 4.4 `useFinanceMutations.ts`: `SaveTxInput.payload` gains `date`

## 5. Category filter on the transactions list

- [x] 5.1 `FinancesTransactions.tsx`: derive the distinct category list from `f.transactions`, add filter chips (including an "all" option) with local component state, filter the rendered rows
- [x] 5.2 New i18n keys for the filter (label, "all" option) in de/en

## 6. Tests

- [x] 6.1 `FinancesContributions.test.tsx`: update tests that assume list-view-by-default to switch to list first; add/adjust a test asserting matrix renders by default
- [x] 6.2 `ContribMatrixView.test.tsx`: clicking an unpaid cell calls `openTxFormForContribution`; clicking a paid/partial cell calls `openContribForm`; clicking an empty cell calls neither
- [x] 6.3 `ContribLinkMatrixDialog.test.tsx`: no avatar image rendered
- [x] 6.4 `useFinanceActions.test.ts`: `openTxFormForContribution` pre-fill assertions; `saveTx` sends `date`
- [x] 6.5 `LinkedPaymentPicker.test.tsx`: rewrite for the single-step layout (no collapsed toggle); fee button opens the matrix dialog, penalty button opens `PenaltyLinkDialog`
- [x] 6.6 New `PenaltyLinkDialog.test.tsx`
- [x] 6.7 `TxFormSheet.test.tsx`: date field present, defaults to today on create / existing date on edit, submits `date`; update existing linking tests for the new button-driven flow
- [x] 6.8 `FinancesTransactions.test.tsx`: category filter narrows the list and clears back to showing everything

## 7. Verification

- [x] 7.1 `cd frontend && npm run lint` (0 errors) + `npm run typecheck` green
- [x] 7.2 `npm test` green (coverage thresholds 80/65/75/80 maintained)
- [x] 7.3 `npm run build` + `npm run check:bundle` within budget
