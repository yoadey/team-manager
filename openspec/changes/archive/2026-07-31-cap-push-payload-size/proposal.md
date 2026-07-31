## Why

The same production logs that surfaced the VAPID key-mismatch issue
(`prune-vapid-mismatched-subscriptions`) also show `push_delivery` jobs
retrying forever with:

```
status 413: {"code":413,"errno":104,"error":"Payload Too Large","message":"This message is intended for a constrained device and is limited in size. Converted buffer is too long by 1441 bytes", ...}
```

`push.WebPusher.Send`'s push body is built by
`jobs.pushPayloadForNotification` from whichever of `EventTitle`, `Title`,
or `Note` is set on the notification — user-authored free text. Most of
those sources are bounded (`title` fields are `maxLength: 255` in
`openapi.yaml`), but `CreatePollRequest.question` has **no** `maxLength`
at all, so a long poll question becomes the push notification body
verbatim. Once encrypted (RFC 8291), that can exceed the push service's
message-size limit (commonly ~4 KB), and — like the VAPID key-mismatch
case — the payload is baked onto the job at enqueue time and never
changes, so retrying is pointless: the same oversized payload will 413
every time. Today a 413 falls into the generic retry-forever path,
producing the same kind of sustained log/retry noise.

## What Changes

- `jobs.pushPayloadForNotification` truncates the notification body to a
  bounded length (with an ellipsis marker) before it's ever enqueued, so a
  push notification's body — meant to be a short preview, not the full
  content — can't produce an oversized encrypted payload regardless of
  how long the source field is.
- `push.WebPusher.Send` maps a 413 response to a new `push.ErrPayloadTooLarge`
  sentinel (distinct from `ErrGone`: the subscription itself is still
  valid, only this message was too big).
- `jobs.PushDeliveryWorker.Work` treats `ErrPayloadTooLarge` as
  non-retryable: it cancels the job (`river.JobCancel`) instead of letting
  River retry it forever, without touching the `push_subscriptions` row.

## Capabilities

### Modified Capabilities
- `push-notifications`: the push body is bounded before enqueue, and a
  413 (payload too large) response stops that job from retrying instead
  of retrying forever.

## Impact

- `backend/internal/jobs/notification_worker.go` (+ its tests).
- `backend/internal/push/push.go`, `webpush.go` (+ `webpush_test.go`).
- `backend/internal/jobs/push_worker.go` (+ `push_worker_test.go`).
- `backend/internal/metrics/business.go` (new counter for cancelled
  oversized-payload jobs, mirroring `PushSubscriptionsPruned`).
- No API, migration, or RBAC changes.
