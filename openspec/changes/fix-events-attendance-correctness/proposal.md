## Why

Three events-domain correctness bugs surfaced in the architecture audit remain open on `main`:

1. **Today's events disappear from `scope=upcoming`.** `events/repository.go:141` uses `today := time.Now().UTC()` (a full timestamp) and compares it against the `DATE` column (`date >= $2`). From 00:00:01 UTC onward, a `DATE` casts to midnight and is no longer `>=` the current timestamp, so today's training vanishes from the upcoming list — exactly on the day it matters. The existing test masks it (`>= 1` assertion; and it truncates `today` itself).
2. **Cancelling "the rest of a series" retroactively cancels past instances.** The series-cancel UPDATE has no date filter, so already-held trainings flip to `cancelled` and drop out of history/stats.
3. **Attendance is editable on cancelled events** (and, for self-service, on past events), letting a member rewrite history after the fact.

## What Changes

- Truncate `today` to date granularity (or compare against `CURRENT_DATE`) so today's events are `upcoming`.
- Add `AND date >= CURRENT_DATE` to the series-cancel UPDATE so only future instances are cancelled.
- Reject self-service attendance changes on cancelled events (and document past-event editability policy).
- Tighten the too-lenient upcoming test to assert the exact expected set.

## Capabilities

### New Capabilities
- `events-scheduling`: how events are classified into past/upcoming, how series cancellation scopes instances, and when attendance may be recorded.

### Modified Capabilities
<!-- none: fresh spec under OpenSpec adoption -->

## Impact

- Backend: `internal/events/repository.go` (ListEvents boundary, series cancel), `internal/events/service.go` (attendance guard), related tests.
- No API/schema change; no migration.
- CI gates: backend lint/test/coverage.
