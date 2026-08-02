## 1. Frontend

- [ ] 1.1 `monitoring.ts`: `setSentryUser` calls `Sentry.setUser({ id })` only
      — drop `username`/`name` from the payload
- [ ] 1.2 `monitoring.ts`: `beforeSend` also deletes `event.user.username`
      alongside the existing `email`/`ip_address` stripping
- [ ] 1.3 `monitoring.test.ts`: cover that a captured event's `user` object
      contains only `id` after `setSentryUser` + `beforeSend`

## 2. Verification

- [ ] 2.1 `npm run lint`
- [ ] 2.2 `npm run typecheck`
- [ ] 2.3 `npm test`
