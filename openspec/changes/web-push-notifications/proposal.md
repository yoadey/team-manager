## Why

Users report web-push notifications never arrive. Root cause: there is no web-push transport. `frontend/public/sw.js` is a navigation-only service worker with **no `push`/`notificationclick` handler**; there is no `PushSubscription`, no VAPID keys, and no backend endpoint to store subscriptions or send pushes. Notifications today exist only as the in-app activity feed (`internal/notifications`). So "push doesn't come through" is really "push was never wired up".

## What Changes

- Frontend: request notification permission, subscribe via `PushManager` using a VAPID public key, and add `push` + `notificationclick` handlers to `sw.js` that show a notification and deep-link into the app.
- Backend: store push subscriptions per user (endpoint + keys), and send Web Push (VAPID / RFC 8291 encrypted payloads) when a notification is created — reusing the existing notification-generation points.
- Prune subscriptions on `404`/`410` from the push service. Guard everything behind config (feature disabled when VAPID keys unset).

## Capabilities

### New Capabilities
- `push-notifications`: deliver activity notifications to subscribed browsers via Web Push.

## Impact

- Spec/backend: `openapi.yaml` (subscribe/unsubscribe endpoints, VAPID public-key endpoint or config-exposed value); new `push_subscriptions` table + migration; a push-sender in `internal/notifications` (or a new `internal/push`) invoked where notifications are created (`notifications.Service`, `news.Service`, event/poll notification paths); River job for async/retry send; `internal/config` (VAPID keys); regenerate clients; tests.
- Frontend: `frontend/public/sw.js` (`push`, `notificationclick`), a subscribe hook + settings toggle, `frontend/src/api/*`, `frontend/src/i18n/{de,en}.ts`.
- Ops: `docs/operations.md` + env docs for `VAPID_PUBLIC_KEY`/`VAPID_PRIVATE_KEY`/`VAPID_SUBJECT`; Helm values.
- CI: openapi-drift, migration gates, backend + frontend gates. **API + migration-affecting.**
