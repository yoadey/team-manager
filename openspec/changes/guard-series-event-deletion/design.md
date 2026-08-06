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

## Risks

- **A caller relying on the current (undocumented) past-inclusive
  delete/update behavior** would see a behavior change. Given the
  destructive potential this fixes and that `SetStatus` already set the
  "future-only" precedent for this exact class of series operation, this
  is treated as a bug fix, not a breaking change requiring a deprecation
  window.
