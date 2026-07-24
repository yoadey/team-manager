## Why

Teamverwaltung has no way for a new user to create their own account. Every
`users` row today is provisioned entirely out-of-band: `users.password_hash`
is nullable (`internal/db/migrations/00001_init.sql:19`) specifically to
allow OIDC-only accounts, but no OIDC integration exists either, and no
production code path anywhere inserts into `users` — every
`INSERT INTO users` in the repository is inside a `_test.go` file. Team
invite acceptance (`teams.Repository.AcceptInvite`,
`internal/teams/repository.go:632-723`) only creates a `memberships` row for
an **already-authenticated existing** user; it never creates an account.
`POST /teams` (`createTeam`, `openapi/openapi.yaml:212-`) is already open to
any authenticated user, so once an account exists, joining or starting a
team is not the gap — creating the account in the first place is.

The product owner originally planned to solve this via OIDC only, but has
decided to also support direct self-service registration with email +
password, on the condition that it follows "a secure flow" — i.e. it must
not let an attacker squat someone else's email address, must not leak which
emails already have accounts, and must be operable (rate-limited,
disableable) in production the same way every other auth-adjacent feature in
this codebase is.

## What Changes

- New `POST /auth/register`: creates an unverified account (email + bcrypt
  password hash) and emails a verification link. Response is identical
  regardless of whether the email was available, already registered and
  verified, or already registered and still pending — so the endpoint never
  reveals account existence.
- New `POST /auth/verify-email`: consumes a single-use, time-limited
  verification token, marks the account verified, and returns a session
  (same `token`+`user` shape as `login`) so the frontend can reuse its
  existing post-login bootstrap.
- New `POST /auth/resend-verification`: always returns the same generic
  response regardless of account state; only actually sends mail for a
  still-unverified account.
- `Login` now rejects a correct email/password pair when the account has
  never been verified, with a response distinct from "wrong credentials".
- New `internal/mailer` package (interface + SMTP implementation + an
  in-memory/logging fake for dev, mirroring `internal/storage`'s
  `ObjectStore` pattern) — the first outbound-email capability in this
  backend.
- New `users.email_verified_at` column and `email_verification_tokens`
  table (hashed tokens at rest, mirroring `sessions.token_hash`).
- New `SELF_REGISTRATION_ENABLED` server-side kill switch (default `true`).
- A new retention-job phase deletes accounts that never completed
  verification, so a squatted email address eventually becomes available
  again.

## Capabilities

### New Capabilities
- `user-registration`: self-service account creation, email verification,
  resend, unverified-login rejection, enumeration-safety, and cleanup of
  abandoned registrations.

### Modified Capabilities
<!-- none -->

## Impact

- Backend: `internal/auth/{service.go,repository.go,handler.go,cookie.go,model.go}`,
  new `internal/mailer/{mailer.go,smtp.go,fake.go}`, `internal/validate/validate.go`,
  `internal/config/config.go`, `internal/jobs/retention.go`,
  `internal/metrics/business.go`, `internal/audit/audit.go`,
  `cmd/server/main.go` (mailer wiring, new routes, rate limiting).
- Database: new migration `internal/db/migrations/00028_self_registration.sql`
  (+ a follow-up `CONCURRENTLY` index migration if required by this repo's
  convention).
- API contract: `backend/openapi/openapi.yaml` (three new operations, three
  new schemas), regenerated `internal/gen/api.gen.go` and
  `frontend/src/api/types.gen.ts`.
- Frontend: `context/{urlState.ts,AppContext.tsx}`,
  `features/auth/components/{Login.tsx,Register.tsx}`,
  `services/serviceLayerReal.ts`, `mocks/{handlers.ts,db.ts}`,
  `services/serviceContract.test.ts`, `i18n/{en.ts,de.ts}`.
- Docs: `CLAUDE.md` env var table (SMTP_*, `SELF_REGISTRATION_ENABLED`,
  `EMAIL_VERIFICATION_TTL_HOURS`, `REGISTER_RATE_LIMIT_PER_MIN`,
  `RESEND_VERIFICATION_RATE_LIMIT_PER_MIN`, `RETENTION_UNVERIFIED_ACCOUNTS_DAYS`).
