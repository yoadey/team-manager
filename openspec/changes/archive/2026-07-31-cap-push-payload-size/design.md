## Context

`jobs.pushPayloadForNotification` (`backend/internal/jobs/notification_worker.go`)
builds the Web Push body from whichever of `EventTitle`/`Title`/`Note` is
set, with no length bound. `CreatePollRequest.question` has no
`maxLength` in `openapi.yaml`, so a poll question of arbitrary length
flows straight into the push body. `push.WebPusher.Send` currently maps
only 404/410 to `ErrGone`; every other non-2xx status (including 413)
returns a generic error that `jobs.PushDeliveryWorker` retries via
River's backoff indefinitely.

## Goals

- Guarantee a push notification body can never itself be the cause of a
  413, regardless of how long the source field is or whether some other
  endpoint later adds a new unbounded field.
- Stop retrying a delivery that failed because the (already-enqueued)
  payload is too large — the payload is fixed once the job is inserted
  (`PushDeliveryArgs` denormalizes it, see that struct's doc comment), so
  retrying changes nothing.
- Do not touch the subscription: unlike a gone/mismatched subscription, a
  413 says nothing about whether the endpoint is still valid — only that
  this particular message was too big.

## Decisions

- **Truncate at the source (`pushPayloadForNotification`), not just react
  to 413 after the fact.** A push notification is a short preview by
  design (platforms show only the first couple of lines regardless), so
  bounding it there fixes the problem for every notification type at
  once, including any future one, rather than special-casing
  `CreatePollRequest.question` in the OpenAPI schema (which would still
  leave the door open for the next unbounded free-text field routed
  through the same code path).
- **Truncate by rune count (200), not byte count**, appending `"…"` when
  truncated. Rune-based truncation can't split a multi-byte UTF-8
  character; 200 runes stays comfortably under any push service's
  message-size limit even at 4 bytes/rune worst case (800 bytes), while
  still being a reasonable notification preview length.
- **New `push.ErrPayloadTooLarge` sentinel, not a reuse of `ErrGone`.**
  Reusing `ErrGone` would delete a subscription that's still perfectly
  usable for the *next* (properly-sized) notification — a different
  failure mode than "this endpoint is gone", so it needs its own
  terminal-but-not-a-prune handling in `PushDeliveryWorker`.
- **`river.JobCancel` to stop retries without deleting anything.** This is
  the same mechanism River itself offers for "this job can never
  succeed, but that's not a signal about anything but the job" — it
  marks the job cancelled (visible in the job table/logs) without
  requiring a new discard/skip convention in `PushDeliveryWorker`.
- **Truncation is defense in depth, not a replacement for validating
  `CreatePollRequest.question`.** Adding a schema `maxLength` there is a
  reasonable separate follow-up (it also matters for storage/display, not
  just push), but out of scope here — this change's job is making sure
  the push path specifically can't 413 regardless of what upstream
  validation does or doesn't do.

## Risks

- **A 200-rune body could truncate mid-word.** Acceptable for a
  notification preview (the in-app notification feed and the source
  content itself are unaffected — only the push body is shortened); a
  trailing `"…"` signals there's more.
- **Metric cardinality**: one new counter
  (`push_payload_too_large_total`), mirroring the existing
  `push_subscriptions_pruned_total` pattern — no new labels.
