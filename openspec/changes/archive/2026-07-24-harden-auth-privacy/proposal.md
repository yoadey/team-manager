## Why

Three auth/privacy findings from the security audit remain open:

1. **Plaintext email persists in `audit_log`.** Login events record `slog.String("email", …)` (`auth/handler.go:85,93`) into `audit_log.attrs`, retained for `RETENTION_AUDIT_LOG_DAYS` (default 365). GDPR erasure (`EraseUser`) anonymizes the `users` row but never touches `audit_log`, so an erased user's email (and any mistyped third-party address from a failed login) stays queryable for up to a year.
2. **bcrypt silently truncates passwords over 72 bytes** — no length check before `bcrypt.GenerateFromPassword`/`CompareHashAndPassword` (`auth/service.go:261,136,140`), so a long passphrase's tail is ignored.
3. **CSRF Origin check allows a missing `Origin`** on state-changing requests (`middleware.go:297-299`).

## What Changes

- Stop persisting the raw email in audit attrs; store a keyed HMAC hash (`email_hash`) instead, preserving brute-force correlation without plaintext PII. Optionally scrub existing rows for a user on erasure.
- Reject passwords longer than 72 bytes with a validation error before bcrypt.
- Harden the CSRF fallback: for cookie-authenticated mutations, treat a request with neither `Origin` nor `Sec-Fetch-Site` as suspicious.

## Capabilities

### New Capabilities
- `auth-hardening`: password-length handling, CSRF fallback strength, and PII minimization in the audit log.

### Modified Capabilities
<!-- none -->

## Impact

- Backend: `auth/handler.go` (audit hash), `auth/service.go` (bcrypt length guard), `middleware/middleware.go` (CSRF), `audit/*` if a hashing helper is added, `auth/repository.go` (optional audit scrub on erase), tests.
- Config: an audit HMAC key (reuse an existing key or add one, documented in `CLAUDE.md`).
