## Why

Production logs (`team-manager-test` namespace) show `push_delivery` jobs
retrying forever with responses that unambiguously identify a *specific*
subscription as permanently stale, not a server-wide auth misconfiguration:

```
status 401: {"code":401,"errno":109,"error":"Unauthorized","message":"VAPID public key mismatch", ...}
status 403: the VAPID credentials in the authorization header do not correspond to the credentials used to create the subscriptions.
```

`push-delivery-error-diagnostics` (already merged) deliberately kept every
401/403 on the existing retry-forever path, reasoning that "a VAPID auth
failure isn't scoped to one subscription (it's this server's key
material)... the subscription itself may still be perfectly valid once the
key mismatch is fixed." That's true for a malformed `VAPID_SUBJECT` or a
misconfigured key pair — those affect every subscription equally and a
config fix resolves all of them at once. But Mozilla autopush's errno 109
and FCM's "credentials ... do not correspond to the credentials used to
create the subscription[s]" messages are a different case: the push
service is telling us *this particular* subscription's stored public key no
longer matches ours. That happens once a subscription was created against
a VAPID public key this server no longer signs with (e.g. keys rotated, or
test-environment data seeded/restored across a key change) — the browser
would have to unsubscribe and re-subscribe for that endpoint to ever work
again, and no amount of retrying or backend config fix changes that for
this row. Left alone, River retries these jobs with backoff for the life
of the job (up to its max-attempts window), which is exactly the sustained
log noise seen in production.

## What Changes

- `push.WebPusher.Send` recognizes the specific known VAPID
  key-mismatch response signatures (Mozilla errno 109 / message text,
  FCM's "credentials used to create the subscription" text) on a 401/403
  response and maps them to `ErrGone` — reusing the existing prune path —
  instead of the generic `ErrPushServiceStatus` retry-forever path. Any
  other 401/403 (e.g. a malformed `VAPID_SUBJECT`) is unaffected and keeps
  retrying, since that class of failure is genuinely server-wide and
  recoverable by a config fix.
- `docs/operations.md`'s Web Push troubleshooting entry is updated to
  describe the new behavior: a key-mismatch-specific 401/403 is now
  pruned automatically (no ops action needed beyond the affected user
  re-enabling push), while other 401 causes still recur until config is
  fixed.

## Capabilities

### Modified Capabilities
- `push-notifications`: a subscription whose push service response
  identifies a VAPID key mismatch is now pruned like a 404/410, instead of
  retrying forever.

## Impact

- `backend/internal/push/webpush.go` (+tests in `webpush_test.go`).
- `docs/operations.md` (Web Push troubleshooting section only).
- No API, migration, or RBAC changes. No change to the generic
  `ErrPushServiceStatus` retry-forever path for other 401/403 causes.
