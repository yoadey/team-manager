## Why

Organizers currently have two related gaps when scheduling trainings/events:

1. **No way to duplicate an event.** A recurring tournament weekend, a
   repeated one-off friendly, or "same training next month" all require
   re-entering every field from scratch in `EventFormSheet.tsx` — there is no
   duplicate/copy affordance anywhere in `EventDetailSheet.tsx`'s
   `EventEditActions` (only Edit/Cancel/Delete).
2. **No way to model a multi-day event.** `events.date`
   (`backend/internal/db/migrations/00001_init.sql:82-120`) is a single
   `DATE` column; `meet_time`/`start_time`/`end_time` are `TIME`-of-day only.
   A training camp, tournament weekend, or multi-day trip has no
   representation — organizers currently work around this with several
   separate single-day events, which fragments attendance/comments across
   entries that are really one thing. `absences` already models a date range
   (`from_date`/`to_date`, `backend/internal/absences/model.go:14-15`); events
   have no equivalent.

## What Changes

- Add an optional **`multiDayEndDate`** to events: when set, the event spans
  from `date` to `multiDayEndDate` inclusive, rather than occurring on a
  single day. Restricted to non-recurring events (a recurring series stays
  single-day per occurrence — see Non-Goals in `design.md`).
- The calendar (`EventCalendar.tsx`) renders a multi-day event's chip on
  every day it spans, mirroring how `groupAbsencesByDate` already expands an
  absence's `from`/`to` range across calendar days.
- Add a **Duplicate** action on an event's detail sheet
  (`EventDetailSheet.tsx`'s `EventEditActions`) that opens the create form
  pre-filled from the source event, with `seriesId`/`recurring` stripped (a
  copy always becomes one new standalone event, never a series) and the date
  reset to today (or the clicked calendar day), for the organizer to adjust
  before saving.

## Capabilities

### New Capabilities
- `event-duplication`: duplicate an existing event into a new standalone
  event pre-filled from the source.

### Modified Capabilities
- `events-scheduling`: an event may optionally span multiple consecutive
  days via `multiDayEndDate`.

## Impact

- Spec/backend: `openapi.yaml` (`TeamEvent`, `CreateEventRequest`,
  `UpdateEventRequest` schemas), migration adding nullable `end_date` to
  `events`, `internal/events/{model,repository,service,handler}.go` +
  tests; regenerate `internal/gen` + `frontend/src/api/*`.
- Frontend: `EventFormSheet.tsx` + `eventFormSchema.ts` (end-date field,
  validation, disabled when recurring), `useEventFormActions.ts` (copy
  variant of `openEventForm`), `EventDetailSheet.tsx` (Duplicate button),
  `EventCalendar.tsx` (`groupEventsByDate` range expansion + multi-day chip
  rendering), `frontend/src/i18n/{de,en}.ts`, MSW handlers
  (`frontend/src/mocks/`).
- CI: openapi-drift, migration-rollback/safety, backend + frontend gates.
  **API + migration-affecting.**
