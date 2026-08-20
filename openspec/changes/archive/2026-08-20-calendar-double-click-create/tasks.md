## 1. Frontend

- [x] 1.1 `useEventFormActions.ts`: `openEventForm` accepts an optional `initialDate?: string`; create-mode default uses it (falls back to `todayStr()` when omitted)
- [x] 1.2 `AppContext.tsx`: update the `openEventForm` type signature to match
- [x] 1.3 `EventCalendar.tsx`: `CalendarDayCell` gets an `onDoubleClick` handler wired to `app.openEventForm(null, ds)`, only for in-month cells and only when `app.can('events', 'write')`; add `cursor: pointer` + hover/focus-visible affordance + `userSelect: 'none'` on writable in-month cells
- [x] 1.4 `EventChip`'s double-click stops propagation so double-clicking an event doesn't also trigger the day's create sheet
- [x] 1.5 `AbsenceChip`/`BirthdayChip` also stop double-click propagation, so double-clicking one of those pills doesn't open the day's create sheet either
- [x] 1.6 Keyboard equivalent: writable in-month cells get `role="button"`, `tabIndex={0}`, an `aria-label` (new `events.calCreateEventOnDay` key, de/en) built from `fmtDateLong`, and an `onKeyDown` handler triggering the same action on Enter/Space

## 2. Tests

- [x] 2.1 `useEventFormActions.test.ts`: `openEventForm(null, '2026-05-01')` produces `formInitial.date === '2026-05-01'`; `openEventForm(null)` still defaults to today
- [x] 2.2 `EventCalendar.test.tsx`: double-clicking an empty in-month day cell calls `app.openEventForm(null, <that day's date>)`; double-clicking without `events:write` does not call it; double-clicking an event/absence/birthday chip does not call it (only the chip's own single-click, where applicable)
- [x] 2.3 `EventCalendar.test.tsx`: a writable in-month cell has `role="button"` and an `aria-label` mentioning the day; pressing Enter or Space calls `app.openEventForm(null, ds)`; a cell without `events:write` has neither the role nor a `tabindex`

## 3. Verification

- [x] 3.1 `npm run lint` (0 errors) + `npm run typecheck` green
- [x] 3.2 `npm test` green
- [x] 3.3 `npm run build` + `check:bundle` within budget
