## Context

`AppContext` loaders (~680-876) carry per-resource sequence refs and `activeTeamId` guards so a slow response for a previously-selected team cannot overwrite fresh data. Mutations manually reload the affected list. A single shared `busy` string gates every Save button, needing the `clearBusyIfOwned` guard against self-clobber.

## Goals / Non-Goals

**Goals:**
- Move server state to `@tanstack/react-query`; delete the hand-rolled loader/sequence/cache machinery.
- Preserve team-switch correctness (the bug the sequence refs defend against must not return).
- Per-mutation pending state replacing the global `busy` flag.

**Non-Goals:**
- Changing the `api` data source or the backend.
- Adopting a data-fetching router or SSR.

## Decisions

- **Team-scoped query keys** (`['teams', teamId, 'events']`) so a team switch changes the key and old queries are discarded automatically — this replaces the `activeTeamId` guards.
- **Retry policy excludes 401/403/422** using the existing typed error classes from `serviceLayerReal.ts`, so an auth failure is not retried.
- Migrate **one feature vertical first** (events) end-to-end, then the rest.
- Devtools imported only behind `import.meta.env.DEV`.

## Risks / Trade-offs

- Team-switch correctness is the key regression risk; enforce the `teamId` key prefix everywhere.
- Bundle budget: verify React Query fits within the 250 KB/chunk, 600 KB total gzipped limits.
- Concurrent edits with the react-hook-form change in `AppContext.tsx`.
