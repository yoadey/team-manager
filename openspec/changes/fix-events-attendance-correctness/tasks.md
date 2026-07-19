## 1. Today boundary
- [ ] 1.1 In `events/repository.go` ListEvents, truncate `today` to date granularity (`time.Now().UTC().Truncate(24*time.Hour)`)
- [ ] 1.2 Tighten the upcoming-scope repository test to assert the exact expected set (incl. a today-dated event), replacing the `>= 1` assertion

## 2. Series cancellation
- [ ] 2.1 Add `AND date >= CURRENT_DATE` to the series-cancel UPDATE in `events/repository.go`
- [ ] 2.2 Add/adjust a test proving past instances of a cancelled series keep their status

## 3. Cancelled-event attendance guard
- [ ] 3.1 Add `ErrEventCancelled` sentinel; in `SetAttendance` reject self-service changes when the event status is `cancelled`
- [ ] 3.2 Map the sentinel to a client error (409/422) in the handler
- [ ] 3.3 Add a test for attendance rejection on a cancelled event

## 4. Verification
- [ ] 4.1 `cd backend && make test` green (new + existing events tests)
- [ ] 4.2 `make lint` green
- [ ] 4.3 Coverage gate holds
