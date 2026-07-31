## 1. Backend

- [x] 1.1 `webpush.go`: add a case-insensitive substring check for known
      VAPID key-mismatch response signatures (Mozilla errno 109 /
      "VAPID public key mismatch"; FCM's "credentials used to create the
      subscription[s]") on a 401/403 response, returning `ErrGone` when
      matched instead of falling through to `ErrPushServiceStatus`
- [x] 1.2 `webpush_test.go`: cover a 401 with the Mozilla-style body and a
      403 with the FCM-style body both mapping to `ErrGone`, and confirm
      an unrelated 401 body (e.g. the existing "Invalid VAPID token" test)
      still falls through to `ErrPushServiceStatus`

## 2. Docs

- [x] 2.1 `docs/operations.md`'s Web Push troubleshooting entry: describe
      the new behavior — a key-mismatch-specific 401/403 is now pruned
      automatically like a 404/410, while other 401 causes (malformed
      `VAPID_SUBJECT`, a config-wide key mismatch not yet reflected in any
      subscription) still recur until fixed

## 3. Verification

- [x] 3.1 `cd backend && make test-unit` green
- [x] 3.2 `cd backend && make lint` green
