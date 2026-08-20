## Context

`internal/attendance/sql.go` exists specifically so the event summary
(`internal/events`) and statistics (`internal/stats`) can't diverge on
how a member's effective attendance for an event is derived. Its
absence-coverage check was written before multi-day events existed and
was never updated when `copy-and-multiday-events` added `events.end_date`
(nullable — null means single-day). It still only tests the absence
range against the event's single `date` column, so it silently ignores
`end_date` entirely.

## Goals

- Absence coverage for an event is a real date-range intersection against
  the event's full span (`date` through `COALESCE(end_date, date)`), not
  just its start day.
- Fix the shared helper and every inline duplicate identically, so
  `internal/events` and `internal/stats` keep agreeing with each other.
- Minimal, mechanical change — no behavior change for single-day events
  (`end_date IS NULL` collapses `COALESCE(e.end_date, e.date)` back to
  `e.date`, i.e. the existing predicate).

## Decisions

- **New predicate:** `ab.from_date <= COALESCE(e.end_date, e.date) AND
  ab.to_date >= e.date`. This is the standard range-overlap test (`A.start
  <= B.end AND A.end >= B.start`) applied to `[e.date, COALESCE(e.end_date,
  e.date)]` as the event's span and `[ab.from_date, ab.to_date]` as the
  absence's span.
- **Fix inline duplicates in place rather than a broad refactor.** The
  finding notes call sites in `internal/events/repository.go`
  (`GetMyEffectiveAttendance`/`GetMyEffectiveAttendances`) re-derive the
  expression inline instead of calling the shared helper. Refactoring
  those call sites to consume `attendance.AbsenceCoversExpr`/
  `EffectiveStatusExpr` directly would be an improvement, but only if it's
  a clean drop-in (same table aliases already in scope); if a call site's
  query shape doesn't line up cleanly with the shared helper's assumed
  aliases (`m`/`e`/`a`), fix that copy identically instead of forcing a
  refactor and risking an unrelated regression.
- **`internal/stats/repository.go` checked for its own inline copy** per
  the finding, and fixed identically if present, even though the package
  doc for `internal/attendance/sql.go` says it's meant to be consumed by
  `internal/stats` too.

## Risks

- Low risk: this is a pure widening of an EXISTS predicate (more absences
  now count as "covering"), not a schema or API change. The only
  behavioral change is for multi-day events where an absence's range
  overlaps the span without covering the start date — exactly the bug
  being fixed.
