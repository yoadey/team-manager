## 1. Sheet plumbing

- [x] 1.1 `context/AppContext.tsx`: `SheetType` replaces `contribForm` with
      `contribDetail`, adds `contribGroupEdit`
- [x] 1.2 `context/AppContext.tsx`: `AppContextValue` replaces
      `openContribForm`/`saveContrib`/`deleteContrib` with
      `openContribDetail`, `openContribGroupEdit`, `editContribGroup`
      (`archiveContribGroup` unchanged)
- [x] 1.3 `sheets/index.tsx`: `sheetTitles` replaces the `contribForm` entry
      with `contribDetail` (e.g. "Beitrag") and adds `contribGroupEdit`
      (e.g. "Beitrag bearbeiten")
- [x] 1.4 `features/finances/index.ts`: `financeSheetMap` replaces
      `contribForm: ContribFormSheet` with `contribDetail: ContribDetailSheet`,
      adds `contribGroupEdit: ContribGroupEditSheet`

## 2. Read-only detail sheet

- [x] 2.1 New `components/contribDetailSheet.tsx` types (or inline): the
      sheet only needs the contribution `id`; the full `Contribution` (name,
      paid/required amount, archived, etc.) is looked up from
      `useFinanceOverviewQuery`'s already-loaded `finances.contributions`,
      same as today's `linkedTx` lookup in `ContribFormSheet`
- [x] 2.2 Rename `components/ContribFormSheet.tsx` -> `ContribDetailSheet.tsx`,
      gut it to read-only: `Av` + member name, a paid/required amount line
      (reuse the `contributionAmountStatus`-derived formatting already used
      by `FinancesContributions`'s list row), `LinkedTransactionsList`
      (unchanged), and a "Beitrag erfassen" `PrimaryButton` calling
      `app.openTxFormForContribution(contribution)`. Remove the
      `useForm`/`zodResolver` form, the label/amount/description/dueDate
      fields, the archive toggle, and the delete button entirely
- [x] 2.3 Delete `components/contribFormSchema.ts` (no longer used — the
      detail view isn't a form)
- [x] 2.4 `hooks/useFinanceActions.ts`: replace `openContribForm`/
      `saveContrib`/`deleteContrib` with `openContribDetail(c: Contribution)`
      setting `{ sheet: { type: 'contribDetail', formInitial: { id: c.id } } }`;
      remove the now-unused `useDeleteContributionMutation` wiring
- [x] 2.5 `hooks/useFinanceMutations.ts`: drop `useDeleteContributionMutation`
      if `deleteContrib` was its only caller (check other call sites first)

## 3. Group edit sheet

- [x] 3.1 New `components/contribGroupEditFormSchema.ts`: `label`, `amount`,
      `description`, `dueDate` (same validation as the old
      `contribFormSchema`, minus `id`/`archived`)
- [x] 3.2 New `components/ContribGroupEditSheet.tsx`: form with those four
      fields (mirrors `ContribCreateSheet`'s label/amount/dueDate fields,
      plus the description textarea from the old `ContribFormSheet`),
      submitting via `app.editContribGroup(rows, values)`
- [x] 3.3 `hooks/useFinanceActions.ts`: `openContribGroupEdit(rows: Contribution[])`
      opens the sheet prefilled from `rows[0]`'s current values (label,
      amount, description, dueDate); `editContribGroup(rows, values)` fans
      `saveContribAsync({ id: c.id, payload: values })` out over every row
      via `Promise.allSettled`, reporting partial failure — mirrors
      `archiveContribGroup` exactly

## 4. Wire up call sites

- [x] 4.1 `components/FinancesContributions.tsx`: the row action button
      calls `app.openContribDetail(c)` (was `openContribForm`); update its
      `aria-label`/icon (no longer "edit" — e.g. `visibility`/chevron) and
      the `editContribLabel` i18n key's meaning accordingly
- [x] 4.2 `components/FinancesContributions.tsx`: add a "Bearbeiten" button
      next to the existing `archiveGroupAction` on the group summary card,
      calling `app.openContribGroupEdit(allGroupRows)`
- [x] 4.3 `components/ContribMatrixView.tsx`: cell `onClick` always calls
      `app.openContribDetail(c)`; remove the
      `c.paidAmount > 0 ? openContribForm(c) : openTxFormForContribution(c)`
      branch

## 5. i18n

- [x] 5.1 `i18n/{de,en}.ts`: `sheet.contribForm` -> `sheet.contribDetail`;
      add `sheet.contribGroupEdit`
- [x] 5.2 `i18n/{de,en}.ts`: add `finances.contribRecordPayment` ("Beitrag
      erfassen"), `finances.contribDetailPaid`/`contribDetailRequired` (or
      reuse the existing paid/required phrasing from the list row),
      `finances.contribGroupEditBtn` ("Bearbeiten"),
      `finances.toastContribGroupEditSuccess`/
      `toastContribGroupEditPartialFailure`
- [x] 5.3 `i18n/{de,en}.ts`: remove now-unused keys —
      `finances.contribDeleteTitle`, `contribDeleteMsg`, `toastContribDeleted`,
      `contribArchive`, `contribUnarchive`, `contribArchivedNotice` (if no
      longer rendered anywhere), `editContribLabel` (repurposed or removed
      per 4.1)

## 6. Tests

- [x] 6.1 Rename/rewrite `ContribFormSheet.test.tsx` ->
      `ContribDetailSheet.test.tsx`: renders name/paid/required/linked
      transactions read-only; no editable fields; "Beitrag erfassen" calls
      `openTxFormForContribution`
- [x] 6.2 New `ContribGroupEditSheet.test.tsx`: prefilled from group values;
      submit fans out `editContribGroup` over every row
- [x] 6.3 `useFinanceActions.test.ts`: replace `openContribForm`/`saveContrib`/
      `deleteContrib` assertions with `openContribDetail`/
      `openContribGroupEdit`/`editContribGroup` (partial-failure case
      included)
- [x] 6.4 `ContribMatrixView.test.tsx`: every populated cell click calls
      `openContribDetail`, regardless of `paidAmount`; empty cell calls
      neither
- [x] 6.5 `FinancesContributions.test.tsx`: row action opens the detail
      sheet; new group "Bearbeiten" button opens the group edit sheet
- [x] 6.6 `FinancesPage.test.tsx`: update any sheet-type/title assertions
      referencing `contribForm`

## 7. Verification

- [x] 7.1 `openspec validate contribution-detail-readonly-parent-edit --strict`
- [x] 7.2 `cd frontend && npm run lint` (0 errors)
- [x] 7.3 `cd frontend && npm run typecheck`
- [x] 7.4 `cd frontend && npm test` (coverage thresholds 80/65/75/80 maintained)
- [x] 7.5 `cd frontend && npm run build && npm run check:bundle`
