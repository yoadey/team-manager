## Why

All server state lives in `src/context/AppContext.tsx` (~1538 lines), which hand-reimplements what TanStack Query provides: manual loaders with monotonic sequence refs (`refreshEventsSeq`, `refreshTeamsSeq`, …) plus `activeTeamId` guards against out-of-order responses, a `Promise.allSettled` initial load, manual cache invalidation after mutations, and a single global `busy` flag with a `clearBusyIfOwned` race guard. This is exactly the orchestration React Query does out of the box, and it is a major driver of the God-context problem.

## What Changes

- Add `@tanstack/react-query`; wrap the app in a `QueryClientProvider`.
- Replace hand-written loaders with `useQuery` hooks (team-scoped query keys) and mutations with `useMutation` + `invalidateQueries`.
- Remove server-state, loaders, sequence refs, the `allSettled` block, the global `busy` flag and `clearBusyIfOwned` from `AppContext`.
- Keep only genuine client/UI state in the context (auth session, active team, route/URL sync, toast, sheet state).

## Capabilities

### New Capabilities
- `client-data-fetching`: how the frontend fetches, caches, invalidates and de-races server data, and reports per-operation pending state.

### Modified Capabilities
<!-- none -->

## Impact

- Frontend: `src/context/AppContext.tsx` (server state removed), `src/context/useFeatureActions.ts`, per-feature query/mutation hooks, feature components (from `useApp().<state>` to `use<Resource>()`), `src/utils/forms.ts` (busy guard removed), `src/main.tsx`.
- Overlaps `AppContext.tsx` with the react-hook-form change — sequence merges.
