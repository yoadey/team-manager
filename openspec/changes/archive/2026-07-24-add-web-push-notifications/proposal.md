## Why

Teamverwaltung's activity feed (`backend/internal/notifications/`) only surfaces
new events/news/polls/absence-relevant changes when a member actually opens the
app. There is no way to reach a member who isn't currently looking at the app —
no push notification of any kind exists today. A repo-wide search confirms
zero push-related code: no `push_subscriptions`-style table, no VAPID
handling, no `web-push`/`webpush-go` dependency anywhere in `go.mod` or
`package.json`. The frontend is already a lightweight installable PWA
(`frontend/public/manifest.webmanifest`, a minimal `frontend/public/sw.js`
registered from `frontend/src/main.tsx`), but that service worker only
precaches an offline shell — it has no `push`/`notificationclick` listeners.

Web Push is a standard the browser and its OS-level push service (Mozilla,
Google FCM, Microsoft) implement end-to-end; the only thing a backend needs to
participate is a VAPID key pair and an HTTP POST per subscriber per
notification. That fits this project's "no new service" constraint exactly —
delivery happens directly from the existing Go backend to the browser's push
endpoint, with no message broker or push-relay to operate.

## What Changes

- New `push_subscriptions` table: one row per browser/device a user has
  opted in from (not per team — a user with several teams gets push for all
  of them through the same subscription).
- New `internal/push` package mirroring `internal/mailer`'s interface + real
  + fake pattern: a `Pusher` interface, a real implementation using
  `github.com/SherClockHolmes/webpush-go` + VAPID keys, and a logging fake
  for dev/tests when VAPID keys aren't configured.
- `internal/jobs/notification_worker.go`'s `NotificationWorker.Work` gains a
  step: after the `notifications` row commits, look up the recipient's
  active `push_subscriptions`, apply the same module-read-permission gate
  `notifications.Service.List` already applies (`notificationModule`/
  `hasReadAccess` in `internal/notifications/service.go`) so a member who
  can't see a module's notifications doesn't get pushed one either, and
  enqueue a `PushDeliveryArgs` job per subscription. A `410 Gone`/`404`
  response from the push service deletes the stale subscription row
  (standard Web Push hygiene).
- New self-service endpoints `POST /users/me/push-subscriptions` (register)
  and `DELETE /users/me/push-subscriptions` (unregister), both
  `x-rbac-module: public` + `x-rbac-self-service: true` — a user manages only
  their own subscriptions.
- New required-when-`COOKIE_SECURE=true` env vars `VAPID_PUBLIC_KEY`,
  `VAPID_PRIVATE_KEY`, `VAPID_SUBJECT`, following the same
  `ErrS3ConfigRequired`/`ErrSMTPConfigRequired` pattern already in
  `internal/config/config.go`. `VAPID_PUBLIC_KEY` is not secret and is also
  exposed to the frontend build as `VITE_VAPID_PUBLIC_KEY`.
- `frontend/public/sw.js` gains `push` and `notificationclick` listeners.
- New opt-in toggle in `ProfileSheet` (`frontend/src/features/team/components/NavSheets.tsx`)
  driving the subscribe/unsubscribe flow (`Notification.requestPermission` →
  `pushManager.subscribe` → POST to the backend; reverse on opt-out/logout).

## Capabilities

### New Capabilities
- `push-notifications`: opt-in Web Push delivery of the same notification
  types already shown in the in-app activity feed, gated by the recipient's
  current module permissions, with automatic cleanup of dead subscriptions.

### Modified Capabilities
<!-- none -->

## Impact

- Backend: new `internal/push/{push.go,webpush.go,fake.go}`,
  `internal/jobs/notification_worker.go` (delivery step),
  `internal/notifications/service.go` (module-gate helpers reused, not
  changed), `internal/config/config.go` (VAPID env vars),
  `cmd/server/main.go` (Pusher wiring, new routes), `internal/metrics/business.go`
  (push send/failure counters).
- Database: new migration `internal/db/migrations/00030_push_subscriptions.sql`.
- API contract: `backend/openapi/openapi.yaml` (two new operations, two new
  schemas), regenerated `internal/gen/api.gen.go` and
  `frontend/src/api/types.gen.ts`.
- Frontend: `public/sw.js`, new `features/notifications/hooks/usePushActions.ts`,
  `features/team/components/NavSheets.tsx` (`ProfileSheet` toggle),
  `services/serviceLayerReal.ts`, `mocks/{handlers.ts,db.ts}`,
  `services/serviceContract.test.ts`, `i18n/{en.ts,de.ts}`, `.env`
  (`VITE_VAPID_PUBLIC_KEY`).
- Docs: `CLAUDE.md` env var table (`VAPID_PUBLIC_KEY`, `VAPID_PRIVATE_KEY`,
  `VAPID_SUBJECT`).
