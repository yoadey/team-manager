## 1. Backend

- [x] 1.1 `internal/jobs/notification_worker.go`: add `Status *string` to
      `NotificationArgs`; include `status` in the `INSERT INTO notifications`
      column list/values
- [x] 1.2 `internal/events/service.go`: after a successful `SetAttendance`
      write, best-effort enqueue an `attendance` notification (actor =
      `userID`, not `callerID`) when `req.Status` is `yes`/`no`/`maybe`; skip
      for `pending`

## 2. Tests

- [x] 2.1 `internal/jobs/notification_worker_test.go`: `Status` round-trips
      through `Work()`'s insert
- [x] 2.2 `internal/events/service_test.go`: `SetAttendance` enqueues an
      `attendance` notification with the responding member as actor for
      yes/no/maybe; enqueues nothing for `pending`; organizer-sets-for-another
      case uses the target member as actor

## 3. Verification

- [x] 3.1 `cd backend && make lint && make test` green
- [x] 3.2 No OpenAPI/migration change, so `backend-openapi-drift` and
      migration CI jobs are unaffected
