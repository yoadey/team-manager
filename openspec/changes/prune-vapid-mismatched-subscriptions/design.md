## Context

`push.WebPusher.Send` (`backend/internal/push/webpush.go`) already reads a
bounded snippet of a non-2xx response body for diagnostics
(`push-delivery-error-diagnostics`). `jobs.PushDeliveryWorker` treats
`push.ErrGone` as "prune the subscription, don't retry" and any other
error as "retry via River's backoff." Today only a 404/410 maps to
`ErrGone`; every 401/403 falls into the retry-forever bucket, on the
reasoning that a VAPID auth failure is server-wide, not per-subscription.

Production logs show that reasoning doesn't hold for every 401/403: some
carry a response body that explicitly says *this subscription's* stored
key doesn't match ours (Mozilla autopush errno 109 `"VAPID public key
mismatch"`; FCM's plain-text `"...do not correspond to the credentials
used to create the subscription[s]"`). That's a permanent, per-subscription
condition — retrying changes nothing, and doing so anyway means these jobs
sit in River's retry queue (and error log) for as long as `max-attempts`
allows.

## Goals

- Stop retrying (and logging) a delivery that can never succeed because
  the specific subscription was created against a VAPID key this server
  no longer uses.
- Leave every other 401/403 cause (malformed `VAPID_SUBJECT`, a
  server-wide key misconfiguration not yet reflected in any subscription)
  on the existing retry-forever path, since a config fix there really
  does recover all subscriptions at once — pruning them would force every
  affected user to manually re-enable push for nothing.
- No new sentinel error or exported API: reuse `ErrGone` so
  `PushDeliveryWorker` needs no changes at all.

## Decisions

- **Match on known response-body signatures, not on status code alone.**
  A 401/403 is ambiguous by status code; the push service's own body text
  is what actually distinguishes "this subscription is stale" from "your
  server's VAPID config is wrong." Matching text is more precise than
  guessing from status code, and keeps the change scoped to the two
  concrete signatures observed in production rather than reopening the
  prior "treat all 401/403 as retryable" decision wholesale.
- **Case-insensitive substring match, checked before the existing
  diagnostic-snippet formatting.** Push services don't guarantee exact
  casing or JSON structure (Mozilla is JSON, FCM is a plain-text body) —
  substring matching on the already-bounded (2 KB) body is simpler and
  more robust than per-vendor JSON parsing, and costs nothing extra since
  the body is already read for the diagnostic path.
- **Reuse `ErrGone` rather than adding a distinct sentinel.** The recovery
  action is identical to a 404/410 (delete the row, stop retrying); a
  second sentinel would only duplicate `PushDeliveryWorker`'s handling and
  the `PushSubscriptionsPruned` metric for no behavioral difference.
- **Do not touch the 413 "payload too large" failures also visible in the
  same production logs.** That's an unrelated failure mode (message size,
  not subscription validity) and out of scope for this VAPID-key-mismatch
  fix; left for separate follow-up.

## Risks

- **A false-positive substring match would wrongly prune a still-valid
  subscription.** Mitigated by matching fairly specific phrases
  ("vapid public key mismatch", "credentials used to create the
  subscription") that are vendor error-code language, not generic words
  likely to appear in an unrelated 401 body.
- **Vendor wording could change.** If a push service rewords its error
  text, the affected subscriptions fall back to the pre-existing
  retry-forever behavior (no regression, just a missed optimization) —
  not a new failure mode.
