## 1. Poll option labels (fastest — do first)

- [ ] 1.1 `PollFormSheet.tsx`: give each option `TextInput` an
      `aria-label={t('polls.optionN')}` (or wrap in `Field`) so the
      label persists once typed

## 2. Calendar keyboard navigation

- [ ] 2.1 `EventCalendar.tsx`: mark the month grid `role="grid"`, day
      cells `role="gridcell"`
- [ ] 2.2 Add arrow-key navigation between day cells (left/right/up/down
      move focus by day/week), Home/End for week start/end if
      proportionate
- [ ] 2.3 Existing mouse/tap day interaction and event-chip focusability
      unchanged

## 3. Shared accessible TabBar

- [ ] 3.1 Extract a `TabBar` component implementing the WAI-ARIA APG
      tabs pattern: roving `tabindex` (only the active tab is a Tab
      stop), arrow-key switching between tabs, `role="tablist"`/
      `role="tab"`/`aria-selected` on triggers, `role="tabpanel"`/
      `aria-controls`/`aria-labelledby` wiring content to its trigger
- [ ] 3.2 `FinancesPage.tsx`: replace the hand-rolled tab implementation
      with `TabBar`
- [ ] 3.3 `EventsPage.tsx`: same
- [ ] 3.4 Polls' equivalent tab implementation: same
- [ ] 3.5 Visual/behavioral parity check against the current
      implementations (active-tab styling, content switching) in each
      adopting page

## 4. Verification

- [ ] 4.1 `npm run lint`
- [ ] 4.2 `npm run typecheck`
- [ ] 4.3 `npm test`
- [ ] 4.4 Manual keyboard-only pass: tab into each adopting page's
      tablist, switch tabs with arrow keys only; tab into the poll
      form, confirm option labels are announced; tab into the calendar,
      navigate days with arrow keys
