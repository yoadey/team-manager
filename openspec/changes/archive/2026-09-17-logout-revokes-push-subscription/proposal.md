## Why

`AppContext.tsx`'s `logout()` (~line 600) calls `api.auth.logout()` but
never touches the browser's Web Push registration. Push delivery on the
backend is keyed only on team membership + module permission
(`backend/internal/push/repository.go`'s `ListForTeamExcludingUser`/
`ListForTeam`), not on whether the browser currently holds a live session.

Concrete failure scenario: on a shared/kiosk device, User A enables push,
then logs out. The `push_subscriptions` row still has `user_id = A`, and
every subsequent event/news/poll/attendance notification for A's team keeps
popping up as an OS-level notification on that device (`frontend/public/
sw.js`'s `push` handler shows notifications regardless of app session
state) until A manually disables push in Settings, or someone else opts in
on the same device — which silently reassigns the endpoint server-side via
`ON CONFLICT (endpoint) DO UPDATE SET user_id = EXCLUDED.user_id`. Either
way, a former session keeps receiving a departed user's notifications, or a
new user's opt-in silently transfers an old subscription's push history.

The unsubscribe logic already exists (`usePushActions.ts`'s `disablePush`)
but is only reachable from the explicit Settings toggle, not from logout.

## What Changes

- Extract the browser-unsubscribe + backend-delete logic already in
  `usePushActions.ts`'s `disablePush` into a standalone async function
  (`unsubscribeWebPush`), usable outside a React hook.
- `disablePush` calls the extracted function (behavior-preserving refactor;
  no change to its own success/error handling).
- `AppContext.tsx`'s `logout()` calls the same function so a Web Push
  subscription is revoked (both the local `PushSubscription.unsubscribe()`
  and the backend's `DELETE /users/me/push-subscriptions`) as part of every
  logout.
- The call is non-blocking and never fails logout: browsers without push
  support, no active subscription, and network/backend failures are all
  tolerated, matching how `logout()` already treats its own
  `api.auth.logout()` failure as non-critical (fire-and-forget +
  `captureException`, not a blocking await).

## Capabilities

### Modified Capabilities
- `push-notifications`: a subscription is also revoked implicitly on
  logout, not only via the explicit Settings toggle.

## Impact

- `frontend/src/features/notifications/hooks/usePushActions.ts` (extract
  `unsubscribeWebPush`, `disablePush` calls it).
- `frontend/src/context/AppContext.tsx` (`logout` calls
  `unsubscribeWebPush`).
- `frontend/src/context/AppContext.test.tsx` (new/updated logout test).
- No API shape change; no migration.
