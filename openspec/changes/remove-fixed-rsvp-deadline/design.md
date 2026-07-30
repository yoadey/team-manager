## Context

`event-cancellation-lead-time` (already implemented, archived-pending) added `cancelLeadMinutes` specifically because the pre-existing `rsvpDeadline` (absolute timestamp, added earlier for `inline-event-rsvp`) doesn't work well for recurring series and forces organizers to think in wall-clock dates instead of "closes N hours before". Both mechanisms are still live in the code (`effectiveRsvpCutoff` takes the earlier of the two), which is the redundancy this change removes.

## Goals / Non-Goals

**Goals:**
- One cutoff mechanism (`cancelLeadMinutes`) end to end: schema, DB, service enforcement, event form.
- Clearer hours/minutes input for the lead time that remains.

**Non-Goals:**
- Backfilling `cancelLeadMinutes` from existing `rsvpDeadline` values — not derivable correctly for series occurrences (a single stored absolute deadline doesn't decompose into "N minutes before this occurrence's start" for every future occurrence), so affected events simply end up with no cutoff post-migration.
- Any change to the lead-time cutoff's enforcement semantics (still: self-service rejected past cutoff, `events:write` overrides).

## Decisions

- Drop `rsvp_deadline` via a new goose migration (`ALTER TABLE ... DROP COLUMN`) rather than reusing/rewriting migration `00007`; migrations are append-only history here (mirrors how `00011` added `cancel_lead_minutes` as a new migration rather than editing `00007`).
- `effectiveRsvpCutoff`/`isRsvpCutoffPassed` (frontend) and the backend's attendance-mutation check collapse to the single `cancelLeadMinutes` branch. `ErrRsvpDeadlinePassed` is removed outright rather than repurposed: the backend already has a distinct `ErrCancelLeadTimePassed` sentinel for this cutoff (added by `event-cancellation-lead-time`), so the two errors were already redundant.
- Cancel-lead-time input: keep two numeric fields (hours, minutes — matches how `cancelLeadMinutes` round-trips through the form today) but give each its own visible label and unit suffix (e.g. "Std." / "Min.") instead of relying on a shared placeholder-only `Field` wrapper, so the pair reads as a duration rather than two anonymous numbers.

## Risks / Trade-offs

- **Breaking for any event that currently has `rsvpDeadline` set**: that event loses its cutoff outright (see Non-Goals). Flagged in the proposal for reviewer sign-off; acceptable because the feature is new enough that live data impact is expected to be minimal, and the alternative (a lossy heuristic backfill) is worse than an explicit no-cutoff state an organizer can immediately re-set via lead time.
- Migration is a `DROP COLUMN` (irreversible data loss on `up`); the `down` migration can only re-add the (now empty) column, not restore prior values — same class of risk already accepted for other destructive-down migrations in this repo.
