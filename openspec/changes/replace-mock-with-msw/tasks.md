## 1. Dependency & scaffolding
- [x] 1.1 Add `msw@^2` as a `devDependency`; run `npx msw init public/ --save`
- [x] 1.2 Create `src/mocks/` with `db.ts`, `handlers.ts`, `browser.ts`, `server.ts`, `seedControls.ts`

## 2. Mock domain in MSW handlers
- [x] 2.1 Port the seed data from `serviceLayer.ts:161-708` into `mocks/db.ts`, decoupled from internal mock types
- [x] 2.2 Write one `http.<method>` handler per OpenAPI operation, typing responses from `@/api/types.gen.ts`
- [x] 2.3 While porting, fix drift bugs: penalty `label`/`amount` snapshot on assignment; stats use effective status; single-choice poll with >1 option returns 422; `scope=upcoming` includes today's events
- [x] 2.4 Add `GET /auth/providers` handler returning a `password` provider; implement a clearly-marked demo `POST /auth/login` that sets a cookie/token

## 3. Switch & fail-safe
- [x] 3.1 Reduce `serviceLayer.ts` to re-export `realApi` as `api`; delete `_mockApi`, `seed`, `todayKey`, provider list. `resetDemoData` is kept as an intentional no-op (see `src/services/index.ts`) — `AppContext`'s "reset demo data" action still calls it and always follows with `location.reload()`, which alone re-seeds `src/mocks/db.ts`'s module-level in-memory DB from scratch.
- [x] 3.2 In `main.tsx`, start MSW before render only when `!config.apiBaseUrl`; throw in prod builds without `VITE_ALLOW_MOCK`
- [x] 3.3 Verify MSW matches the relative `/api/v1` baseUrl from `api/client.ts`

## 4. Tests
- [x] 4.1 Start MSW `server` in `src/test/setup.ts` (`listen`/`resetHandlers`/`close`, unhandled → error)
- [x] 4.2 Replace `serviceLayer.test.ts` with `mocks/handlers.test.ts` (handler behavior)
- [x] 4.3 Replace `serviceContract.test.ts` with behavioral scenarios (penalty snapshot, opt_out stats quote, single-choice reject, today upcoming)
- [x] 4.4 Update component/hook tests that relied on localStorage/`_mockApi`
- [x] 4.5 Wire E2E (`frontend-e2e`) to run against `realApi` + MSW under `VITE_ALLOW_MOCK`

## 5. Verification
- [x] 5.1 `npm run typecheck`, `npm run lint`, `npm run test` green
- [x] 5.2 `npm run build` + `npm run check:bundle` under budget; grep `dist/` confirms no seed names / no `msw`
- [x] 5.3 Dev smoke without `API_BASE_URL`: DevTools shows intercepted `/api/v1/...`; prod build without URL throws
