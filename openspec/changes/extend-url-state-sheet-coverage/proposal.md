## Why

`frontend/src/context/urlState.ts`'s `UrlState.detail` only models two
sheet kinds — `event` and `member` (`urlState.ts:60-87`). But
`AppContext.tsx`'s `PAGE_SHEETS` list (`AppContext.tsx:258-269`) also
treats `teamSettings`, `roles`, `roleForm`, `calendarShares`,
`sharedCalendars`, `eventForm`, and `memberForm` as page-level sheets.
Opening Team Settings (or any of the others) and refreshing the page, or
sharing/bookmarking the URL, silently drops back to the underlying route
instead of restoring the sheet — the URL never reflected it in the first
place.

## What Changes

- Extend `UrlState.detail` (or an equivalent structure) to cover the
  remaining `PAGE_SHEETS` types that make sense as a URL-addressable
  state: `teamSettings`, `roles`, `roleForm` (with its target role id),
  `calendarShares`, `sharedCalendars`. `eventForm`/`memberForm` (create/
  edit forms) are evaluated case by case — deep-linking into a
  half-filled create form is lower value and may be deliberately
  excluded; document the decision in `design.md`.
- Extend `buildPath`/`parseLocation` accordingly.
- Where a sheet type is deliberately excluded from URL coverage, update
  the code comment on `PAGE_SHEETS` (or an equivalent marker) so it's
  explicit which sheets are deep-linkable and which aren't, rather than
  leaving the gap implicit.

## Capabilities

### Added Capabilities
- `state-based-routing`: page-level sheets that represent navigable
  destinations (not transient create/edit forms) are reflected in the
  URL and restorable on refresh or via a shared link.

## Impact

- `frontend/src/context/urlState.ts` (+ tests).
- `frontend/src/context/AppContext.tsx` (`PAGE_SHEETS` and whatever
  reads/writes `UrlState` alongside sheet open/close).
- No backend changes.
