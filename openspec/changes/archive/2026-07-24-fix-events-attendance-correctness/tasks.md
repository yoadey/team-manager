## 1. Today boundary
- [x] 1.1 In `events/repository.go` ListEvents, truncate `today` to date granularity (`time.Now().UTC().Truncate(24*time.Hour)`)
- [x] 1.2 Tighten the upcoming-scope repository test to assert the exact expected set (incl. a today-dated event), replacing the `>= 1` assertion

## 2. Series cancellation
- [x] 2.1 Add `AND date >= CURRENT_DATE` to the series-cancel UPDATE in `events/repository.go` (`SetStatus` series branch)
- [x] 2.2 Add a test proving past instances of a cancelled series keep their status (`TestEventRepository_SetStatus_Series_DoesNotCancelPastInstances`)

## 3. Cancelled-event attendance guard
- [x] 3.1 Add `ErrEventCancelled` sentinel; in `SetAttendance` reject changes when the event status is `cancelled` (after the permission check)
- [x] 3.2 Map the sentinel to a client error (409 Conflict) in the handler
- [x] 3.3 Add a test for attendance rejection on a cancelled event (`TestEventService_SetAttendance_RejectsCancelledEvent`)

## 4. Verification
- [x] 4.1 `go test ./internal/events/... -short` green + full `go test ./... -short` green (integration/testcontainer cases run in CI)
- [x] 4.2 `go vet` + `gofmt` clean (full `make lint` runs in CI)
- [ ] 4.3 Coverage gate holds — confirmed by CI's `backend-coverage` job
