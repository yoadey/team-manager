## Why

`SetStatus`'s series-wide branch (`backend/internal/events/repository.go:673-683`)
deliberately restricts itself to `date >= CURRENT_DATE`, with an explicit
comment: "Bulk-changing the status of already-held (past) occurrences
would retroactively rewrite team history... cancelling 'the rest of the
series' must not flip completed trainings to cancelled and drop them from
stats." `DeleteEvent` with `scope=series` (`repository.go:704-746`) has no
such guard — its own doc comment says it deletes "all occurrences, past
and future, plus their attendance and comments." `UpdateEvent`'s
`updateSeriesEvents` (`repository.go:571-602`) likewise applies to every
event in the series regardless of date.

Concrete failure scenario: an organizer creates a weekly recurring
training, months pass with real attendance recorded, then they delete the
series (e.g. team disbanding, or fixing an unrelated future date) with the
default series scope — every past training's attendance and comments are
permanently destroyed, silently, with no UI distinction between "delete
only future occurrences" and "delete everything, including history
already relied on for stats." This directly contradicts the design
principle `SetStatus` already encodes in the same file, and it's
irreversible (unlike a status flip, which could in principle be reverted).

## What Changes

- `DeleteEvent(scope=series)` only deletes occurrences dated today or
  later, mirroring `SetStatus`'s `date >= CURRENT_DATE` guard. The
  specific event addressed by `eventID` is still deleted individually
  regardless of its date (matching `SetStatus`'s existing precedent for
  the single-event case).
- `updateSeriesEvents` applies the same `date >= CURRENT_DATE` guard,
  for consistency with the other two series-wide mutations.
- The API/frontend series-delete confirmation makes clear that only
  future occurrences of the series are removed (the specific event
  itself, if in the past, is still removed) — matching how series
  cancellation is already communicated.

## Capabilities

### Modified Capabilities
- `events-scheduling`: series-wide deletion and update, like series-wide
  status changes, never affect already-held occurrences dated before
  today.

## Impact

- `backend/internal/events/repository.go` (`DeleteEvent`,
  `updateSeriesEvents`) + `repository_test.go`.
- `frontend/src/features/events/` — series-delete/edit confirmation
  copy, if it currently implies "everything" is removed/changed.
- No migration; no API shape change (only the effect of `scope=series`
  narrows).
