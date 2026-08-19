## Why

The shared "does a planned absence cover this event" SQL predicate only
checks the event's *start* date (`ab.from_date <= e.date AND ab.to_date >=
e.date`), even though `copy-and-multiday-events` added a nullable
`events.end_date` so an event can span multiple days. An absence that
covers only the *later* part of a multi-day event's span (e.g. a 3-day
camp `06-01..06-03`, absence logged `06-02..06-03`) never satisfies
`ab.from_date <= e.date` for `e.date = 06-01`, so it's never treated as
covering the event at all — the member's effective attendance defaults to
pending/attending for the whole event instead of resolving to absent.

This predicate is the shared source of truth
(`backend/internal/attendance/sql.go`'s `AbsenceCoversExpr`,
`EffectiveStatusExpr`, `NotRelevantAbsenceCoversExpr`), consumed by
`internal/events` (event attendance summary, `GetMyEffectiveAttendance(s)`)
and `internal/stats` (attendance-rate statistics) — so the bug
silently miscounts absences in both the event summary and season-long
stats for every multi-day event.

## What Changes

- Change the absence-coverage predicate everywhere it appears (the shared
  `internal/attendance/sql.go` helpers, plus every inline duplicate in
  `internal/events/repository.go` and `internal/stats/repository.go`) from
  a start-date-only check to a real date-range intersection against the
  event's full span: `ab.from_date <= COALESCE(e.end_date, e.date) AND
  ab.to_date >= e.date`.
- No API shape change, no migration — this only corrects the effective
  status a covering absence resolves to for multi-day events.

## Capabilities

### Modified Capabilities
- `attendance-statistics`: effective attendance (and the derived
  statistics/matrix) for a multi-day event now correctly resolves to "no"
  when a member's planned absence overlaps any part of the event's span,
  not only its start date.

## Impact

- `backend/internal/attendance/sql.go` (`AbsenceCoversExpr`,
  `NotRelevantAbsenceCoversExpr`; `EffectiveStatusExpr` inherits the fix
  via `AbsenceCoversExpr`).
- `backend/internal/events/repository.go`: `GetAttendanceSummary`,
  `GetAttendanceSummaries`, `GetMyEffectiveAttendance` (inline
  duplicate), `GetMyEffectiveAttendances` (inline duplicate),
  `ListAttendance`.
- `backend/internal/stats/repository.go`, if it holds its own inline
  copy rather than consuming the shared helper.
- Test coverage: a multi-day event + an absence covering only the later
  portion of its span, asserting effective attendance resolves to
  absent/not-pending.
