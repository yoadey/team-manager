## Context

`usePushActions.ts`'s `disablePush` already implements the full
unsubscribe flow (local `PushSubscription.unsubscribe()` +
`DELETE /users/me/push-subscriptions`), but it is a `useCallback` closing
over hook-local `busy`/`support`/`toastMsg` state, only reachable from the
Settings NotificationsPanel toggle. `AppContext.tsx`'s `logout()` is a
plain function, not itself a component driving that hook's state, so it
cannot call `disablePush` directly without dragging in unrelated UI state
(toast messages, a `busy` flag with no consumer).

## Goals

- Reuse the existing unsubscribe logic verbatim rather than
  reimplementing it a second time in `AppContext.tsx`.
- Keep `disablePush`'s existing behavior (including its error toast on
  failure) unchanged for the Settings UI.
- Logout must succeed regardless of push-unsubscribe outcome: unsupported
  browser, no active subscription, or a failed backend call must never
  block or fail `logout()`.

## Decisions

- **Extract a standalone `unsubscribeWebPush(api)` function** from the body
  of `disablePush`, exported from `usePushActions.ts` alongside the hook.
  It performs the support check, `PushSubscription.unsubscribe()`, and the
  backend delete call, letting rejections propagate — it does not itself
  toast or swallow errors, since its two callers need different error
  handling.
  - `disablePush` wraps the call in its existing try/catch/toast, so its
    contract to the Settings UI is unchanged.
  - `logout()` calls it fire-and-forget with `.catch(captureException)`,
    matching the existing convention already used in the same function for
    `api.auth.logout()`'s own failure and the post-logout `providers()`
    refresh — logout's established pattern for non-critical cleanup is
    "don't await, just report."
- **Not using the `usePushActions` hook itself in `AppContext`.** Running a
  second instance of the hook's `subscribed`/`busy` state inside the
  top-level provider would create state nobody reads and could race with
  the Settings panel's own instance (e.g. both instances independently
  polling `getSubscription()` on mount). A plain async function has no such
  state to duplicate.
- **No new API endpoint.** `DELETE /users/me/push-subscriptions` already
  exists and is idempotent-in-effect (unsubscribing an endpoint that's
  already gone is a no-op both locally and server-side).

## Risks

- **Extra network round-trip on every logout**, even when push was never
  enabled. Mitigated: `unsubscribeWebPush` short-circuits before touching
  `navigator.serviceWorker` when `config.vapidPublicKey` is unset or the
  browser lacks `serviceWorker`/`PushManager` support, and again when
  `getSubscription()` resolves to no active subscription — the network
  call only happens when there's an actual subscription to revoke.
- **Logout takes marginally longer to visually settle** if `logout()` is
  changed to await the call. Avoided: the call stays fire-and-forget
  (matching the existing `api.auth.logout()` pattern), so it never gates
  the UI transition to the login screen.
