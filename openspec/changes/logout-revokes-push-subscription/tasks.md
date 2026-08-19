## 1. Frontend

- [x] 1.1 `usePushActions.ts`: extract `disablePush`'s unsubscribe logic
      (local `PushSubscription.unsubscribe()` + `api.push.unsubscribe`)
      into a standalone exported `unsubscribeWebPush(api)` async function
- [x] 1.2 `disablePush` calls `unsubscribeWebPush`, keeping its existing
      try/catch/toast behavior unchanged
- [x] 1.3 `AppContext.tsx`'s `logout()` calls `unsubscribeWebPush(api)`
      fire-and-forget with `.catch(captureException)`, matching the
      existing non-blocking pattern already used for `api.auth.logout()`'s
      own failure in the same function
- [x] 1.4 `AppContext.test.tsx`: add/update a logout test asserting
      `unsubscribeWebPush` is called on logout, and that logout still
      completes (reaches the `login` phase) when it rejects

## 2. Verification

- [x] 2.1 `cd frontend && npm test -- --run` (scoped to
      `usePushActions.test.ts` and `AppContext.test.tsx`)
- [x] 2.2 `cd frontend && npm run typecheck`
