## Why

Four small, independent robustness gaps surfaced during a full-codebase
review, none currently exploitable but each removing a defense-in-depth
layer that the surrounding code already relies on elsewhere:

1. `mailer.smtp.go`'s `buildMessage` interpolates `to`/`from` directly
   into `To:`/`From:` headers with no CRLF check of its own. Not
   currently reachable (`validate.Email`/`net/mail.ParseAddress` rejects
   bare CR/LF before any handler calls the mailer), but the mailer
   package has no guard if a future caller skips that upstream
   validation.
2. `retention.go`'s `deleteBatched`/`deleteUnverifiedUsers` build
   `DELETE FROM %s` via `fmt.Sprintf` with an internal, fixed table
   name — safe today, but exactly the shape static scanners (`gosec`)
   flag, generating recurring noise without a suppressing annotation.
3. `auth/handler.go`'s `GetMyPhoto` returns 404 when
   `UserFromContext` fails, instead of 401 — currently unreachable since
   `AuthMiddleware` always runs first, but masks an auth failure as "not
   found" rather than "unauthenticated" if route wiring ever changes.
4. `openapi.yaml`'s `info.version: "1.0.0"` is static and never bumped;
   the real deprecation signal is the out-of-band
   `API_DEPRECATION_DATE`/`Deprecation`/`Sunset` headers and the
   hardcoded `API-Version: v1` response header, so the spec's own
   `version` field is decorative and could mislead a spec consumer.

## What Changes

- `mailer/smtp.go`: reject a `to`/`from`/`subject` containing `\r` or
  `\n` inside `buildMessage`/`send`, independent of caller validation.
- `jobs/retention.go`: add a `//nolint:gosec` annotation (with the
  existing "fixed internal literal" justification already in the code
  comment) to the `fmt.Sprintf`-built `DELETE`/date-column statements.
- `auth/handler.go`: `GetMyPhoto` returns 401 (not 404) when
  `UserFromContext` fails.
- `openapi.yaml`: derive `info.version` from the build version, or drop
  it in favor of the existing path/header versioning scheme (decision
  left to implementation; either removes the misleading static value).

## Capabilities

### Added Capabilities
- `auth-hardening`: outbound email headers are defended against
  injection at the mailer layer itself, not only by upstream validation;
  a missing auth context on an authenticated-only route is reported as
  401, not 404.

## Impact

- `backend/internal/mailer/smtp.go` (+ test).
- `backend/internal/jobs/retention.go`.
- `backend/internal/auth/handler.go` (+ test).
- `backend/openapi/openapi.yaml`.
- No behavior change for any currently-reachable request path.
