## 1. Shared predicate

- [x] 1.1 `backend/internal/attendance/sql.go`: `AbsenceCoversExpr` and
      `NotRelevantAbsenceCoversExpr` change `ab.from_date <= e.date` to
      `ab.from_date <= COALESCE(e.end_date, e.date)`, keeping `ab.to_date
      >= e.date` (`EffectiveStatusExpr` inherits the fix via
      `AbsenceCoversExpr`)

## 2. Call sites

- [x] 2.1 `backend/internal/events/repository.go`: fix every inline
      duplicate of the predicate (`GetMyEffectiveAttendance`,
      `GetMyEffectiveAttendances`) identically; confirm
      `GetAttendanceSummary`, `GetAttendanceSummaries`, `ListAttendance`
      pick up the fix automatically via the shared helper
- [x] 2.2 `backend/internal/stats/repository.go`: fix its own inline
      copy of the predicate, if any, identically (or confirm it already
      consumes the shared helper) — confirmed it has no inline copy;
      consumes the shared helper exclusively

## 3. Tests

- [x] 3.1 Add/extend a test pinning the bug: a multi-day event, a member
      with a planned absence covering only the later portion of the
      event's span (not the start date), asserting effective attendance
      resolves to absent/not-pending — in whichever of
      `internal/attendance`, `internal/events`, `internal/stats` has the
      right existing harness for `EffectiveStatusExpr`/
      `GetAttendanceSummary`/`GetMyEffectiveAttendance` — added
      `TestEventRepository_MultiDayEvent_AbsenceCoversLaterPortion` in
      `internal/events/repository_test.go`

## 4. Verification

- [x] 4.1 `cd backend && go build ./...`
- [x] 4.2 `cd backend && go test ./internal/attendance/... ./internal/events/... ./internal/stats/...`
      (integration tests skip without Docker in this environment; CI has
      Docker and will run them)
- [x] 4.3 `cd backend && golangci-lint run ./internal/attendance/...
      ./internal/events/... ./internal/stats/...` — 0 issues (first
      attempt hit a transient "parallel golangci-lint is running" lock
      from a concurrent process in this environment; retry succeeded)
