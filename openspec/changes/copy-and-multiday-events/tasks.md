## 1. Spec

- [x] 1.1 Add optional `multiDayEndDate` (date) to `TeamEvent`,
      `CreateEventRequest`, and `UpdateEventRequest` in `openapi.yaml`,
      documented as mutually exclusive with `recurring`
- [x] 1.2 Run `cd backend && make generate` + repo-root `make generate-ts`;
      commit generated output

## 2. Migration + backend

- [x] 2.1 Migration: add nullable `end_date DATE` to `events`, with
      `CHECK (end_date IS NULL OR end_date >= date)`
- [x] 2.2 `internal/events/model.go`: add `EndDate *time.Time` to
      `EventRow`, `CreateEventParams`, `UpdateEventParams`
- [x] 2.3 `internal/events/repository.go`: add `end_date` to
      `selectEventFields`/`scanEventRow`, `CreateEvent`, `UpdateEvent`
      (rejecting `multiDayEndDate` on an event belonging to a series, and
      excluding it from `updateSeriesEvents` same as `date`); `ListEvents`
      filters past/upcoming on `COALESCE(end_date, date)`
- [x] 2.4 `internal/events/service.go`/`handler.go`: reject
      `multiDayEndDate` set together with `recurring: true` on create, and
      `multiDayEndDate < date` on create (mirroring absences' from/to
      validation); the DB CHECK constraint is the backstop for partial
      updates
- [x] 2.5 `internal/events/service.go`: `toGenEvent` maps `EndDate` to/from
      `multiDayEndDate`
- [x] 2.6 `repository_test.go`/`service_test.go`/`handler_test.go`: cover
      create/update/get/list round-tripping `multiDayEndDate`, the
      recurring+multiDayEndDate rejection, and `multiDayEndDate < date`
      rejection

## 3. Frontend — multi-day events

- [x] 3.1 `eventFormSchema.ts`: optional `multiDayEndDate` field,
      `>= date` validation, cleared/disabled when `recurring` is on
- [x] 3.2 `EventFormSheet.tsx`: end-date input next to the existing date
      field (mirroring `AbsenceFormSheet.tsx`'s from/to fields), disabled
      while `RecurringSection` is active
- [x] 3.3 `useEventFormActions.ts`: `buildBasePayload` includes
      `multiDayEndDate`; `openEventForm` populates it from an existing event
- [x] 3.4 `EventCalendar.tsx`: `groupEventsByDate` expands `date` →
      `multiDayEndDate ?? date` inclusive across local calendar days
      (mirroring `groupAbsencesByDate`); chip shows a day-N-of-M indicator
      on spanning days
- [x] 3.5 `frontend/src/i18n/{de,en}.ts`: end-date field label/validation
      strings, multi-day chip indicator string
- [x] 3.6 `frontend/src/mocks/`: MSW handlers/db persist and validate
      `multiDayEndDate` the same way the real backend does

## 4. Frontend — duplicate event

- [x] 4.1 `useEventFormActions.ts`: new `duplicateEvent(event, initialDate?)` —
      builds `EventFormValues` from `event`, strips `seriesId`, forces
      `recurring: false`, resets `date`/`multiDayEndDate` to today (shifting
      a multi-day span by the same length), opens the form in create mode
- [x] 4.2 `EventDetailSheet.tsx`: `EventEditActions` gets a `Duplicate`
      button (requires the same `write`-on-`events` permission as `Edit`),
      wired to `duplicateEvent`
- [x] 4.3 `frontend/src/i18n/{de,en}.ts`: "Duplicate"/"Kopieren" strings

## 4a. Backend — span cap, clearing, and calendar feed follow-ups

- [x] 4a.1 `maxMultiDaySpanDays` (1095) cap, mirroring absences'
      `maxAbsenceSpanDays`: early rejection in `Service.CreateEvent`
      (`ErrMultiDaySpanTooLong`) plus a DB CHECK
      (`events_multiday_span_within_limit`) as the partial-update backstop
- [x] 4a.2 `clearMultiDayEndDate` (boolean, `UpdateEventRequest`), mutually
      exclusive with `multiDayEndDate`: `UpdateEventParams.ClearEndDate`,
      `buildEventUpdateSets` sets `end_date = NULL`, frontend `update()`
      translates an empty `multiDayEndDate` into this flag so blanking the
      field in the edit form actually persists
- [x] 4a.3 `internal/calendarfeed/ics.go` + `useCalExportActions.ts`:
      DTEND anchored to `multiDayEndDate` (the event's last day), not
      `date`, for both the server-side feed and the client-side .ics export

## 4b. Follow-up correctness/perf fixes

- [x] 4b.1 `EventFormSheet.tsx`: toggling recurring on clears the (now
      unmounted, not unregistered) `multiDayEndDate` field's value, so a
      previously-typed value can no longer fail eventFormSchema's mutual-
      exclusion check invisibly and block submission with no visible error
- [x] 4b.2 Migration 00025's two new CHECK constraints use `NOT VALID` +
      a follow-up `VALIDATE CONSTRAINT` (matches this repo's migration-
      safety lint requirement for `ADD CONSTRAINT ... CHECK` on an
      existing table)
- [x] 4b.3 Migration 00026: `idx_events_team_coalesce_enddate_id` on
      `(team_id, (COALESCE(end_date, date)), id)`, restoring a sargable
      index for `ListEvents`' upcoming/past scope filter

## 4c. Frontend "is this event past?" consistency

- [x] 4c.1 `features/events/rsvpCutoff.ts`: shared `eventEffectiveEndDate`/
      `isEventPast` helpers (multiDayEndDate when set, else date)
- [x] 4c.2 Replace every inline `e.date < today` / `e.date >= today` past/
      upcoming check with the shared helper: `EventDetailSheet.tsx` (both
      `MyResponseSection` and the main detail body), `EventsPage.tsx`,
      `components/cards.tsx` (`EventCard`), `pages/Home.tsx` (next-events,
      pending count, upcoming stat), `layouts/AppShell.tsx` (nav pending
      badge)

## 5. Verification

- [x] 5.1 openapi-drift green (regenerated clients committed)
- [x] 5.2 Backend: `make test-unit`, `make test-integration`, `make lint`,
      migration-rollback (up→down→up) + migration-safety (nullable ADD
      COLUMN, no unsafe DDL) green
- [x] 5.3 Frontend: `npm run lint`, `npm run typecheck`, `npm test`,
      `npm run build` + bundle budget green
