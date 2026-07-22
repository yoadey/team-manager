## 1. Database
- [ ] 1.1 New migration `00030_push_subscriptions.sql`: `push_subscriptions`
      (`id UUID PK`, `user_id UUID FK -> users ON DELETE CASCADE`,
      `endpoint TEXT NOT NULL UNIQUE`, `p256dh TEXT NOT NULL`,
      `auth_key TEXT NOT NULL`, `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`,
      `last_used_at TIMESTAMPTZ`)
- [ ] 1.2 Index on `push_subscriptions(user_id)`
- [ ] 1.3 `make migrate` locally; confirm `migration-rollback` (up→down→up)
      and `migration-safety` gates pass

## 2. Push package
- [ ] 2.1 `internal/push/push.go`: `Pusher` interface
      (`Send(ctx, subscription Subscription, payload Payload) error`),
      `Subscription{Endpoint, P256dh, AuthKey string}`,
      `Payload{Title, Body string, URL string}`, `ErrGone` sentinel for
      404/410 so callers can distinguish "delete this subscription" from
      other failures
- [ ] 2.2 `internal/push/webpush.go`: `WebPusher` using
      `github.com/SherClockHolmes/webpush-go` + VAPID keys, maps
      404/410 responses to `ErrGone`
- [ ] 2.3 `internal/push/fake.go`: `FakePusher` (logs, exposes a test
      accessor for the last payload sent, mirroring `mailer.FakeMailer`)
- [ ] 2.4 `go get github.com/SherClockHolmes/webpush-go`; pin identically in
      `go.mod`/`Makefile`/`ci.yml`/`Dockerfile` per `openspec/config.yaml`'s
      tool-pin-sync rule (only if this dep needs pinning beyond go.mod —
      confirm during implementation whether `tool-pin-sync` applies to
      library deps or only pinned CLI tools)

## 3. Config
- [ ] 3.1 `VAPID_PUBLIC_KEY`, `VAPID_PRIVATE_KEY`, `VAPID_SUBJECT`;
      `loadVAPIDConfig(cookieSecure bool)` required-when-`COOKIE_SECURE=true`
      (`ErrVAPIDConfigRequired`), same shape as `loadSMTPConfig`/`s3Settings`
      in `internal/config/config.go`

## 4. OpenAPI
- [ ] 4.1 `POST /users/me/push-subscriptions` (register: `endpoint`,
      `keys.p256dh`, `keys.auth` — the shape `PushSubscription.toJSON()`
      already produces in the browser), `DELETE /users/me/push-subscriptions`
      (body or query: `endpoint`) — both `x-rbac-module: public`,
      `x-rbac-self-service: true`; new schemas `PushSubscriptionRequest`
- [ ] 4.2 `cd backend && make generate` (commit `internal/gen/api.gen.go`)
- [ ] 4.3 repo-root `make generate-ts` (commit `frontend/src/api/types.gen.ts`)

## 5. Backend wiring
- [ ] 5.1 `internal/push/repository.go` (or fold into an existing
      user-scoped repo): `Upsert(ctx, userID, sub) error` (`ON CONFLICT
      (endpoint) DO UPDATE` so re-subscribing the same browser doesn't
      duplicate), `Delete(ctx, userID, endpoint) error`,
      `ListByUser(ctx, userID) ([]Subscription, error)`
- [ ] 5.2 New handler (`internal/push/handler.go` or alongside `auth`) for
      the two self-service routes, resolving the caller via
      `auth.UserFromContext`
- [ ] 5.3 `internal/jobs/notification_worker.go`: after the `notifications`
      insert commits, load the recipient's subscriptions, apply
      `notificationModule`/`hasReadAccess` (exported from
      `internal/notifications` or duplicated if that would create an import
      cycle — resolve during implementation), enqueue `PushDeliveryArgs{
      SubscriptionID, Title, Body, URL}` per subscription that passes the
      gate
- [ ] 5.4 New `PushDeliveryWorker` (same file or `internal/jobs/push_worker.go`):
      calls `Pusher.Send`; on `push.ErrGone` deletes the subscription row;
      registered via `river.AddWorker` in `jobs.NewClient` alongside
      `NotificationWorker`
- [ ] 5.5 `cmd/server/main.go`: construct `Pusher` (real vs `FakePusher`
      depending on whether VAPID env vars are set), pass into
      `jobs.NewClient`/the new handler, register the two routes

## 6. Metrics
- [ ] 6.1 `internal/metrics/business.go`: push send success/failure counters
      (mirroring `NotificationJobFailures`/`NotificationEnqueueFailures`)

## 7. Frontend
- [ ] 7.1 `public/sw.js`: `push` listener (`self.registration.showNotification`
      using the payload's title/body), `notificationclick` listener
      (focuses/opens a client at the payload's URL)
- [ ] 7.2 New `features/notifications/hooks/usePushActions.ts`: `subscribe()`
      (`Notification.requestPermission` → `pushManager.subscribe({
      applicationServerKey: VITE_VAPID_PUBLIC_KEY, userVisibleOnly: true })`
      → POST `/users/me/push-subscriptions`), `unsubscribe()` (
      `pushManager.getSubscription()` → `unsubscribe()` → DELETE)
- [ ] 7.3 `features/team/components/NavSheets.tsx`: `ProfileSheet` gets a
      "Web-Push aktivieren" toggle calling the new hook; reflects current
      `Notification.permission`/subscription state
- [ ] 7.4 `services/serviceLayerReal.ts`: `push.subscribe/unsubscribe`
- [ ] 7.5 `mocks/handlers.ts` + `mocks/db.ts`: MSW handlers for both routes
- [ ] 7.6 `i18n/en.ts` + `i18n/de.ts`: toggle label/description, permission-denied copy
- [ ] 7.7 `.env`/`.env.example`: `VITE_VAPID_PUBLIC_KEY`

## 8. Tests
- [ ] 8.1 Backend: `push.WebPusher`/`FakePusher` unit tests; `NotificationWorker`
      module-gate + enqueue behavior (permission granted vs "none");
      `PushDeliveryWorker` success, transient failure (retry), `ErrGone`
      (subscription deleted); config `loadVAPIDConfig` required-when-secure
      + parsing; handler CRUD for subscriptions (own vs another user's
      subscription cannot be deleted)
- [ ] 8.2 Frontend: `usePushActions` subscribe/unsubscribe flow;
      `ProfileSheet` toggle; `serviceContract.test.ts` new scenarios

## 9. Docs
- [ ] 9.1 `CLAUDE.md` env var table: `VAPID_PUBLIC_KEY`, `VAPID_PRIVATE_KEY`, `VAPID_SUBJECT`

## 10. Verification
- [ ] 10.1 `openspec validate add-web-push-notifications --strict`
- [ ] 10.2 `cd backend && make generate` / repo-root `make generate-ts` — no diff
- [ ] 10.3 `cd backend && make lint`
- [ ] 10.4 `cd backend && make test` (unit + integration)
- [ ] 10.5 `govulncheck`
- [ ] 10.6 `migration-rollback` / `migration-safety` on the new migration
- [ ] 10.7 `backend-openapi-drift`
- [ ] 10.8 `cd frontend && npm run lint && npm run typecheck && npm test && npm run build`
- [ ] 10.9 Manual: subscribe in a real browser, trigger a notification
      (e.g. create an event as another member), confirm the push arrives and
      clicking it opens the app at the right place; revoke browser
      permission and confirm the next delivery attempt prunes the
      subscription row
