## Why

Accepting/declining an event currently requires opening the event detail sheet (`EventDetailSheet.tsx`). In the events overview a member can't RSVP without drilling in, which is friction for the most common action. Members want to accept/decline straight from the list — compact icon controls are enough.

## What Changes

- Add inline RSVP icon controls (accept / maybe / decline) to each event row in the events overview, reflecting and updating the member's own attendance.
- Reuse the existing set-attendance mutation and optimistic-update path; no new endpoint.
- Respect the same rules as the detail view: disable once the cancellation cutoff has passed (see `event-cancellation-lead-time`) and reflect the current status.

## Capabilities

### New Capabilities
- `event-rsvp`: set one's own attendance directly from the events overview.

## Impact

- Frontend only: the events overview list/row component, wiring to `frontend/src/features/events/hooks/useEventActions.ts` / `useAbsenceActions.ts` / `useEventMutations.ts`, shared status icons, `frontend/src/i18n/{de,en}.ts` (aria-labels), tests, MSW-backed.
- Reuses the existing `POST .../attendance` contract — no API/spec change.
- CI: frontend lint/typecheck/test/build + bundle budget. Low risk.
