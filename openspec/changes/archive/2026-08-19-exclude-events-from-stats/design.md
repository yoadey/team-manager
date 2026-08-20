## Context

`events` and `event_series` already share several fields that are set at
series-creation time and seeded onto every generated occurrence, then
independently editable per-occurrence afterward — `cancel_lead_minutes` is
the direct precedent (`backend/internal/events/model.go`,
`repository.go`'s `CreateSeries` ~L424, and the `scope` (`single`|`series`)
parameter on update/status-change/delete, L585-787). `excludeFromStats`
follows that exact shape: no new scoping mechanism is needed, only a new
boolean column treated the same way.

## Goals / Non-Goals

**Goals:**
- An event flagged `excludeFromStats` is invisible to every statistics
  query, with no other behavior change (RSVP, comments, notifications,
  cancellation all work exactly as before).
- Setting the flag on a series (scope="series") applies it to every
  occurrence of that series, mirroring exactly how `cancel_lead_minutes`
  and every other series-wide-editable field already behave (no date
  filtering) — but any single occurrence can still be toggled
  independently afterward (scope="single").
- Zero behavior change for any existing event or series (column defaults
  to `false`).

**Non-Goals:**
- No new event "type" (the existing `training`/`auftritt`/`event` enum is
  unchanged) — a GL-Training is still typed `training` and merely flagged,
  since introducing a fourth type would require every consumer of
  `EventType` (matrix type filter, i18n labels, event list icons) to learn
  a new case for something that is purely a statistics-visibility concern,
  not a different kind of event.
- No team-level default ("all trainings of type X are excluded by
  default") — each event/series opts in individually. A team wanting many
  GL-Trainings excluded creates them as a series once, which already gives
  one flag flip covering all future occurrences.

## Decisions

**Column lives on both `events` and `event_series`, mirroring
`cancel_lead_minutes`.** The series-template column is the source of truth
for what gets seeded onto new occurrences and what a scope=`series` edit
updates; the per-event column is what stats queries actually filter on.
This avoids inventing a second inheritance mechanism beyond the one that
already exists for series-templated fields.

**Stats queries filter on the occurrence's own column, never the series
column.** `stats/repository.go` never joins `event_series` today (it works
entirely off `events`), and there is no reason to start — a per-occurrence
override already exists by construction (the occurrence's own column is
what a scope=`single` edit changes), so filtering there is both sufficient
and consistent with how every other per-event stats-relevant field
(`status`, `date`) is already read.

**Excluded events are excluded from `EventStats` too, not just member/
matrix queries.** Unlike the member-exclusion feature (a separate,
independent change) where an excluded *member*'s historical attendance
should still count toward an event's own turnout number, an excluded
*event* has no legitimate turnout number to show in the stats section at
all — it's not part of "the team's statistics" by definition. It remains
fully visible in the normal event list/detail with its own attendance
summary; only the stats-specific aggregations drop it.

## Risks

- **None database-destructive**: purely additive columns, default `false`,
  no backfill needed.
- Series scope semantics must exactly match the existing `cancel_lead_minutes`
  precedent so users don't encounter two different mental models for "this
  field applies to the whole series vs. just this occurrence" within the
  same form.
