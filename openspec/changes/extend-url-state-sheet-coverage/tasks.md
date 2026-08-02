## 1. URL state model

- [ ] 1.1 `urlState.ts`: extend `UrlState` to represent `teamSettings`,
      `roles`, `calendarShares`, `sharedCalendars` (no id) and
      `roleForm` (with a role id)
- [ ] 1.2 `buildPath`: emit the corresponding path/query for each new
      sheet state
- [ ] 1.3 `parseLocation`: parse each corresponding path/query back into
      the sheet state; fall back to the underlying route (not an error)
      for a stale/invalid id

## 2. Wiring

- [ ] 2.1 `AppContext.tsx`: sheet open/close actions for the five newly
      covered types read/write `UrlState` the same way `eventDetail`/
      `memberDetail` already do
- [ ] 2.2 `PAGE_SHEETS`'s comment (or a new one near it) documents that
      `eventForm`/`memberForm` are deliberately excluded from URL
      coverage, and why

## 3. Tests

- [ ] 3.1 `urlState.test.ts` (or equivalent): round-trip `buildPath` →
      `parseLocation` for each newly covered sheet type
- [ ] 3.2 A refresh/direct-navigation test confirming each newly covered
      sheet reopens from its URL

## 4. Verification

- [ ] 4.1 `npm run typecheck`
- [ ] 4.2 `npm test`
- [ ] 4.3 `npm run lint`
