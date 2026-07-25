## 1. Chrome-safe bottom layout
- [x] 1.1 Ensure the viewport meta uses `viewport-fit=cover` (already present in `index.html`)
- [x] 1.2 `AppShell.tsx`: full-height mobile container uses `100dvh` (via `@supports`, `100vh` fallback); bottom nav uses `minHeight` + `pb: calc(8px + env(safe-area-inset-bottom))`; FAB + main scroll padding also offset by the inset
- [x] 1.3 `Toast.tsx` bottom offset now includes `env(safe-area-inset-bottom)`; `SheetHost.tsx` is a full-screen `inset:0` overlay, so no change needed

## 2. Denser mobile calendar
- [x] 2.1 `EventCalendar.tsx`: mobile grid `gap: 0` and day-cell `borderRadius: 4px`; cells keep their 1px borders so the grid stays legible
- [x] 2.2 Desktop keeps `gap: 6px` / `borderRadius: 9px` (all gated on `mobile`)

## 3. Tests
- [~] 3.1 `EventCalendar.test.tsx` mocks `useCompact → false` (desktop path); the mobile density is inline `sx` (generated classes, not introspectable without brittle assertions). Existing render test still passes; the visual change is verified manually. No new fragile style assertion added.

## 4. Verification
- [x] 4.1 `npm run lint` (0 errors) + `npm run typecheck` green
- [x] 4.2 `npm test` green — cards + EventCalendar suites pass; the 5 intermittent `a11y.test.tsx` failures are axe timeouts under saturated parallel load (setup ~378s), pass 11/11 in isolation, and render mocked/empty content that never mounts the changed components
- [x] 4.3 `npm run build` + `check:bundle` within budget (254.4 KB total). Manual mobile-viewport check of bottom controls clearing the address bar still recommended.
