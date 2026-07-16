## 1. Setup
- [ ] 1.1 Add `@tanstack/react-query@^5` (and devtools as a dev-only dynamic import)
- [ ] 1.2 Create `src/query/client.ts` with a `QueryClient`; retry policy excludes `AuthError`/`ForbiddenError`/`ValidationError`
- [ ] 1.3 Wrap the app in `<QueryClientProvider>` in `main.tsx`, inside existing ErrorBoundaries
- [ ] 1.4 Create `src/query/keys.ts` — team-scoped key factory (`['teams', teamId, <resource>]`)

## 2. Queries & mutations
- [ ] 2.1 Add `use<Resource>` query hooks per feature (events, members, finances, polls, news, absences, notifications, stats)
- [ ] 2.2 Add `use<Action>` mutation hooks with `onSuccess` → `invalidateQueries` on the team-scoped key
- [ ] 2.3 Replace the global `busy` flag usage with `mutation.isPending`

## 3. Context slimming
- [ ] 3.1 Remove list state, loaders, sequence refs and the `Promise.allSettled` block from `AppContext.tsx`
- [ ] 3.2 Keep auth session, `activeTeamId`, route/URL sync, toast, sheet state
- [ ] 3.3 Delete `clearBusyIfOwned` and the `busy` field once fully migrated
- [ ] 3.4 On deep-link/`popstate` restore, set route/team only and let queries load (or `prefetchQuery`)

## 4. Migration order
- [ ] 4.1 Migrate the `events` vertical fully and get it green first
- [ ] 4.2 Migrate remaining features one at a time

## 5. Verification
- [ ] 5.1 `npm run typecheck` + `npm run lint` green
- [ ] 5.2 `npm run test` green; tests use a per-test `QueryClient` with `retry: false`
- [ ] 5.3 `npm run build` + `check:bundle` under budget
- [ ] 5.4 Manual smoke: rapid team switch shows no stale lists; a mutation refreshes its list; a failing module leaves others intact
