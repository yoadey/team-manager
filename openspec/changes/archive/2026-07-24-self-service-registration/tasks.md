## 1. Database
- [x] 1.1 New migration `00028_self_registration.sql`: `users.email_verified_at TIMESTAMPTZ`, backfill existing rows to `created_at` (treated as verified), new `email_verification_tokens` table (hashed token, `user_id` FK `ON DELETE CASCADE`, `expires_at`, `consumed_at`)
      Note: `00028_self_registration.sql` and its follow-up index migration (`00029_...`) were later squashed into `internal/db/migrations/00001_init.sql` by the `alpha-initial-setup` migration-squash (commit `a9e4c22`, never deployed beyond an alpha tag) — the schema itself (`users.email_verified_at`, `email_verification_tokens` with `ON DELETE CASCADE`) is present in `00001_init.sql` unchanged.
- [x] 1.2 Indices on `email_verification_tokens(user_id)` / `(expires_at)`, split into a `CONCURRENTLY` migration if that's this repo's established convention for new indexes
      Present in `00001_init.sql` as `CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_email_verification_tokens_user_id`/`idx_email_verification_tokens_expires_at`.
- [ ] 1.3 `make migrate` locally; confirm `migration-rollback` (up→down→up) and `migration-safety` gates pass
      Could not run in this sandbox: no Postgres instance and no Docker daemon available (`docker ps` fails to reach `/var/run/docker.sock`), so neither `make migrate` nor the Docker-backed rollback/safety CI gates can execute here. CI (with Docker + Postgres) should confirm.

## 2. Mailer package
- [x] 2.1 `internal/mailer/mailer.go`: `Mailer` interface (`SendVerificationEmail`)
- [x] 2.2 `internal/mailer/smtp.go`: `SMTPMailer` (stdlib `net/smtp`, explicit STARTTLS)
- [x] 2.3 `internal/mailer/fake.go`: `FakeMailer` (logs the link, exposes a test accessor)
- [x] 2.4 Wire into `cmd/server/main.go` (`initMailer`, fallback to `FakeMailer` when `SMTP_HOST` unset)

## 3. Config
- [x] 3.1 `SMTP_HOST/PORT/USERNAME/PASSWORD/FROM_ADDRESS`, `loadSMTPConfig` required-when-`COOKIE_SECURE=true` (`ErrSMTPConfigRequired`)
- [x] 3.2 `SELF_REGISTRATION_ENABLED` (default true), `EMAIL_VERIFICATION_TTL_HOURS` (default 48), `REGISTER_RATE_LIMIT_PER_MIN` (default 5), `RESEND_VERIFICATION_RATE_LIMIT_PER_MIN` (default 3), `RETENTION_UNVERIFIED_ACCOUNTS_DAYS` (default 7)

## 4. OpenAPI
- [x] 4.1 `POST /auth/register`, `POST /auth/verify-email`, `POST /auth/resend-verification` (all `security: []`, no `x-rbac-*`), new schemas `RegisterRequest`, `RegisterResponse`, `VerifyEmailRequest`, `ResendVerificationRequest`; `verify-email` reuses `LoginResponse`
- [x] 4.2 `cd backend && make generate` (commit `internal/gen/api.gen.go`)
- [x] 4.3 repo-root `make generate-ts` (commit `frontend/src/api/types.gen.ts`)

## 5. Backend auth: validate/repository/service/handler
- [x] 5.1 `validate.PasswordStrength`: add a 72-byte bound alongside the existing 8–128 rune bound
- [x] 5.2 `auth/model.go`: `UserRow.EmailVerifiedAt *time.Time`; `auth/repository.go`: `selectUserFields`/`scanUser` include it; new `CreateUnverifiedUser` (`ON CONFLICT (email) DO NOTHING`), `CreateEmailVerificationToken`, `FindEmailVerificationToken`, `ConsumeEmailVerificationToken`, `MarkEmailVerified`
      Note: no separate `FindUserVerificationStatusByEmail` method exists — `Service.Register`/`ResendVerification` reuse the pre-existing `FindUserByEmail` (checking `EmailVerifiedAt` on the returned row) instead of adding a redundant near-duplicate query. Behavior (never overwrite an existing account/password, distinguish verified vs. pending) matches design.md and is covered by `TestService_Register_EmailTaken_*`/`TestService_ResendVerification_UniformAcrossAllAccountStates`.
- [x] 5.3 `auth/service.go`: `Register` (3-case enumeration-safe logic per design.md), `VerifyEmail` (consume token, mark verified, reuse `Login`'s session-creation tail), `ResendVerification` (uniform response), `Login` rejects unverified accounts with `ErrEmailNotVerified`
- [x] 5.4 `auth/handler.go`: `Register`/`VerifyEmail`/`ResendVerification` handlers, `Login` maps `ErrEmailNotVerified` distinctly, audit events, `metrics.RegisterAttempts`
- [x] 5.5 `auth/cookie.go`: `applyCookie` gains `case "VerifyEmail"`
- [x] 5.6 `cmd/server/main.go`: `PerIPRateLimit` wiring for register/resend-verification using the new config values

## 6. Retention job
- [x] 6.1 5th phase in `RetentionWorker.Work` deleting never-verified `users` past cutoff + lighter cleanup of merely-expired-but-not-yet-cutoff tokens
- [x] 6.2 `Timeout()` multiplier `4*` → `5*`
      Actual multiplier is `6*retentionPhaseTimeout` (not `5*`): the pre-existing `invites` phase (added independently of this change) made the pre-change base 5, not 4, so this change's two new phases (never-verified users + expired tokens) took it from 4→6, not 4→5. Six phases total: notifications, sessions, invites, audit_log, never-verified users, expired verification tokens.

## 7. Frontend
- [x] 7.1 `context/urlState.ts`: `parseVerifyEmailToken`
- [x] 7.2 `context/AppContext.tsx`: `doRegister`, `doResendVerification`, bootstrap-effect verify-email branch reusing `establishSession`
- [x] 7.3 `features/auth/components/Register.tsx` + `Login.tsx` toggle + `features/auth/index.ts` export
- [x] 7.4 `services/serviceLayerReal.ts`: `auth.register/verifyEmail/resendVerification`
- [x] 7.5 `mocks/handlers.ts` + `mocks/db.ts`: enumeration-safe MSW handlers for all three endpoints
- [x] 7.6 `i18n/en.ts` + `i18n/de.ts`: register/verify/resend/feature-disabled copy

## 8. Tests
- [x] 8.1 Backend: `validate.PasswordStrength` byte/rune edge case; `Register` all 3 enumeration cases; `VerifyEmail` valid/expired/consumed/wrong-user token; `Login` unverified rejection (+ verified regression); `ResendVerification` uniform response; `RetentionWorker` never-verified cleanup + `Timeout()` update; `config` SMTP required-when-secure + new env var parsing
      `RetentionWorker` never-verified/expired-token cleanup was missing test coverage on audit — added `TestRetentionWorker_DeletesNeverVerifiedAccountsPastCutoff` and `TestRetentionWorker_DeletesExpiredVerificationTokensButKeepsUnexpired` to `internal/jobs/retention_test.go`; `Timeout()` update was already covered by `TestRetentionWorker_TimeoutExceedsRiverDefault`. Everything else was already present.
- [x] 8.2 Frontend: `Register.tsx`/`Login.tsx` toggle; `serviceContract.test.ts` new scenarios; `AppContext` bootstrap verify-email branch

## 9. Docs
- [x] 9.1 `CLAUDE.md` env var table: SMTP_*, `SELF_REGISTRATION_ENABLED`, `EMAIL_VERIFICATION_TTL_HOURS`, `REGISTER_RATE_LIMIT_PER_MIN`, `RESEND_VERIFICATION_RATE_LIMIT_PER_MIN`, `RETENTION_UNVERIFIED_ACCOUNTS_DAYS`

## 10. Verification
- [x] 10.1 `openspec validate self-service-registration --strict`
      Passes: "Change 'self-service-registration' is valid".
- [x] 10.2 `cd backend && make generate` / repo-root `make generate-ts` — no diff
      Both ran clean; `git status` showed no changes to generated files.
- [x] 10.3 `cd backend && make lint`
      `golangci-lint run ./...` → "0 issues."
- [x] 10.4 `cd backend && make test` (unit + integration)
      Ran `go test ./... -short -race` (unit-only, per sandbox scope) — all packages pass. Integration tests (`testutil.NewTestDB`) auto-skip without Docker; not run here, same limitation as 1.3/10.6.
- [ ] 10.5 `govulncheck`
      Could not run in this sandbox: `govulncheck ./...` fails fetching `https://vuln.go.dev/index/modules.json.gz` (403 Forbidden — no network egress to vuln.go.dev). CI (with network access) should confirm.
- [ ] 10.6 `migration-rollback` / `migration-safety` on the new migration(s)
      Could not run in this sandbox: both are Docker-backed CI gates and no Docker daemon is available here (same limitation as 1.3). CI should confirm.
- [x] 10.7 `backend-openapi-drift`
      Exercised locally via 10.2 (`make generate` + `make generate-ts` produced no diff), which is what this CI job runs.
- [x] 10.8 `cd frontend && npm run lint && npm run typecheck && npm test && npm run build`
      All four green: ESLint 0 warnings, `tsc -b --noEmit` clean, 1161/1161 tests passed across 82 files, production build succeeded.
- [x] 10.9 Grep confirms no raw verification token is ever logged or persisted unhashed
      Grepped `internal/auth/` and `internal/mailer/` for `rawToken`/`tokenHash` usage: every DB write (`CreateEmailVerificationToken`, `sessions.token_hash`) and every log statement in production code paths uses only `tokenHash` (SHA-256 of the raw token), never the raw value. The one place a raw token's *value* is exposed is the verification URL (`verifyURL := s.publicBaseURL + "/verify-email/" + rawToken`) handed to `Mailer.SendVerificationEmail` — `SMTPMailer` sends it as the email body only; `FakeMailer` logs it, but only as the documented, intentional dev/unconfigured-SMTP fallback ("Unset in dev falls back to a logging fake mailer (the verification link is only written to the server log)" — see CLAUDE.md's `SMTP_HOST` row), never in the SMTP-configured/production path.
