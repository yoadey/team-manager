## 1. Backend

- [x] 1.1 `repository.go`: `DeleteEvent`'s series branch adds
      `AND date >= CURRENT_DATE` to the `DELETE FROM events WHERE
      series_id = $1 AND team_id = $2` statement, mirroring `SetStatus`
- [x] 1.2 `repository.go`: `updateSeriesEvents` adds the same
      `date >= CURRENT_DATE` guard to its `UPDATE` statement
- [x] 1.3 Verify the specific event addressed by `eventID` is still
      deleted/updated individually regardless of its own date (existing
      behavior, must not regress)
- [x] 1.3b `repository.go`: the `event_series` row is only deleted once
      no event still references it — dropping it while past occurrences
      survive would detach them (`series_id` is `ON DELETE SET NULL`)
- [x] 1.4 `repository_test.go`: cover series delete/update with a mix of
      past and future occurrences — assert past occurrences (and their
      attendance/comments, for delete) survive untouched, future ones
      (and the specifically addressed event) are affected

## 2. Frontend

- [x] 2.1 Series delete confirmation copy (`deleteSeriesMsg`,
      `seriesDeleteDesc` in `frontend/src/i18n/de.ts` and `en.ts`) no
      longer claims *all* events of the series are removed — it states
      that occurrences from today on go and past ones are kept
- [x] 2.2 `docs/end-user/termine.md` explains what "die ganze Serie"
      covers, since the chapter documents series delete/cancel/edit

## 3. Demo backend (MSW)

- [x] 3.1 `frontend/src/mocks/handlers.ts`: shared `seriesTargets` helper
      applies the same "today onwards, plus the addressed event" scoping
      to the series delete, status and patch handlers — the status
      handler had already drifted from the real backend's long-standing
      `SetStatus` guard
- [x] 3.2 `frontend/src/services/serviceContract.test.ts`: pins the demo
      backend's series delete/cancel/edit behavior against past
      occurrences

## 4. Verification

- [x] 4.1 `cd backend && make test-unit`
- [x] 4.2 `cd backend && make test-integration`
- [x] 4.3 `cd backend && make lint`
- [x] 4.4 `cd frontend && npm test`
- [x] 4.5 `cd frontend && npm run typecheck && npm run lint`
