## 1. Fix

- [x] 1.1 `urlState.ts`: `buildPath` derives a detail sheet's path
      segment from `detail.kind` instead of requiring `s.route` to
      already equal `'events'`/`'members'`
- [x] 1.2 `AppContext.tsx`: refine the state→URL sync effect's
      `isNavigation` so opening a detail sheet always pushes (even
      across a route switch) and closing one always replaces

## 2. Tests

- [x] 2.1 `urlState.test.ts`: `buildPath` emits `/events/<id>` /
      `/members/<id>` for a detail sheet even when `route` is a
      different value (e.g. `'home'`)
- [x] 2.2 `AppContext.test.tsx`: opening an event detail sheet while on
      a different route (e.g. Home) updates `window.location.pathname`
      to `/events/<id>` and pushes a history entry Back can undo

## 3. Verification

- [x] 3.1 `npm run typecheck`
- [x] 3.2 `npm test`
- [x] 3.3 `npm run lint`
