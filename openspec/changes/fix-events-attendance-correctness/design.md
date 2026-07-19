## Context

`ListEvents` (`events/repository.go:138-180`) sets `today := time.Now().UTC()` and builds `date < $2` (past) / `date >= $2` (upcoming) against the `events.date` `DATE` column. Series cancellation updates `WHERE series_id = $2 AND team_id = $3` with no date bound. `SetAttendance` re-checks permission/membership atomically but does not look at the event's `date` or `status`.

## Goals / Non-Goals

**Goals:**
- Today's events classify as upcoming.
- Series cancellation affects only instances from today forward.
- Attendance cannot be changed on a cancelled event via self-service.

**Non-Goals:**
- Reworking the cursor/keyset pagination.
- Deciding a broad "no editing past attendance ever" policy — see Decisions; only the cancelled-event case is enforced here, and past-event self-service editing is documented for a follow-up product decision.

## Decisions

- Use `today := time.Now().UTC().Truncate(24 * time.Hour)` so the `DATE >=` comparison includes today, matching the existing test helper and the frontend's local-date expectation. (SQL `CURRENT_DATE` is an equivalent alternative; Truncate keeps the parameterized-query shape unchanged.)
- Series cancel: add `AND date >= CURRENT_DATE` to the UPDATE.
- Attendance guard: in `SetAttendance`, if the event is `cancelled`, reject self-service changes with a domain sentinel (`ErrEventCancelled`) mapped to 409/422. Trainers with `events:write` performing roster management are out of scope of the self-service path and unaffected.
- Past-event self-service editing: **not** blocked in this change (it interacts with legitimate late corrections); documented as an open product decision. Enforcing it later is a one-line date check mirroring the cancelled guard.

## Risks / Trade-offs

- Timezone nuance: truncating in UTC means a club east of UTC editing just after local midnight sees the UTC day; acceptable and matches current behavior elsewhere. A later move to team-local dates would be a separate change.
- The cancelled-event guard changes behavior for any UI that lets members toggle attendance on cancelled events; verify the frontend hides that path (it renders cancelled events read-only).
