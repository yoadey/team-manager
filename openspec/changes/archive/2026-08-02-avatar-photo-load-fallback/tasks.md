## 1. Shared avatar fallback
- [x] 1.1 `Av` (`frontend/src/components/ui.tsx`): preload `photo` via `new Image()`, track load failure in state, reset on `photo` change, and fall back to the colored-initials render when `!photo || failed`.
- [x] 1.2 `MyAvatar` (`frontend/src/layouts/AppShell.tsx`): reimplemented on top of `Av` instead of duplicating the background-image markup.

## 2. Tests
- [x] 2.1 `ui.test.tsx`: a photo URL that fails to load falls back to initials (mocked `Image` that synchronously fires `onerror`).
- [x] 2.2 Existing `Root.test.tsx`/`SheetHost.test.tsx`/`AppContext.test.tsx` (which exercise the shell/profile sheet) still pass unchanged.

## 3. Verification
- [x] 3.1 `npm run lint`
- [x] 3.2 `npm run typecheck`
- [x] 3.3 `npm test` (1308 tests, all passing)
- [x] 3.4 `npm run build` + `check:bundle` (271.9 KB total gzipped, within 600 KB budget)
