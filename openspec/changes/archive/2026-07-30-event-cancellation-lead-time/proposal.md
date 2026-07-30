## Why

Whether/when a member may still decline (or change RSVP) for an event should be configurable per event, but organizers think in **lead time** ("registration closes 24 h / 90 min before start"), not an absolute wall-clock timestamp. An absolute timestamp is awkward for recurring/series events and has to be recomputed for every occurrence. Event creation currently has no such deadline concept at all.

## What Changes

- Add an optional **cancellation/RSVP-change deadline expressed as a lead time before event start** — a duration in hours and minutes — to events (and event series, applying per occurrence).
- Enforce it server-side: once `startTime - leadTime` has passed, attendance changes for that event are rejected (organizers with write on `events` may still override).
- Surface it in the event form (hours + minutes inputs) and show the effective cutoff on the event detail, disabling the RSVP controls after it.

## Capabilities

### New Capabilities
- `event-cancellation`: a per-event RSVP/cancellation cutoff defined relative to event start.

## Impact

- Spec/backend: `openapi.yaml` event + series schemas (new `cancelLeadMinutes` field), create/update requests, `SetAttendanceRequest` enforcement; regenerate `internal/gen` + `frontend/src/api/*`; `internal/events` service/repository, a migration adding the column; tests.
- Frontend: event form (`useEventFormActions`), `EventDetailSheet.tsx` RSVP controls, `useAbsenceActions`/`useEventActions`, `frontend/src/i18n/{de,en}.ts`, MSW handlers.
- CI: openapi-drift, migration-rollback/safety, backend + frontend gates. **API + migration-affecting.**
