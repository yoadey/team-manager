## Context

`internal/notifications/service.go`'s `Service.List` already re-derives, on
every read, which module a notification type belongs to
(`notificationModule`) and whether the caller currently has at least "read"
on that module (`hasReadAccess`), failing closed on any unrecognized module.
This exists specifically so a member whose permission changes after a
notification was created doesn't see stale-but-now-forbidden items. Push
delivery must not create a side channel around that gate: a member with
`news: none` must not receive a push for a news post just because the push
job ran before their permission changed, or because push is wired
independently of the read-time filter.

Delivery already runs through `internal/jobs`, a River (Postgres-backed)
queue — `NotificationWorker.Work` inserts the `notifications` row with
at-least-once semantics (`ON CONFLICT (river_job_id) ... DO NOTHING`).
`internal/mailer` is the closest existing analog for an outbound,
best-effort, network-calling side effect: a tiny interface, a real
implementation, and a fake used in dev/tests, wired in `cmd/server/main.go`
exactly like `internal/storage`'s `ObjectStore`.

## Goals / Non-Goals

**Goals:**
- Deliver a push notification for every notification type already shown
  in-app, gated by the recipient's *current* module read permission at
  send time, not at enqueue time.
- No new operational service — VAPID + `webpush-go` talk directly to the
  browser vendor's push endpoint over plain HTTPS.
- Dead subscriptions (browser uninstalled, permission revoked, endpoint
  expired) are pruned automatically instead of accumulating and eating send
  quota forever.
- Opt-in, self-service, reversible from the client at any time.

**Non-Goals:**
- No push notification preferences *per module/type* in v1 — a subscription
  is all-or-nothing, matching the in-app feed's own lack of per-type
  filtering. A follow-up change can add granularity if requested.
- No admin visibility into who has push enabled.
- No native app / FCM SDK — Web Push only, since the frontend is a web PWA,
  not a native app.

## Decisions

**Where the permission gate runs.** The gate re-runs inside
`NotificationWorker.Work`, immediately before enqueuing `PushDeliveryArgs`
jobs — i.e. at the same moment the notification row itself is durably
created, using the same `notificationModule`/`hasReadAccess` helpers
`notifications.Service.List` already uses (exported from
`internal/notifications` for this purpose, or duplicated verbatim if an
import cycle would otherwise result — `internal/jobs` currently has no
dependency on `internal/notifications`, so the cleaner path is decided
during implementation by checking for a cycle). This is a "gate at
notify-time" property, not "gate at read-time" like `List` — a permission
change between notify and actual push delivery (seconds to minutes later,
bounded by River's worker latency) is an accepted, negligible window,
consistent with push notifications generally being best-effort/at-most-once
in spirit even though the queue itself is at-least-once.

**Subscription scope: per user, not per (user, team).** A `push_subscriptions`
row keys on `user_id` alone. A member typically belongs to a handful of
teams and expects one "enable push" toggle to cover all of them — scoping
per team would mean re-subscribing per team switch, which doesn't match how
the rest of the account-level settings (color scheme, locale) already work
in `ProfileSheet`.

**Stale-subscription cleanup.** `webpush-go`'s `Send` returns the push
service's HTTP status. A `404`/`410` unambiguously means "this endpoint will
never accept another push" per the Web Push protocol (RFC 8030) — the
`PushDeliveryWorker` deletes the `push_subscriptions` row on those codes and
otherwise just logs+retries (River's built-in retry/backoff) on transient
failures (5xx, network errors).

**VAPID key handling.** Same required-when-`COOKIE_SECURE=true` pattern as
`S3_*`/`SMTP_*`: `loadVAPIDConfig(cookieSecure bool)` returns
`ErrVAPIDConfigRequired` if `VAPID_PUBLIC_KEY`/`VAPID_PRIVATE_KEY`/
`VAPID_SUBJECT` are incomplete while `cookieSecure` is true; unset in dev
falls back to a `FakePusher` that logs instead of sending, mirroring
`FakeMailer`. `VAPID_PUBLIC_KEY` is not a secret (it's shipped to every
browser as `applicationServerKey`) — it's additionally exposed as
`VITE_VAPID_PUBLIC_KEY` so the frontend doesn't need a dedicated "fetch the
public key" endpoint.

**Payload contents.** The push payload is a small JSON blob (title, body,
an optional deep-link route) built from the same fields `toGenNotification`
already assembles — no separate templating system. Payload size stays well
under the 4 KB Web Push practical limit.

## Risks / Trade-offs

- **Best-effort delivery.** Web Push has no delivery guarantee once it
  leaves this backend (the browser vendor's infrastructure can drop it,
  and a closed/backgrounded browser may coalesce or delay it). This is
  inherent to the standard, not a gap in this design — the in-app feed
  remains the source of truth.
- **Payload gate window.** As noted above, the module-permission check runs
  at enqueue time, not at actual delivery time (which can lag by however
  long the worker queue takes) — an edge case, not a broad exposure, since
  the same lag already exists for the `notifications` row itself becoming
  visible in-app.
- **New runtime dependency** `github.com/SherClockHolmes/webpush-go` —
  justified per `openspec/config.yaml`'s "justify new runtime deps" rule:
  implementing VAPID/aes128gcm payload encryption by hand is exactly the
  kind of security-sensitive crypto code this project should not
  hand-roll.
