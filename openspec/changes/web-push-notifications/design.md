## Context

The app is an installable PWA (`manifest.webmanifest`, `sw.js`) but the service worker only does offline navigation fallback. The backend already computes per-user notifications (activity feed) and filters them by module permission (`notifications.Service.List`). What's missing is a delivery channel to the browser.

## Goals / Non-Goals

**Goals:**
- Real Web Push delivery of the notifications the backend already generates.
- Per-user opt-in; graceful no-op when unconfigured or permission denied.
- Reliable pruning of dead subscriptions.

**Non-Goals:**
- Email/native push (APNs/FCM direct); this is standards-based Web Push only.
- Reworking what counts as a notification.

## Decisions

- **VAPID** application-server keys via config (`VAPID_PUBLIC_KEY`/`VAPID_PRIVATE_KEY`/`VAPID_SUBJECT`); feature is disabled (endpoints 404/short-circuit) when unset. Public key exposed to the client (config-driven, like other public runtime config).
- Store subscriptions in a `push_subscriptions` table keyed by user, unique on endpoint; store p256dh/auth keys.
- Send asynchronously via a **River** job (matches existing `internal/jobs`), so a slow/failing push service never blocks the request creating the notification. Respect the notification's module-permission filtering already in `notifications.Service`.
- Payloads: RFC 8291 encrypted, minimal (title, body, deep-link URL, tag). On `404`/`410`, delete the subscription.
- Use a maintained Go webpush library (pin per repo convention across go.mod/Makefile/ci/Dockerfile); justify the new runtime dep.

## Risks / Trade-offs

- iOS Safari requires the PWA be installed to the Home Screen for push; document this limitation rather than promising universal delivery.
- New runtime dependency (webpush + crypto) — deliberately dependency-light repo; justify in proposal, keep it backend-only.
- Payload privacy: notification bodies may contain member-related text — keep payloads terse and rely on the app for detail; do not leak PII beyond what the in-app feed already shows the same user.
- Secret handling: VAPID private key is a secret (env/Helm secret), never committed.
