## Context

`users` is a global (not team-scoped) table. `password_hash` is nullable so
an OIDC-only account could exist without one, but no such account-creation
path exists yet — this change introduces the first one. Login
(`auth.Service.Login`) already defends against user enumeration via a
constant-time dummy bcrypt compare (`dummyPasswordHash`) on both "user not
found" and "wrong password"; registration must extend the same property to
account-creation. `bcrypt`'s 72-byte input limit is already enforced at the
`HashPassword`/`Login` layer (`maxPasswordBytes`, `ErrPasswordTooLong` in
`auth/service.go`) as of `harden-auth-privacy`, but `validate.PasswordStrength`
(used for request-level pre-validation) only bounds *rune* count (8–128), so
a long multi-byte password can still pass that check and then fail bcrypt's
byte check with a less clean error.

## Goals / Non-Goals

**Goals:**
- Self-service signup with email + password, consistent with the existing
  login identifier (no separate username field).
- Real email verification before an account can log in.
- Registration and resend never reveal whether an email is already
  registered, mirroring `Login`'s existing enumeration defenses.
- Operable in production: SMTP required when `COOKIE_SECURE=true` (same
  pattern as `S3_*`/`JWT_*`), rate-limited, and disableable via
  `SELF_REGISTRATION_ENABLED`.
- Abandoned (never-verified) registrations are eventually cleaned up so the
  email address becomes reusable.

**Non-Goals:**
- No separate username field — email remains the sole identifier.
- No changes to OIDC (still unimplemented) or to team creation/invite flows
  (`POST /teams` and invite acceptance already work for any authenticated
  user).
- No password-reset flow (out of scope; noted as a risk below).
- No CAPTCHA/bot-detection — rate limiting is the only anti-abuse layer for
  request volume.

## Decisions

**Enumeration-safety matrix for `POST /auth/register`.** All three cases
return the same HTTP 202 with an identical generic body
(`{"message": "..."}`), and a bcrypt hash of the submitted password is
computed unconditionally on every path (mirroring `Login`'s
`dummyPasswordHash` trick) so response timing doesn't distinguish branches:

1. **Email available** — insert a new `users` row
   (`email_verified_at = NULL`, real bcrypt hash), insert an
   `email_verification_tokens` row, send the verification email.
2. **Email taken, already verified** — do not touch the row, do not
   overwrite the password. No email is sent for v1 (an audit log entry is
   recorded instead) — the response is identical either way, so this choice
   has no enumeration impact; it only affects whether the legitimate owner
   gets a courtesy notice.
3. **Email taken, still unverified (pending)** — the `UNIQUE(email)`
   constraint rejects a second insert; that conflict is treated as this
   case. The existing `password_hash` is **never** overwritten (an attacker
   without inbox access must not be able to hijack a pending registration by
   re-registering with a password they chose). Instead, a **fresh**
   verification token is issued and emailed to the same address — equivalent
   to a resend. A user who mistyped their password and immediately
   re-registers before verifying will need to wait for account cleanup and
   re-register from scratch, since there is no password-reset flow (see
   Risks).

The same generic-response principle applies to `POST /auth/resend-verification`.

**Token storage.** `email_verification_tokens.token_hash` stores only the
SHA-256 hex digest of the raw token (mirroring `sessions.token_hash`); the
raw token is never persisted or logged. Consumption is a single
`UPDATE ... WHERE consumed_at IS NULL` guard for atomicity against a
double-submit race.

**Session on verify.** `VerifyEmail` returns the same `LoginResponse` shape
(`token`+`user`) as `Login`, reusing `Login`'s session-creation tail (JWT
signing + `CreateSession`) via a shared private helper, and gets its own
`case "VerifyEmail":` in `SessionCookieCodec.applyCookie` alongside
`"Login"`. This lets the frontend reuse its existing `establishSession`
mechanism unmodified, including redeeming a pending invite parsed from the
URL.

**Password byte/rune reconciliation.** `validate.PasswordStrength` gains an
explicit byte-length bound (72, matching `maxPasswordBytes`) so an
over-length multi-byte password is rejected with a clear validation error at
the request-validation layer, instead of only failing later inside
`HashPassword` with a less specific error.

**Retention.** The existing `RetentionWorker` (`internal/jobs/retention.go`)
gains a fifth phase deleting `users` rows where `email_verified_at IS NULL`
and `created_at` is older than a new, independently configurable
`RETENTION_UNVERIFIED_ACCOUNTS_DAYS` (unlike `inviteRetention`, this governs
when an email address becomes reusable — a more consequential choice worth
its own knob). Deleting the `users` row cascades to
`email_verification_tokens` and `sessions` via existing/new `ON DELETE
CASCADE` foreign keys.

## Risks / Trade-offs

- **No password-reset flow.** A user who mistypes their password at
  registration and doesn't notice until after clicking "verify" has no
  self-service way to change it before this change ships anything for that
  — they must wait for the never-verified cleanup job and start over, or an
  operator resets it by hand. Explicitly out of scope; flagged for a
  follow-up change.
- **SMTP misconfiguration in production** would silently prevent every new
  signup from ever becoming usable. Mitigated by making SMTP config
  required-at-startup when `COOKIE_SECURE=true`, exactly like `S3_*`/`JWT_*`.
- **Mail delivery latency/failure after the `users` row is already
  committed**: `Register` does not roll back the insert if sending fails —
  the row stays reachable via `resend-verification`, and eventually via
  retention cleanup if never verified.
- **Squatting an email address before verifying** is bounded by the new
  retention phase (default cleanup window) rather than prevented outright;
  until cleanup runs, the real owner cannot register that address, only
  request a fresh verification resend to it (case 3 above), which still
  requires inbox access to actually use.
