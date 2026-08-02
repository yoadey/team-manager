## Why

Three related accessibility gaps surfaced during review, all in
interactive navigation/form elements:

1. **Duplicated, incomplete ARIA-tabs implementation.**
   `frontend/src/features/finances/FinancesPage.tsx:70-104`,
   `frontend/src/features/events/EventsPage.tsx:32-75`, and the
   equivalent in Polls each hand-roll `role="tablist"`/`role="tab"` +
   `aria-selected`, but every tab stays independently focusable (no
   roving `tabindex`, no arrow-key switching) and content isn't wired
   via `role="tabpanel"`/`aria-controls`. A screen-reader or
   keyboard-only user gets tab *semantics* announced without the
   interaction model assistive tech expects.
2. **Poll options rely on placeholder-only labels.**
   `features/polls/components/PollFormSheet.tsx:106-121` — the four
   option `TextInput`s only have `placeholder`, no `aria-label`/
   associated label. Placeholder text disappears once typed and isn't
   reliably read as a label by all assistive tech.
3. **Calendar day grid has no keyboard navigation of days.**
   `features/events/components/EventCalendar.tsx:141-209,263-281` — only
   event chips inside a day are focusable; day cells aren't part of a
   `role="grid"` with arrow-key traversal, so month prev/next buttons are
   the only keyboard-accessible way to browse dates.

## What Changes

- Extract one shared `TabBar` component implementing the WAI-ARIA APG
  tabs pattern (roving `tabindex`, arrow-key switching,
  `role="tabpanel"`/`aria-controls` wiring) and adopt it in
  Finances/Events/Polls, replacing their independent hand-rolled
  versions.
- Give each poll option input an `aria-label` (or wrap in the existing
  `Field` component already used elsewhere for consistent
  `aria-required`/`aria-invalid` wiring).
- Add `role="grid"`/`role="gridcell"` with arrow-key day-to-day
  navigation to `EventCalendar`'s month view, keeping existing
  mouse/tap interaction unchanged.

## Capabilities

### Added Capabilities
- `accessibility`: tabbed navigation follows the WAI-ARIA APG tabs
  pattern consistently; form inputs have a persistent accessible label,
  not only a placeholder; the calendar's day grid is keyboard-navigable.

## Impact

- New shared component, e.g. `frontend/src/components/TabBar.tsx`.
- `frontend/src/features/finances/FinancesPage.tsx`,
  `frontend/src/features/events/EventsPage.tsx`,
  `frontend/src/features/polls/PollsPage.tsx` (or equivalent) — adopt
  `TabBar`.
- `frontend/src/features/polls/components/PollFormSheet.tsx`.
- `frontend/src/features/events/components/EventCalendar.tsx`.
