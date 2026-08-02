## Why

An operator reported `jobs.PushDeliveryWorker: push.WebPusher.Send: push:
push service returned a non-2xx status: status 401` in production logs. A
401 from a browser push service (Mozilla autopush, FCM, ...) almost always
means the VAPID authentication itself is rejected — most commonly a
`VAPID_PUBLIC_KEY` mismatch between what the frontend used to create the
subscription and what the backend now signs with (e.g. after rotating the
backend's VAPID keypair without every browser re-subscribing), or a
malformed `VAPID_SUBJECT`. But `push.WebPusher.Send`
(`backend/internal/push/webpush.go`) currently discards the push service's
response body on a non-2xx status, so the log line carries only the HTTP
status code — push services (Mozilla, FCM) return a descriptive JSON body
(`{"code":401,"errno":109,"message":"..."}`) on auth failures that would
otherwise pinpoint the cause immediately. There is also no documented
troubleshooting entry for a 401 specifically, only the general VAPID
configuration reference.

## What Changes

- `push.WebPusher.Send` reads (bounded) and includes the push service's
  response body in the error it returns on a non-2xx, non-404/410 status,
  so the body reaches the job-queue error log without any new logging
  call.
- `docs/operations.md`'s Web Push section gets a short troubleshooting
  entry for a 401 response, pointing at VAPID key mismatch (frontend vs.
  backend `VAPID_PUBLIC_KEY`, or a rotated backend keypair with no
  resubscription) as the most likely cause, since VAPID keys have no
  rotation story today (unlike `COOKIE_ENCRYPTION_KEYS`/`JWT_*`).

## Capabilities

### Modified Capabilities
- `push-notifications`: delivery failures now carry enough detail in the
  job-queue error log to diagnose a VAPID auth rejection without adding
  request logging or reproducing the failure by hand.

## Impact

- `backend/internal/push/webpush.go` (+tests in `webpush_test.go`).
- `docs/operations.md` (Web Push section — troubleshooting entry only, no
  env var/behavior change).
- No API, migration, or RBAC changes.
