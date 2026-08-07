## 1. Database

- [x] 1.1 `00022_contribution_archive_description.sql`: `contributions` gains
      `description TEXT`, `archived BOOLEAN NOT NULL DEFAULT FALSE`
- [x] 1.2 `00023_contribution_archive_description_indexes.sql` (`NO
      TRANSACTION`): partial index backing "exclude archived" scans, e.g.
      `idx_contributions_team_archived ON contributions (team_id) WHERE NOT
      archived` (`CONCURRENTLY`)
- [x] 1.3 `00024_transaction_note.sql`: `transactions` gains `note TEXT`
- [x] 1.4 `make migrate` locally if Docker is available; otherwise rely on
      CI's `backend-migration-rollback` (up→down→up) and
      `backend-migration-safety` gates

## 2. OpenAPI

- [x] 2.1 `Contribution`: add `description` (nullable string), `archived`
      (bool, required in response)
- [x] 2.2 `CreateContributionRequest`: add optional `description`
      (`maxLength: 2000`)
- [x] 2.3 `UpdateContributionRequest`: add optional `description`
      (`maxLength: 2000`), `archived` (bool)
- [x] 2.4 `Transaction`: add `note` (nullable string)
- [x] 2.5 `CreateTransactionRequest`/`UpdateTransactionRequest`: add optional
      `note` (`maxLength: 10000`)
- [x] 2.6 `cd backend && make generate` (commit `internal/gen/api.gen.go`)
- [x] 2.7 repo-root `make generate-ts` (commit
      `frontend/src/api/{types.gen.ts,zod.gen.ts}`)

## 3. Backend: finances module

- [x] 3.1 `model.go`: `ContributionRow` gains `Description *string`,
      `Archived bool`; `ContributionPatch` gains `Description *string`,
      `Archived *bool`; `TransactionRow`/`TransactionPatch` gain
      `Note *string`
- [x] 3.2 `repository.go`: `contributionSelectColumns`/`scanContributionRow`
      include `description`/`archived`; `UpdateContribution`'s
      `sqlbuilder` calls add both; `CountOpenContributions` adds `AND NOT
      c.archived`; transaction select/insert/update columns include `note`
- [x] 3.3 `service.go`: `UpdateContribution` maps `body.Description`/
      `body.Archived` into the patch; `CreateTransaction`/`UpdateTransaction`
      map `body.Note`; mapper functions (`toGenContribution`,
      `toGenTransaction`) include the new fields
- [x] 3.4 `handler.go`: validate `Description`/`Note` with
      `validate.MaxLen` (2000 / 10000) when present on create/update bodies

## 4. Backend: tests

- [x] 4.1 `repository_test.go`: `UpdateContribution` persists
      description/archived; `CountOpenContributions` excludes archived rows
      (paid, partial, and open archived cases); transaction note round-trips
      through create/update
- [x] 4.2 `service_test.go`: patch mapping for description/archived/note
- [x] 4.3 `handler_test.go`: validation errors for oversized
      description/note; archived round-trip through the update endpoint

## 5. Frontend: types, API mapping, mocks

- [x] 5.1 `features/finances/types.ts`: `Contribution` gains `description?`,
      `archived`; `Transaction` gains `note?`
- [x] 5.2 `api/map.ts`: `mapContribution`/`mapTransaction` map the new fields
- [x] 5.3 `services/serviceLayerReal.ts`: `updateContribution` passes
      `description`/`archived`; `addTransaction`/`updateTransaction` pass
      `note`
- [x] 5.4 `mocks/db.ts`: `ContributionRow` gains `description?`/`archived`;
      transaction mock row gains `note?`
- [x] 5.5 `mocks/handlers.ts`: `toWireContribution`/`toWireTransaction`
      include new fields; update-contribution handler applies
      description/archived patches; create/update-transaction handlers apply
      `note`

## 6. Frontend: shared status helper

- [x] 6.1 New `features/finances/contributionStatus.ts`:
      `contributionAmountStatus(amount, paidAmount)` returning
      `{ status: 'open'|'partial'|'paid'|'overpaid', displayAmount, excess }`,
      used by the list row, matrix cell, and linking-picker cell
- [x] 6.2 Unit tests for `contributionStatus.ts` covering all four buckets
      including the `paidAmount === amount` (`paid`, not `overpaid`) boundary

## 7. Frontend: contribution edit + archive

- [x] 7.1 `components/contribFormSchema.ts`: add `description`
- [x] 7.2 `components/ContribFormSheet.tsx`: description textarea; replace
      the single delete action with delete + "Archivieren"/"Aus dem Archiv
      holen" (toggling based on the row's current `archived`)
- [x] 7.3 `hooks/useFinanceMutations.ts`: `SaveContribInput.payload` gains
      `description?`/`archived?`
- [x] 7.4 `hooks/useFinanceActions.ts`: `openContribForm`/`saveContrib`
      pass description/archived through; new `archiveContribGroup(group,
      archived)` fanning `saveContribAsync({archived})` out over every row
      in the group via `Promise.allSettled`, reporting partial failure
- [x] 7.5 `components/FinancesContributions.tsx`: "Archivierte Beiträge
      anzeigen" toggle (default off) filtering both the group chips and the
      row list; "Archiviert" chip on archived rows/groups when the toggle is
      on; group summary card gains an archive/un-archive bulk action button

## 8. Frontend: contribution matrix view

- [x] 8.1 New `components/ContribMatrixView.tsx`: member × fee-group grid
      reusing `Stats.tsx`'s `MatrixView` table/sticky-column structure;
      cells use `contributionAmountStatus` + a new overpaid glyph/color
- [x] 8.2 `FinancesContributions.tsx`: list/matrix tab switch (mirrors
      `Stats.tsx`'s `quota`/`matrix` tabs)
- [x] 8.3 Tests: `ContribMatrixView.test.tsx`, updated
      `FinancesContributions.test.tsx`

## 9. Frontend: linking-picker matrix + transaction note

- [x] 9.1 New `components/ContribLinkMatrixDialog.tsx`: MUI `Dialog` wrapping
      a member × fee-group grid of single-select checkboxes (open,
      non-archived contributions only), each showing the still-owed amount
- [x] 9.2 `components/LinkedPaymentPicker.tsx`: contribution tab opens
      `ContribLinkMatrixDialog` instead of the flat list; penalty tab
      unchanged
- [x] 9.3 `components/txFormSchema.ts`: add `note`
- [x] 9.4 `components/TxFormSheet.tsx`: optional note textarea (create +
      edit), not rendered by `FinancesTransactions`
- [x] 9.5 `hooks/useFinanceActions.ts`: `openTxForm`/`saveTx` pass `note`
      through
- [x] 9.6 Tests: `ContribLinkMatrixDialog.test.tsx`, updated
      `LinkedPaymentPicker.test.tsx`, `TxFormSheet.test.tsx`

## 10. i18n

- [x] 10.1 `i18n/{de,en}.ts`: new `finances.*` keys for description field,
      archive/un-archive actions and confirms, "archivierte anzeigen"
      toggle, matrix tab labels/headers/empty state, overpaid status label,
      linking-matrix dialog, note field

## 11. Verification

- [x] 11.1 `openspec validate contribution-archive-matrix-editing --strict`
- [x] 11.2 `cd backend && make generate` / repo-root `make generate-ts` — no
      diff
- [x] 11.3 `cd backend && make lint`
- [x] 11.4 `cd backend && make test`
- [x] 11.5 `cd frontend && npm run lint && npm run typecheck && npm test &&
      npm run build`
