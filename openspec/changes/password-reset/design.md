## Context

`users.password_hash` is nullable (OIDC-ready, though no OIDC integration
exists yet), so an account may legitimately have no password to reset.
`Login` already defends against user enumeration via a constant-time dummy
bcrypt compare (`dummyPasswordHash`); `self-service-registration` extended
that property to account creation (`Register`/`ResendVerification`, uniform
generic responses). This change extends the same property to password
recovery. `email_verification_tokens` (hashed token at rest, single-use via
`consumed_at`, TTL via `expires_at`) is the direct structural precedent for
the new `password_reset_tokens` table; `mailer.Mailer` is the direct
precedent for adding `SendPasswordResetEmail`.

## Goals / Non-Goals

**Goals:**
- Self-service password recovery via an emailed link, for accounts that have
  a password set.
- Never reveal whether an email has an account, and never reveal that an
  existing account has no password set (OIDC-only) — both must produce the
  same response as a normal request.
- Resetting a password invalidates every existing session for that account,
  not just the browser that completed the reset — a forgotten-password flow
  is also the recovery path for a possibly-compromised account, and a stale
  session on another device (or an attacker's) must not survive the reset.
- Operable in production: rate-limited, and the reset link's TTL is short
  (default 1h) since it grants a credential change, not just an email
  confirmation.

**Non-Goals:**
- No "change password while logged in" endpoint (a separate, smaller feature
  the profile settings screen doesn't currently expose either) — out of
  scope; the authenticated account-erasure/data-export flows in
  `internal/auth` already establish that pattern, but adding it here would
  conflate two independent capabilities.
- No CAPTCHA/bot-detection — rate limiting is the only anti-abuse layer,
  consistent with every other rate-limited auth endpoint in this codebase.
- No change to OIDC (still unimplemented).

## Decisions

**Enumeration-safety matrix for `POST /auth/forgot-password`.** All three
cases below return the same HTTP 202 with an identical generic body
(`{"message": "..."}`), mirroring `Register`'s contract:

1. **No account for this email** — no-op, generic response.
2. **Account exists but has no password set** (`password_hash` is
   NULL/empty — an OIDC-only account, though none exist in production yet)
   — no-op (nothing to reset, and a reset link would let anyone who guesses
   the email set a password on someone else's OIDC-only account), generic
   response.
3. **Account exists with a password** — issue a reset token, email the
   link, generic response.

Unlike `Register` (which needs a bcrypt compare on every branch to keep
timing constant, because hashing itself is the expensive, branch-dependent
step), `ForgotPassword` does no password comparison at all — the only
timing-sensitive operation is `FindUserByEmail`, which already runs
identically whether or not a row is found. No dummy-work compensation is
needed here.

**Token storage.** `password_reset_tokens.token_hash` stores only the
SHA-256 hex digest of the raw token (mirroring `email_verification_tokens`);
the raw token is never persisted or logged. A dedicated table (rather than
reusing `email_verification_tokens` with a "purpose" column) keeps the two
token kinds from ever being cross-redeemable by construction — a verified
row consumed as `WHERE token_hash = $1` on the wrong table simply doesn't
exist, rather than relying on an extra runtime discriminator check that a
future call site could omit.

**Session invalidation on reset.** `ResetPassword` deletes every row in
`sessions` for the user (new `Repository.DeleteSessionsForUser`) before
issuing the fresh one via the existing `createSessionAndSign` helper (the
same tail `Login`/`VerifyEmail` use). This is a deliberate difference from
`VerifyEmail`, which establishes a session without touching any existing
one (there is nothing to revoke — the account was never able to log in
before). A password reset can follow a real compromise, so every other
session (including an attacker's, if the compromise is why the password
is being reset at all) must not survive it.

**TTL.** `PASSWORD_RESET_TTL_HOURS` defaults to 1, deliberately much shorter
than `EMAIL_VERIFICATION_TTL_HOURS`'s 48 — a reset token is a live credential
change, sent to an inbox that may itself be transiently compromised (e.g. a
shared/public device), so the exposure window is minimized. Configurable
per-deployment like every other TTL in this codebase.

**Rate limiting.** `FORGOT_PASSWORD_RATE_LIMIT_PER_MIN` (default 3) mirrors
`RESEND_VERIFICATION_RATE_LIMIT_PER_MIN`'s default and reasoning — bounding
mail-bomb volume via the same email address repeatedly. `reset-password`
itself is not separately rate-limited, mirroring `verify-email`'s existing
reasoning: its token is a high-entropy (32-byte, `crypto/rand`), single-use
secret, not a guessable value, so brute-forcing it is infeasible regardless
of request rate.

**Response shape on success.** `ResetPassword` returns the same
`LoginResponse` (`token`+`user`) shape as `Login`/`VerifyEmail`, and gets its
own `case "ResetPassword":` in `SessionCookieCodec.applyCookie`, so the
frontend can reuse `establishSession` unmodified — consistent with how
`VerifyEmail` was wired into the same mechanism.

**Password strength.** `ResetPassword`'s new password goes through the same
`validate.PasswordStrength` (8–128 runes, 72-byte bcrypt bound) and
`Service.HashPassword` as `Register`/`Login` — no separate policy.

## Risks / Trade-offs

- **Mail delivery failure after the token row is committed**: `ForgotPassword`
  does not roll back the token insert if sending fails (mirrors
  `issueVerificationToken`'s existing best-effort semantics) — the user can
  simply request another reset link.
- **A stale reset link left in an inbox** is bounded by the 1-hour default
  TTL and single-use consumption; the retention job also deletes expired
  rows so they don't accumulate indefinitely.
- **Global session invalidation on every reset** means a user who resets
  their password from a new device is logged out of all their other,
  legitimate devices too, requiring them to log back in there. This is an
  accepted, deliberate trade-off (see Decisions) given the account-recovery
  nature of this flow.
