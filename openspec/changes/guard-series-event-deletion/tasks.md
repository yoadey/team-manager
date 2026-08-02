## 1. Backend

- [ ] 1.1 `repository.go`: `DeleteEvent`'s series branch adds
      `AND date >= CURRENT_DATE` to the `DELETE FROM events WHERE
      series_id = $1 AND team_id = $2` statement, mirroring `SetStatus`
- [ ] 1.2 `repository.go`: `updateSeriesEvents` adds the same
      `date >= CURRENT_DATE` guard to its `UPDATE` statement
- [ ] 1.3 Verify the specific event addressed by `eventID` is still
      deleted/updated individually regardless of its own date (existing
      behavior, must not regress)
- [ ] 1.4 `repository_test.go`: cover series delete/update with a mix of
      past and future occurrences — assert past occurrences (and their
      attendance/comments, for delete) survive untouched, future ones
      (and the specifically addressed event) are affected

## 2. Frontend

- [ ] 2.1 Series delete/edit confirmation copy in
      `frontend/src/features/events/` reviewed and updated if it
      currently implies past occurrences are also removed/changed

## 3. Verification

- [ ] 3.1 `cd backend && make test-unit`
- [ ] 3.2 `cd backend && make test-integration`
- [ ] 3.3 `cd backend && make lint`
- [ ] 3.4 `cd frontend && npm test` (if confirmation copy/tests changed)
