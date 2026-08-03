## Why

Creating an event from the calendar month view requires opening the "+" action in the page header, which always defaults the new event's date to today. A member who wants to schedule something for a day they're already looking at in the calendar has to create the event and then manually change the date field. Double-clicking the target day is a common calendar-UI shortcut and lets the date come pre-filled.

## What Changes

- Double-clicking an in-month day cell in `EventCalendar` opens the event-creation sheet with that day pre-selected as the event date, instead of defaulting to today.
- Gated the same way as the existing header "+" action: only available to members with `events:write`.
- Double-clicking an event/absence/birthday chip inside a cell still only triggers that chip's own single-click behavior (opens the event detail); it must not also open the create sheet for the day underneath it.
- `openEventForm` gains an optional pre-selected date parameter; the header "+" action keeps its existing today-default behavior unchanged.

## Capabilities

### New Capabilities
- `calendar-quick-create`: create an event pre-dated to a specific day via a calendar-day double-click.

## Impact

- Frontend only: `frontend/src/features/events/components/EventCalendar.tsx` (day-cell double-click handler), `frontend/src/features/events/hooks/useEventFormActions.ts` (`openEventForm` gains an optional date arg), `frontend/src/context/AppContext.tsx` (type signature), tests for both.
- No API/spec change — reuses the existing create-event payload and mutation.
- CI: frontend lint/typecheck/test/build + bundle budget. Low risk.
