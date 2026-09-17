## 1. Backend: repository

- [x] 1.1 `internal/statsprefs/model.go`: add `ErrInvalidDateRange =
      errors.New("to date must not be before from date")`, doc comment
      mirroring `absences.ErrInvalidDateRange`'s (partial-PATCH gap, only
      catchable here since the merge happens inside the `UPDATE`).
- [x] 1.2 `internal/statsprefs/repository.go`: add the `pgCheckViolation =
      "23514"` SQLSTATE constant (or reuse if already present) and wrap
      `UpdatePreset`'s `QueryRow(...).Scan(...)` error check with
      `errors.As(err, &pgErr) && pgErr.Code == pgCheckViolation` →
      `ErrInvalidDateRange`, mirroring `absences.Repository.Update` exactly
      (single CHECK here, so no constraint-name branch needed).

## 2. Backend: handler

- [x] 2.1 `internal/statsprefs/handler.go`'s `UpdateStatsPreset`: add
      `errors.Is(err, ErrInvalidDateRange)` → `apierror.BadRequest("'from'
      must not be after 'to'")` case, before the `apierror.Internal`
      fallback (after the existing `pgx.ErrNoRows` → 404 case).

## 3. Tests

- [x] 3.1 `internal/statsprefs/repository_test.go`: new test — create a
      preset with a known range, `UpdatePreset` with only `from` set past
      the stored `to`, assert `errors.Is(err, statsprefs.ErrInvalidDateRange)`
      (mirrors `absences/repository_test.go`'s
      `TestAbsenceRepository_Update_PartialPatch_RejectsExcessiveSpan`
      shape, for the plain ordering violation instead of the span one).
- [x] 3.2 `internal/statsprefs/handler_test.go`: new test — mock service
      returns `statsprefs.ErrInvalidDateRange`, assert the handler maps it
      to `*apierror.APIError` with `Status == 400` (mirrors
      `absences/handler_test.go`'s
      `TestAbsenceHandler_UpdateAbsence_InvalidDateRange_Returns400`).

## 4. Verification

- [x] 4.1 `cd backend && go build ./...`
- [x] 4.2 `cd backend && go test ./internal/statsprefs/...` (integration
      repository tests use `testutil.NewTestDB` and skip without Docker —
      note in the result whether they ran or skipped)
- [x] 4.3 `cd backend && make lint`
