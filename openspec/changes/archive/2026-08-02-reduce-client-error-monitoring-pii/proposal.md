## Why

`frontend/src/monitoring.ts`'s `beforeSend` hook strips `event.user.email` and
`event.user.ip_address` before an event reaches Sentry, with an explicit
comment that email and IP "must never leave the browser". But
`setSentryUser` (called on login) sets `username: user.name` — the member's
real display name — which is never stripped by `beforeSend` and is
attached to every subsequent event via Sentry's scope. This is an
unintentional PII leak to a third-party error tracker that contradicts the
file's own stated intent, and sits oddly next to the app's otherwise
GDPR-conscious posture (Art. 15/17 export/delete flows documented
elsewhere).

## What Changes

- `setSentryUser` passes only the user's opaque `id` to `Sentry.setUser`,
  not `name`.
- `beforeSend` additionally strips `event.user.username` as defense in
  depth, in case a future call site sets it directly.

## Capabilities

### Added Capabilities
- `client-error-monitoring`: error events sent to Sentry carry no
  personally identifying user data (email, IP, display name) — only an
  opaque user id.

## Impact

- `frontend/src/monitoring.ts` (+ its test file).
- No backend, API, or migration changes.
