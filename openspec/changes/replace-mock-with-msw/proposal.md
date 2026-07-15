## Why

The frontend maintains two parallel implementations of the API contract: `serviceLayerReal.ts` (real HTTP via the generated `openapi-fetch` client) and `serviceLayer.ts` (`_mockApi`, ~2098 lines re-implementing the entire domain logic in localStorage). They drift — stats calculation, penalty-amount snapshotting, single-choice poll handling and the events "today" boundary all differ between them, and `serviceContract.test.ts` only checks method signatures, not behavior. Worse, because `_mockApi` is referenced directly, the full mock plus its seed (PII-like demo data) ships in the production bundle: with an empty `API_BASE_URL` a production build silently boots the mock with a password-less admin login.

## What Changes

- Introduce **MSW (Mock Service Worker)** as the demo/test backend, intercepting the generated client's `fetch` calls on the network layer.
- Reduce `serviceLayer.ts` to a thin re-export of `realApi`; delete `_mockApi`, `seed()`, `resetDemoData`, `todayKey` and the hardcoded provider list.
- Move the in-memory domain + seed into `src/mocks/` as MSW handlers that respond with OpenAPI wire shapes (`types.gen.ts`), fixing the known drift bugs in the process.
- Add a **production fail-safe**: a prod build without `API_BASE_URL` (and without `VITE_ALLOW_MOCK`) throws instead of booting the mock.

## Capabilities

### New Capabilities
- `demo-mode`: how the app serves a backend-less demo/test experience without shipping a second business-logic implementation or exposing it in production.

### Modified Capabilities
<!-- none: specs/ is being populated fresh under OpenSpec adoption -->

## Impact

- Frontend: `src/services/serviceLayer.ts` (gutted), `src/services/serviceLayerReal.ts`, `src/api/*`, `src/main.tsx` (MSW bootstrap + fail-safe), `src/test/setup.ts`, `vitest.config.ts`, `package.json` (devDependency `msw`).
- Tests: `serviceLayer.test.ts` / `serviceContract.test.ts` replaced by MSW handler + behavioral contract tests.
- CI gates: bundle budget (mock/seed must tree-shake out of prod), coverage floors, Playwright E2E.
