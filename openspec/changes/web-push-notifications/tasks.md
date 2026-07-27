## 1. Spec + config
- [ ] 1.1 Add `POST /push/subscribe` + `DELETE /push/subscribe` (and a way to expose the VAPID public key) to `openapi.yaml`
- [ ] 1.2 Add `VAPID_PUBLIC_KEY`/`VAPID_PRIVATE_KEY`/`VAPID_SUBJECT` to `internal/config`; feature off when unset
- [ ] 1.3 Run `make generate` + `make generate-ts`; commit generated output

## 2. Backend
- [ ] 2.1 Migration: `push_subscriptions` (user_id, endpoint unique, p256dh, auth, created_at)
- [ ] 2.2 Subscription repository + handlers (subscribe/unsubscribe), team/user-scoped and auth-guarded
- [ ] 2.3 Push sender (new webpush dep, pinned across go.mod/Makefile/ci/Dockerfile) sending RFC 8291 payloads; prune on 404/410
- [ ] 2.4 Enqueue a River job when a notification is created; job sends to the user's subscriptions, honoring existing module-permission filtering

## 3. Frontend
- [ ] 3.1 `sw.js`: add `push` (show notification) + `notificationclick` (focus/deep-link) handlers
- [ ] 3.2 Subscribe hook: request permission, `PushManager.subscribe` with the VAPID public key, POST to backend; settings toggle to enable/disable
- [ ] 3.3 `de`/`en` strings; handle permission-denied/unsupported gracefully

## 4. Ops/docs
- [ ] 4.1 Document VAPID env vars (CLAUDE.md env table + `docs/operations.md`), Helm values/secret, and the iOS install-to-home-screen caveat

## 5. Verification
- [ ] 5.1 openapi-drift green; new endpoints in the RBAC table as appropriate
- [ ] 5.2 Backend tests: subscribe/unsubscribe, send happy-path (mocked push service), 410 → subscription pruned, feature-off short-circuit
- [ ] 5.3 `golangci-lint` + `go test ./...` + govulncheck + migration gates green
- [ ] 5.4 Frontend lint/typecheck/test/build green; sw handlers covered
