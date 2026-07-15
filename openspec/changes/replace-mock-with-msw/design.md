## Context

Default build ships the mock: `serviceLayer.ts:2097` does `export const api = (config.apiBaseUrl ? realApi : _mockApi)`. `_mockApi` reimplements domain rules (with "Matches backend X" comments that are partly false), holds ~550 lines of seed data (16 users with German PII-like data, club "TSC Schwarz-Gelb Aachen"), and ignores the password on login (`serviceLayer.ts:963-971`).

## Goals / Non-Goals

**Goals:**
- One client implementation (`realApi`) for prod, dev-demo, and tests.
- Demo/test backend on the network layer via MSW, so demo traffic exercises the real client + generated types.
- Mock and seed physically absent from the production bundle.
- Production fail-safe against an unset `API_BASE_URL`.

**Non-Goals:**
- Changing the real backend or the OpenAPI contract.
- Building a persistent demo backend (MSW in-memory is sufficient).

## Decisions

- **MSW v2**, dev-only `devDependency`; browser worker registered behind `import.meta.env.DEV`/`VITE_ALLOW_MOCK` via dynamic `import()` so Rollup drops it from prod chunks.
- Handlers derive request/response types from `@/api/types.gen.ts` so they are type-checked against the contract; a spec change surfaces as a typecheck failure.
- Node `setupServer` in `src/test/setup.ts` with `onUnhandledRequest: 'error'`.
- While porting the mock, fix the four known drift bugs (penalty snapshot, stats effective-status, single-choice reject, today-is-upcoming) rather than carrying them over.

## Risks / Trade-offs

- Large one-time migration touching many component/hook tests; mitigate by migrating one feature vertical first.
- MSW relative-URL matching must line up with `client.ts` baseUrl (`apiBaseUrl + '/api/v1'`); verify in a dev smoke test.
- Demo login must set a cookie/token like the real backend so `credentials: 'include'` works.
