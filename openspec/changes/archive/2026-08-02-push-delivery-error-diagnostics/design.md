## Context

`push.WebPusher.Send` wraps a non-2xx response into
`fmt.Errorf("push.WebPusher.Send: %w: status %d", ErrPushServiceStatus,
resp.StatusCode)` and closes `resp.Body` without reading it. River's
default error handler logs the returned error verbatim, so the operator
only ever sees `status 401` with no indication of *why* — was it a bad
VAPID subject, a key mismatch, an expired JWT, or something else the push
service can already tell us.

## Goals

- Surface the push service's own diagnostic message in the existing error
  path, with no new logging call and no change to the `ErrGone`/retry
  control flow.
- Bound how much body we read/embed so a misbehaving push service can't
  bloat log lines or hold the connection open.
- Give operators a documented first guess (VAPID key mismatch) for a 401
  specifically, since that's the failure mode with no automatic recovery
  (unlike a transient 5xx, which just retries).

## Decisions

- **Read via `io.LimitReader` (2 KB cap) before checking status.** Push
  service error bodies are small JSON objects; 2 KB is generous headroom
  without risking large log lines. The body is read once regardless of
  status so the existing 404/410 → `ErrGone` branch is untouched (still no
  body reference needed there — `ErrGone` stays a plain sentinel, since
  callers only compare it with `errors.Is`).
- **Truncate + single-line the body before embedding**, not error-wrap it
  as a distinct type. A structured field would require a new field on
  the returned error and updating every caller/log call site to unpack
  it; a formatted string keeps the fix to `webpush.go` and its test.
- **No change to retry behavior for 401/403.** A VAPID auth failure isn't
  scoped to one subscription (it's this server's key material), so
  treating it like `ErrGone` and deleting the subscription would be wrong
  — the subscription itself may still be perfectly valid once the key
  mismatch is fixed. It keeps retrying via River's backoff, same as any
  other non-Gone failure; the fix here is purely diagnostic.
- **Docs, not code, for the "how do I fix a 401" story.** There's no way
  for the backend to know *which* of subject/format/key-mismatch caused a
  given 401 beyond what the push service's own body says, so the
  actionable guidance (check `VITE_VAPID_PUBLIC_KEY`/`VAPID_PUBLIC_KEY`
  parity, remember VAPID keys have no rotation mechanism unlike
  `COOKIE_ENCRYPTION_KEYS`) belongs in `docs/operations.md`, not as a new
  code branch.

## Risks

- **Response bodies could contain sensitive data.** Push service error
  bodies are vendor-authored JSON describing the auth failure, not
  user content; low risk, and capped at 2 KB regardless.
- **Reading the body changes timing negligibly** (one bounded read before
  `resp.Body.Close()`, already deferred) — no functional risk to the
  10s `webpushTimeout`.
