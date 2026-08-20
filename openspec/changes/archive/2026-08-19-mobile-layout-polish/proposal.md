## Why

Two mobile layout problems:

1. **Bottom controls hidden behind the browser chrome.** On mobile browsers the address/toolbar overlaps the bottom of the viewport, covering the bottom navigation / action symbols in `frontend/src/layouts/AppShell.tsx` (and any bottom-fixed UI). The layout doesn't account for the browser UI / safe-area inset.
2. **Calendar is cramped on mobile.** `frontend/src/features/events/components/EventCalendar.tsx` wastes horizontal/vertical space: gaps between day cells and large rounded corners shrink the usable area for day entries on small screens.

## What Changes

- Respect the mobile browser UI and device safe area: use dynamic viewport units (`100dvh`) and `env(safe-area-inset-bottom)` padding so bottom nav/actions are never occluded.
- On mobile, tighten the calendar: remove inter-day gaps and reduce day-cell corner radius so each day cell gets more room for its entries; keep the roomier desktop styling above the mobile breakpoint.

## Capabilities

### New Capabilities
- `mobile-layout`: mobile-viewport-correct chrome-safe layout and denser calendar.

## Impact

- Frontend only: `frontend/src/layouts/AppShell.tsx` (bottom nav safe-area + `dvh`), possibly `SheetHost.tsx`/`Toast.tsx` bottom offsets, `frontend/src/features/events/components/EventCalendar.tsx` (gap + radius at the mobile breakpoint), related styles/tokens; tests (`EventCalendar.test.tsx`), e2e accessibility spec unaffected.
- CI: frontend lint/typecheck/test/build + bundle budget. Low risk, no API change.
