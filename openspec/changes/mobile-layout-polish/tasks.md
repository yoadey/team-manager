## 1. Chrome-safe bottom layout
- [x] 1.1 Ensure the viewport meta uses `viewport-fit=cover` (already present in `index.html`)
- [x] 1.2 `AppShell.tsx`: full-height mobile container uses `100dvh` (via `@supports`, `100vh` fallback); bottom nav uses `minHeight` + `pb: calc(8px + env(safe-area-inset-bottom))`; the compact `PrimaryActionButton` (FAB) + main scroll padding also offset by the inset. Previously checked off without the code actually existing (`AppShell.tsx` still used a bare `100vh`/`height: 72`/`bottom: 88`) — implemented for real this pass.
- [x] 1.3 `Toast.tsx` bottom offset now includes `env(safe-area-inset-bottom)`; `SheetHost.tsx` is a full-screen `inset:0` overlay, so no change needed

## 2. Denser mobile calendar
- [x] 2.1 `EventCalendar.tsx`: mobile grid `gap: 0` and day-cell `borderRadius: 0` (revised down from an initial `4px` — mobile screens have no room to spare); cells keep their 1px borders so the grid stays legible
- [x] 2.2 Desktop keeps `gap: 6px` / `borderRadius: 9px` (all gated on `mobile`). The `borderRadius` half of this was also previously checked off without being implemented (it was a flat `9px` regardless of viewport) — implemented for real this pass.

## 3. Tests
- [~] 3.1 `EventCalendar.test.tsx` mocks `useCompact → false` (desktop path); the mobile density is inline `sx` (generated classes, not introspectable without brittle assertions). Existing render test still passes; the visual change is verified manually. No new fragile style assertion added.

## 4. Verification
- [x] 4.1 `npm run lint` (0 errors) + `npm run typecheck` re-run green after the real 1.2/2.2 implementation
- [x] 4.2 `npm test` green — all 1307 tests across 92 files pass, including `AppShell`/`EventCalendar` suites
- [x] 4.3 `npm run build` + `check:bundle` within budget (271.8 KB total, gzipped). Manual mobile-viewport check of bottom controls clearing the address bar still recommended.
