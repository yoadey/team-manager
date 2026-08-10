## Context

Events store a single `date` (`DATE`) plus optional `meet_time`/
`start_time`/`end_time` (`TIME`-of-day, no date component) — every event is
assumed to occur within one calendar day
(`backend/internal/db/migrations/00001_init.sql:82-120`, `events` table's
`CHECK (events_end_after_start_time)`). Recurring series
(`event_series`/`series_id`) already have their own `endDate` field in
`CreateEventRequest`, meaning "the series' last occurrence date" — an
unrelated concept from a single occurrence spanning multiple days. There is
no existing duplicate/copy affordance anywhere in the app.

`absences` already has the precedent for a date range:
`from_date`/`to_date`, a `CHECK (to_date >= from_date)`, and
`groupAbsencesByDate` (`frontend/src/features/events/components/
EventCalendar.tsx`) already expands a range across every calendar day it
touches, DST-safely.

## Goals / Non-Goals

**Goals:**
- An organizer can mark an event as spanning multiple consecutive days.
- The calendar shows the event on every day it spans.
- An organizer can duplicate an event in one action instead of re-entering
  every field.

**Non-Goals:**
- Multi-day **recurring series** (each generated occurrence stays
  single-day). Combining both would require per-occurrence range math with
  no clear organizer use case yet; out of scope here.
- Server-side atomic duplication of an event's comments/attendance history
  — a copy is a *new* event with no attendance/comments, matching how a
  freshly created event starts empty.
- Duplicating a whole series — copy always produces one standalone event.
- A week/day calendar view (none exists today; only the month grid).

## Decisions

- **New field name `multiDayEndDate`**, not `endDate`: `CreateEventRequest.
  endDate` already means "series recurrence ends on this date." Reusing the
  name for a different concept on the same request body (a single request
  can carry `recurring: true` and a multi-day span in principle, even though
  we reject that combination — see below) would be ambiguous. `multiDayEndDate`
  is added to `TeamEvent`, `CreateEventRequest`, and `UpdateEventRequest`
  (none of which currently define an `endDate` for the occurrence itself,
  per `openapi.yaml:2584-2745`).
- **Mutually exclusive with `recurring`**: request validation
  (`handler.go`'s `validateEventFields`) rejects `multiDayEndDate` set
  together with `recurring: true` (400). This keeps the recurrence-expansion
  logic (`repository.go`'s `seriesDates`/`CreateSeries`) untouched.
- **DB**: nullable `end_date DATE` on `events` only (not `event_series`,
  consistent with the recurring exclusion above), with
  `CHECK (end_date IS NULL OR end_date >= date)`, mirroring absences'
  `to_date >= from_date` guard exactly. A second CHECK,
  `events_multiday_span_within_limit (end_date - date <= 1095)`, mirrors
  absences' identical `absences_span_within_limit` cap -- without it, an
  unbounded span would make every calendar render (which expands the event
  across every day it covers) do unbounded work for a single event.
  `Service.CreateEvent` enforces the same cap early for an immediate 400;
  the CHECK is the backstop for a partial update that only touches one of
  `date`/`multiDayEndDate`.
- **Clearing a multi-day span back to single-day**: `UpdateEventRequest`
  gets a separate `clearMultiDayEndDate: boolean`, mutually exclusive with
  `multiDayEndDate`. A plain optional date field can't itself distinguish
  "not provided" from "explicitly clear" without a schema-wide nullable/
  triple-state convention this codebase doesn't otherwise use, so a
  dedicated boolean flag (matching how e.g. `meetTimeMandatory` is already
  a plain boolean) is the minimal-diff way to make "make this a single-day
  event again" an actual, working action instead of a silently-ignored one.
- **Every "is this past?" UI check must use the effective end date, not
  `date` alone**: `EventDetailSheet.tsx`, `EventsPage.tsx`,
  `components/cards.tsx` (`EventCard`), `pages/Home.tsx`, and
  `layouts/AppShell.tsx` each independently classified an event as past/
  upcoming from `date` — mirroring the same bug `ListEvents` had before its
  `COALESCE(end_date, date)` fix, just client-side. A shared
  `isEventPast`/`eventEffectiveEndDate` helper
  (`features/events/rsvpCutoff.ts`) replaces every one of those inline
  comparisons, so an ongoing multi-day event stays "upcoming" (RSVP
  controls visible, included in next-events/pending counts, not dimmed)
  consistently everywhere, not just in the calendar and the backend list.
- **Upcoming/past index**: `ListEvents`' scope filter on
  `COALESCE(end_date, date)` isn't served by the existing
  `idx_events_team_date_id (team_id, date, id)` index as a sargable range
  condition, since the expression isn't a bare column -- a matching
  expression index (`idx_events_team_coalesce_enddate_id`, migration 00026)
  restores the seek instead of a per-team full scan.
- **Calendar feed / .ics export**: both the server-side feed
  (`internal/calendarfeed/ics.go`) and the client-side one-off export
  (`useCalExportActions.ts`) anchor `DTEND` to the event's last day
  (`multiDayEndDate`), not its first -- otherwise a multi-day event would
  export as a single ~2h block on day one only, with the remaining days it
  covers invisible in a subscribed calendar app.
- **Copy is client-side only, no new endpoint**: `Duplicate` builds an
  `EventFormValues` from the fetched source event (reusing
  `useEventFormActions.ts`'s existing `buildBasePayload`), strips
  `seriesId`/sets `recurring: false`, resets `date` (and `multiDayEndDate`,
  shifted by the same span length if the source was multi-day) to today,
  opens the form in create mode for the organizer to review/adjust, then
  submits through the existing `createEvent` POST. No backend change needed
  for duplication itself.
- **Upcoming/past listing** (`repository.go`'s `ListEvents`): filters on
  `COALESCE(end_date, date)` instead of `date` alone, so a multi-day event
  that has started but not finished stays "upcoming" rather than dropping
  into "past" the moment its first day passes. Keyset ordering/cursor still
  sorts on `date` (unchanged) — only the scope predicate changes.
- **Calendar rendering**: `groupEventsByDate` iterates `date` →
  `multiDayEndDate ?? date` inclusive per local calendar day (same increment
  pattern as `groupAbsencesByDate`), bucketing the event under every day it
  spans. Chips for a spanning day show a compact indicator (e.g. "(2/3)")
  rather than repeating the full title unmodified, so the day grid doesn't
  visually imply N separate events.

## Risks / Trade-offs

- Chip layout: a wide multi-day event drawn as repeated per-day chips
  (rather than a spanning bar) is simpler to implement in the existing month
  grid but reads less obviously as "one continuous thing" — accepted for
  this change; a true spanning-bar view is a follow-up if organizers want it.
- `end_date >= date` still allows a single-day event to set
  `multiDayEndDate = date` (degenerate span of one day) — harmless, but the
  UI should just leave the field empty for single-day events rather than
  round-tripping a value equal to `date`.
- Existing rows get `NULL` for `end_date` — no behavior change until an
  organizer opts in.
