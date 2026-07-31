## 1. Backend

- [x] 1.1 `notification_worker.go`: bound `pushPayloadForNotification`'s
      body to a fixed rune count, truncating with an ellipsis when the
      source text exceeds it
- [x] 1.2 `push.go`: add `ErrPayloadTooLarge` sentinel with a doc comment
      explaining it's distinct from `ErrGone` (subscription still valid)
- [x] 1.3 `webpush.go`: map a 413 response to `ErrPayloadTooLarge`
      (wrapped with the response body snippet, same as the existing
      diagnostic path)
- [x] 1.4 `push_worker.go`: treat `ErrPayloadTooLarge` as non-retryable —
      `river.JobCancel` instead of falling through to the generic
      retry-forever return, without deleting the subscription
- [x] 1.5 `metrics/business.go`: add `PushPayloadTooLarge` counter,
      incremented alongside the cancel in 1.4

## 2. Tests

- [x] 2.1 `notification_worker_test.go` (or a new table test): a body
      longer than the cap is truncated with an ellipsis; a shorter body
      is unchanged
- [x] 2.2 `webpush_test.go`: a 413 response maps to `ErrPayloadTooLarge`
- [x] 2.3 `push_worker_test.go`: `ErrPayloadTooLarge` from the pusher
      results in a cancelled job (`errors.As` into `*river.JobCancelError`)
      and the deleter is never called

## 3. Verification

- [x] 3.1 `cd backend && make test-unit` green
- [x] 3.2 `cd backend && make lint` green
