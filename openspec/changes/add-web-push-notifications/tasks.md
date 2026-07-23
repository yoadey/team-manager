## 1. Database
- [x] 1.1 New migration `00002_push_subscriptions.sql` (renumbered from
      `00030` after `main` squashed all prior migrations into a single
      `00001_init.sql`): `push_subscriptions`
      (`id UUID PK`, `user_id UUID FK -> users ON DELETE CASCADE`,
      `endpoint TEXT NOT NULL UNIQUE`, `p256dh TEXT NOT NULL`,
      `auth_key TEXT NOT NULL`, `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`,
      `last_used_at TIMESTAMPTZ`)
- [x] 1.2 Index on `push_subscriptions(user_id)`
- [x] 1.3 `make migrate` locally against a real Postgres; confirmed
      up→down→up round-trips cleanly (`migration-rollback`/`migration-safety`
      CI gates not run directly — no `golangci-lint`-style local runner for
      them, but the same `goose up`/`down`/`up` sequence they use was
      exercised manually)

## 2. Push package
- [x] 2.1 `internal/push/push.go`: `Pusher` interface, `Subscription`,
      `Payload`, `ErrGone` sentinel
- [x] 2.2 `internal/push/webpush.go`: `WebPusher` using
      `github.com/SherClockHolmes/webpush-go` + VAPID keys, maps 404/410 to
      `ErrGone`
- [x] 2.3 `internal/push/fake.go`: `FakePusher`
- [x] 2.4 `go get github.com/SherClockHolmes/webpush-go` (go.mod/go.sum).
      Confirmed `tool-pin-sync` doesn't apply: that rule pins CLI tools
      installed via `go install ...@version` in `Makefile`/`ci.yml`/
      `Dockerfile` (oapi-codegen, goose, golangci-lint, govulncheck, sqlc);
      webpush-go is an ordinary imported library dependency, version-pinned
      by go.mod/go.sum alone like every other library in the module.

## 3. Config
- [x] 3.1 `VAPID_PUBLIC_KEY`, `VAPID_PRIVATE_KEY`, `VAPID_SUBJECT`;
      `loadVAPIDConfig(cookieSecure bool)` required-when-`COOKIE_SECURE=true`
      (`ErrVAPIDConfigRequired`)

## 4. OpenAPI
- [x] 4.1 `POST /users/me/push-subscriptions`, `DELETE /users/me/push-subscriptions`
      (`endpoint` as a query param, not a DELETE body) — no `x-rbac-module`
      needed (not team-scoped, same as `/auth/me/photo`); new schema
      `PushSubscriptionRequest`
- [x] 4.2 `cd backend && make generate` (committed `internal/gen/api.gen.go`,
      `internal/middleware/rbac_table.gen.go`, `internal/db/gen/models.go`)
- [x] 4.3 repo-root `make generate-ts` (committed `frontend/src/api/types.gen.ts`,
      `frontend/src/api/zod.gen.ts`)

## 5. Backend wiring
- [x] 5.1 `internal/push/repository.go`: `Upsert` (`ON CONFLICT (endpoint)
      DO UPDATE`), `Delete` (scoped to `user_id`), `DeleteByID` (for the
      delivery worker's prune-on-gone path), `ListForTeamExcludingUser`
      (joins `memberships` directly instead of a separate "list team member
      user IDs" call)
- [x] 5.2 `internal/push/handler.go`: `RegisterPushSubscription`/
      `DeletePushSubscription`, resolving the caller via `auth.UserFromContext`
- [x] 5.3 `internal/jobs/notification_worker.go`: `NotificationWorker.
      WithPushDelivery(perms, pushRepo)` gates on a genuinely new insert
      (`tag.RowsAffected() > 0`, not a retry hitting `ON CONFLICT DO NOTHING`),
      applies `notifications.NotificationModule`/`HasReadAccess` (exported
      from `internal/notifications` — no import cycle existed), enqueues
      `PushDeliveryArgs` via `river.ClientFromContextSafely` (gracefully
      skips when no River client is in context, e.g. a direct `Work()` call
      in a unit test)
- [x] 5.4 `internal/jobs/push_worker.go`: `PushDeliveryWorker`, registered via
      `river.AddWorker` in `jobs.NewClient` (new `PushDeps` param, optional —
      `nil` runs without Web Push)
- [x] 5.5 `cmd/server/main.go`: `initVAPIDPusher` (real vs `FakePusher`),
      wired into `jobs.NewClient`, `push.Handler` registered in the
      authenticated route group

## 6. Metrics
- [x] 6.1 `internal/metrics/business.go`: `PushDeliverySuccess`,
      `PushDeliveryFailures`, `PushSubscriptionsPruned`

## 7. Frontend
- [x] 7.1 `public/sw.js`: `push`, `notificationclick`, and (beyond the
      original scope) `pushsubscriptionchange` listeners — the last
      re-registers with the backend when a push service rotates a
      subscription's endpoint
- [x] 7.2 `features/notifications/hooks/usePushActions.ts`
- [x] 7.3 `features/team/components/NavSheets.tsx`: `ProfileSheet` toggle,
      only rendered when `config.vapidPublicKey` is set and the browser
      supports the Push API
- [x] 7.4 `services/serviceLayerReal.ts`: `push.subscribe/unsubscribe`
- [x] 7.5 `mocks/handlers.ts` + `mocks/db.ts`: MSW handlers for both routes
- [x] 7.6 `i18n/en.ts` + `i18n/de.ts`: new `push.*` namespace
- [x] 7.7 `.env.example`: `VITE_VAPID_PUBLIC_KEY` — additionally wired
      `VAPID_PUBLIC_KEY` through the existing runtime-config mechanism
      (`config.js.template`, `docker-entrypoint-runtime-config.sh`,
      `src/config.ts`), matching how `SENTRY_DSN` already lets a released
      image pick up a per-deployment value without a rebuild — not in the
      original task list, but required for the same reason `SENTRY_DSN`
      needed it: a rotated backend VAPID keypair must reach an
      already-built frontend image.

## 8. Tests
- [x] 8.1 Backend: `push.WebPusher`/`FakePusher`/`Service`/`Handler` unit
      tests; `NotificationWorker` push-gating tests (new insert vs.
      deduped retry, disabled-by-default, graceful skip without a River
      client in context); `PushDeliveryWorker` success/gone/transient-failure;
      `config` `loadVAPIDConfig` required-when-secure + parsing (new
      `TestLoad_VAPIDConfig*` cases). Integration tests (`push.Repository`)
      need Docker (testcontainers) and are written but skip in this sandbox
      (no Docker available) — verified equivalently via a manual smoke
      script against a real local Postgres instead (Upsert dedup, delete
      scoped to owner, `ListForTeamExcludingUser` team+actor scoping).
- [x] 8.2 Frontend: `usePushActions` subscribe/unsubscribe/permission-denied/
      no-op-disable tests; `NavSheets.test.tsx` regression-checked (toggle
      hidden when unsupported, matching existing 38 tests still green);
      `serviceLayerReal.test.ts` new `push:` describe block (not
      `serviceContract.test.ts`, which covers cross-implementation drift
      scenarios specific to the now-removed mock service layer — this repo
      already migrated to MSW-only per `replace-mock-with-msw`, so
      `serviceLayerReal.test.ts` is the correct home, matching its existing
      `notifications:`/`events:` blocks).

## 9. Docs
- [x] 9.1 `CLAUDE.md` env var table: `VAPID_PUBLIC_KEY`, `VAPID_PRIVATE_KEY`,
      `VAPID_SUBJECT`, `VITE_VAPID_PUBLIC_KEY` — additionally updated
      `docs/operations.md` (frontend runtime-config section) and the Helm
      chart (`values.yaml` env map + `existingSecret` doc,
      `templates/deployment.yaml` `VAPID_PRIVATE_KEY` secretKeyRef in both
      containers) for production deployability, since the new
      required-when-`COOKIE_SECURE=true` config has to be reachable from
      the existing Helm chart, not just documented in prose.

## 10. Verification
- [x] 10.1 `openspec validate add-web-push-notifications --strict`
- [x] 10.2 `cd backend && make generate` / repo-root `make generate-ts` — no diff
- [x] 10.3 `cd backend && make lint` (golangci-lint: 0 issues)
- [x] 10.4 `cd backend && make test` — unit tests pass; integration tests
      skip cleanly (no Docker in this sandbox) rather than failing
- [ ] 10.5 `govulncheck` — could not run in this sandbox: the outbound
      proxy returns 403 for `vuln.go.dev`; needs to run in CI, which has
      unrestricted network access
- [x] 10.6 `migration-rollback` / `migration-safety` — exercised manually
      (`goose up`/`down`/`up` against a real local Postgres 16), not via the
      CI job itself
- [x] 10.7 `backend-openapi-drift` — confirmed no diff after regenerating
- [x] 10.8 `cd frontend && npm run lint && npm run typecheck && npm test && npm run build`
      — all pass (1168 tests, 0 lint issues, bundle within budget)
- [ ] 10.9 Manual real-browser subscribe/notify/permission-revoke walkthrough
      — not performed: this sandbox has no real browser or reachable push
      service to test against. The HTTP/DB layer was verified end-to-end
      instead (curl + a real local Postgres): login, register a
      `PushSubscription`-shaped payload, confirm the row lands correctly,
      unregister, confirm it's gone. Needs a manual pass in a real browser
      before shipping.
