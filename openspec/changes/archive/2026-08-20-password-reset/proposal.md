## Why

A user who forgets their password has no way to recover their account today.
`internal/auth` has a full password-login path (`Service.Login`,
`bcrypt`-hashed `users.password_hash`) and, since `self-service-registration`,
a complete email-token infrastructure (`email_verification_tokens`, the
`mailer.Mailer` interface, `SMTPMailer`/`FakeMailer`) — but no code path ever
updates an existing user's `password_hash` after account creation. The only
way to recover from a forgotten password today is an operator manually
updating the row in the database, which does not scale and is not
self-service. `self-service-registration`'s own design.md flagged this
explicitly under Risks: "No password-reset flow ... flagged for a follow-up
change." This is that follow-up.

## What Changes

- New `POST /auth/forgot-password`: given an email address, emails a
  password-reset link if (and only if) a password-based account exists for
  it. The response is identical regardless of whether the email has no
  account, an OIDC-only account with no password set, or a real
  password account — mirroring `Register`/`ResendVerification`'s
  enumeration-safety contract, so this endpoint never reveals account
  existence.
- New `POST /auth/reset-password`: consumes a single-use, time-limited reset
  token and sets a new password. On success, **every existing session for
  that account is invalidated** (not just the one making the request) and a
  fresh session is returned in the same shape as `login`'s response, so the
  client can reuse its normal post-login bootstrap on the device that
  performed the reset.
- New `users.password_hash`-mutating repository method (`UpdateUserPassword`)
  — the first one in the codebase; every existing write to that column is a
  fresh `INSERT` (registration) or a raw SQL statement in a `_test.go` file.
- New `password_reset_tokens` table, structurally mirroring
  `email_verification_tokens` (hashed token at rest, single-use, expiring),
  but on its own short TTL — a reset token grants a credential change, not
  just an email confirmation, so it defaults to 1 hour instead of 48.
- `mailer.Mailer` gains `SendPasswordResetEmail`, implemented in both
  `SMTPMailer` and `FakeMailer` alongside the existing
  `SendVerificationEmail`.
- New `PASSWORD_RESET_TTL_HOURS` (default 1) and
  `FORGOT_PASSWORD_RATE_LIMIT_PER_MIN` (default 3) env vars, following the
  same operability conventions (configurable, rate-limited) as every other
  auth-adjacent feature in this codebase.
- The retention job gains a phase deleting expired `password_reset_tokens`,
  mirroring the existing expired-`email_verification_tokens` phase.
- Frontend: a "Forgot password?" link on the password login form, a
  `ForgotPassword` request form (mirrors `Register`'s "check your email"
  confirmation UX), and a `ResetPassword` form reached via a
  `/reset-password/<token>` link that sets a new password and logs the user
  in.

## Capabilities

### New Capabilities
- `password-reset`: self-service password recovery via emailed reset link —
  request flow, enumeration-safety, token single-use/expiry, session
  invalidation on reset, rate limiting, retention cleanup.

## Impact

- Backend: `internal/auth/{model.go,repository.go,service.go,handler.go,cookie.go}`,
  `internal/mailer/{mailer.go,smtp.go,fake.go}`, `internal/config/config.go`,
  `internal/jobs/retention.go`, `internal/metrics/business.go`,
  `internal/audit/audit.go`, `cmd/server/main.go` (routes + rate limiting).
- Database: new migration `internal/db/migrations/00014_password_reset_tokens.sql`.
- API contract: `backend/openapi/openapi.yaml` (two new operations, three new
  schemas), regenerated `internal/gen/api.gen.go` and
  `frontend/src/api/types.gen.ts`.
- Frontend: `context/{urlState.ts,AppContext.tsx}`,
  new `features/auth/components/{ForgotPassword.tsx,ResetPassword.tsx}`,
  `features/auth/components/Login.tsx`, `features/auth/index.ts`,
  `services/serviceLayerReal.ts`, `mocks/{handlers.ts,db.ts}`,
  `services/serviceContract.test.ts`, `i18n/{en.ts,de.ts}`.
- Docs: `CLAUDE.md` env var table (`PASSWORD_RESET_TTL_HOURS`,
  `FORGOT_PASSWORD_RATE_LIMIT_PER_MIN`).
