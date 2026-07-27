## 1. Backend

- [x] 1.1 `push.WebPusher.Send`: read the response body (bounded, 2 KB) and
      include a truncated, single-line snippet in the error returned for a
      non-2xx/non-404/410 status
- [x] 1.2 `webpush_test.go`: cover a non-2xx response whose body is
      included in the returned error, and that a body larger than the cap
      is truncated rather than causing an error itself

## 2. Docs

- [x] 2.1 `docs/operations.md`'s Web Push section: add a short
      troubleshooting entry for a 401 response (VAPID key mismatch between
      frontend/backend, or a rotated backend keypair with no
      resubscription — no rotation mechanism exists today for VAPID keys)

## 3. Verification

- [x] 3.1 `cd backend && make test-unit` green
- [x] 3.2 `cd backend && make lint` green
