## Context

`events.Repository` has three series-wide mutations: `SetStatus`,
`DeleteEvent`, and `UpdateEvent`'s `updateSeriesEvents`. Only `SetStatus`
currently restricts its series-wide effect to `date >= CURRENT_DATE`,
with an explicit rationale comment. The other two silently touch past
occurrences too.

## Goals

- Make all three series-wide mutations consistent: none of them
  retroactively changes or removes an already-held occurrence's data.
- Preserve the existing behavior for the specific event addressed by
  `eventID` — it is still deleted/updated regardless of its own date,
  exactly as `SetStatus` already does for the single event it targets.
- Keep the fix minimal and mirror the existing `SetStatus` pattern
  rather than introducing a new mechanism.

## Decisions

- **Apply the same `date >= CURRENT_DATE` guard to both `DeleteEvent`
  and `updateSeriesEvents`.** This is the smallest change that restores
  consistency with `SetStatus`'s already-reviewed and commented
  precedent, rather than inventing a different cutoff or a
  confirmation-based override.
- **`updateSeriesEvents` is included, not just `DeleteEvent`.** Although
  update is less destructive than delete (no data is lost, just
  overwritten), the same "don't retroactively rewrite team history"
  principle applies — e.g. bulk-changing a series' location or nominated
  roles shouldn't silently apply to trainings that already happened and
  were attended under the old value. If product feedback later wants
  past-inclusive bulk edits for non-destructive fields (e.g. correcting
  a typo in a series title across all occurrences, including past ones),
  that would be a deliberate, separately-proposed opt-in — not the
  default.
- **No new API parameter.** `scope=series` keeps its existing meaning;
  only its effective date range narrows to match `SetStatus`. Clients
  don't need to change how they call the endpoint.
- **The `event_series` definition row is kept while past occurrences
  survive.** `events.series_id` is `ON DELETE SET NULL`, so deleting the
  series row alongside the future occurrences would leave the preserved
  past ones detached — still present, but no longer recognizable as part
  of a series. The row is therefore dropped only once nothing references
  it, which for the all-future case (the common one) is still the same
  single statement's worth of behavior as before.
- **`replaceEventTeamsForSeries` is deliberately left past-inclusive.**
  Re-targeting which teams a series is shared with is not destructive
  and not a rewrite of what happened — un-sharing in particular should
  plausibly withdraw access to past occurrences too, not just future
  ones. Narrowing it would also silently strip a removed team's access
  to future occurrences while leaving it on past ones, which is harder
  to reason about than the current all-or-nothing behavior. If this
  should change, it belongs in its own proposal alongside the wider
  question of historical cross-team visibility.
- **`excludeFromStats` is exempt from the guard.** It is the one field in
  a series-wide edit that is *about* history rather than a description of
  it: it decides whether an occurrence counts towards statistics, and the
  occurrences already held are precisely the ones statistics have counted.
  Narrowing it would mean a trainer correcting a mis-counted recurring
  event silently fixes only the occurrences that were never counted yet.
  It is also non-destructive and reversible, and a single occurrence can
  still override it afterwards.
- **The cutoff is a date, not a timestamp.** `date >= CURRENT_DATE` keeps
  an occurrence held earlier *today* in scope, so deleting a series on the
  evening of a training day still removes that training. This matches
  `SetStatus`'s long-standing behaviour rather than inventing a second
  rule, and the user-facing copy says "from today onwards" rather than
  "already took place" so the dialog does not overstate the guard.
  `CURRENT_DATE` is also the database server's date, which these
  deployments run in UTC, while the demo backend uses the browser's local
  date — for a European club the two disagree between local midnight and
  02:00. Worth revisiting together with the wider question of a club-local
  timezone, which this codebase does not model anywhere yet.
- **The addressed occurrence is always affected, whatever its date.** That
  follows `SetStatus`'s existing single-event precedent, and a caller who
  opened *that* event and chose delete plainly means it. It is the one way
  a series delete can still destroy recorded history, so the confirmation
  copy names it explicitly instead of promising blanket preservation.

- **The demo backend (MSW) is brought in line in the same change.**
  `serviceContract.test.ts` pins the demo backend as what `realApi` is
  expected to match, so leaving the mock past-inclusive would both keep
  the data loss reachable in demo mode and re-introduce the exact class
  of drift this repo has repeatedly had to fix. Its status handler had
  in fact already drifted from `SetStatus`'s long-standing guard.

## Risks

- **A caller relying on the current (undocumented) past-inclusive
  delete/update behavior** would see a behavior change. Given the
  destructive potential this fixes and that `SetStatus` already set the
  "future-only" precedent for this exact class of series operation, this
  is treated as a bug fix, not a breaking change requiring a deprecation
  window.
