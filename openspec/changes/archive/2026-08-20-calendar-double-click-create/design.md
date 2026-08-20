## Context

`EventCalendar.tsx` renders 42 `CalendarDayCell`s per month grid. The event-creation sheet (`EventFormSheet`) is opened via `app.openEventForm(event: TeamEvent | null)`, which builds `formInitial` in `useEventFormActions.ts`; the `null`/create branch currently always defaults `date: todayStr()`. The page-header "+" action (`pageMeta.ts`) calls `app.openEventForm(null)` and is gated on `app.can('events', 'write')`.

## Goals / Non-Goals

**Goals:**
- Double-click on a day cell opens the create sheet with that day's date pre-filled.
- Reuse the existing create sheet/mutation; no new endpoint or payload shape.
- Don't let a double-click on a chip inside the cell (event/absence/birthday) also fire the day's create handler.
- Expose a keyboard equivalent for the same action, so the day-cell affordance isn't mouse-only despite visually signaling interactivity (`cursor: pointer` + hover).

**Non-Goals:**
- No change to the trailing/leading (out-of-month) cell behavior beyond what double-click gating already implies (see Decisions).

## Decisions

- `openEventForm` gains an optional second parameter: `openEventForm(event: TeamEvent | null, initialDate?: string)`. When `event` is `null` and `initialDate` is provided, the create-mode default uses it instead of `todayStr()`. The edit branch and the header "+" call site (`pageMeta.ts`) are unaffected since they don't pass the new argument.
- `CalendarDayCell` gets an `onDoubleClick` prop, wired only for in-month cells (`inMonth`) and only when the caller has `events:write` — out-of-month/no-permission cells render without the handler so double-clicking them does nothing, matching the header action's own gating.
- `EventChip`'s `ButtonBase` stops the double-click from bubbling (`onDoubleClick={(e) => e.stopPropagation()}`) so double-clicking an event opens/keeps just the event, not also the day's create sheet. `AbsenceChip`/`BirthdayChip` aren't clickable targets, but a double-click landing on one still bubbles by default, so they get the same `stopPropagation` treatment — otherwise double-clicking a name/pill inside a day would surprisingly open the day's create sheet instead of being a no-op.
- Pass the day's `formatDateOnly(date)` string (already computed in the render loop as `ds`) straight through — same format the schema/date input already expects.
- The cell also gets `role="button"`, `tabIndex={0}`, and an `aria-label` (`events.calCreateEventOnDay`, interpolating `fmtDateLong`) plus an `onKeyDown` handler treating Enter/Space the same as the double-click — all only when `onCreateEvent` is set, so out-of-month/no-permission cells stay out of the tab order and unlabelled, exactly like they get no pointer handler.

## Risks / Trade-offs

- Double-click is not discoverable without a hint; mitigated by `cursor: pointer` + a subtle hover/focus-visible background on writable in-month cells, consistent with other clickable surfaces in the app.
- Double-clicking selects text in some browsers by default; mitigated with `sx={{ userSelect: 'none' }}` on the cell, matching the existing non-selectable feel of the chips.
- A `role="button"` div wrapping other real `<button>` chips (event chips) is an unusual nesting for assistive tech to announce; accepted here as a minimal, incremental fix rather than redesigning the grid with `role="grid"`/`gridcell` semantics.
