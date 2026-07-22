## 1. Database
- [ ] 1.1 New migration `00028_self_registration.sql`: `users.email_verified_at TIMESTAMPTZ`, backfill existing rows to `created_at` (treated as verified), new `email_verification_tokens` table (hashed token, `user_id` FK `ON DELETE CASCADE`, `expires_at`, `consumed_at`)
- [ ] 1.2 Indices on `email_verification_tokens(user_id)` / `(expires_at)`, split into a `CONCURRENTLY` migration if that's this repo's established convention for new indexes
- [ ] 1.3 `make migrate` locally; confirm `migration-rollback` (up→down→up) and `migration-safety` gates pass

## 2. Mailer package
- [ ] 2.1 `internal/mailer/mailer.go`: `Mailer` interface (`SendVerificationEmail`)
- [ ] 2.2 `internal/mailer/smtp.go`: `SMTPMailer` (stdlib `net/smtp`, explicit STARTTLS)
- [ ] 2.3 `internal/mailer/fake.go`: `FakeMailer` (logs the link, exposes a test accessor)
- [ ] 2.4 Wire into `cmd/server/main.go` (`initMailer`, fallback to `FakeMailer` when `SMTP_HOST` unset)

## 3. Config
- [ ] 3.1 `SMTP_HOST/PORT/USERNAME/PASSWORD/FROM_ADDRESS`, `loadSMTPConfig` required-when-`COOKIE_SECURE=true` (`ErrSMTPConfigRequired`)
- [ ] 3.2 `SELF_REGISTRATION_ENABLED` (default true), `EMAIL_VERIFICATION_TTL_HOURS` (default 48), `REGISTER_RATE_LIMIT_PER_MIN` (default 5), `RESEND_VERIFICATION_RATE_LIMIT_PER_MIN` (default 3), `RETENTION_UNVERIFIED_ACCOUNTS_DAYS` (default 7)

## 4. OpenAPI
- [ ] 4.1 `POST /auth/register`, `POST /auth/verify-email`, `POST /auth/resend-verification` (all `security: []`, no `x-rbac-*`), new schemas `RegisterRequest`, `RegisterResponse`, `VerifyEmailRequest`, `ResendVerificationRequest`; `verify-email` reuses `LoginResponse`
- [ ] 4.2 `cd backend && make generate` (commit `internal/gen/api.gen.go`)
- [ ] 4.3 repo-root `make generate-ts` (commit `frontend/src/api/types.gen.ts`)

## 5. Backend auth: validate/repository/service/handler
- [ ] 5.1 `validate.PasswordStrength`: add a 72-byte bound alongside the existing 8–128 rune bound
- [ ] 5.2 `auth/model.go`: `UserRow.EmailVerifiedAt *time.Time`; `auth/repository.go`: `selectUserFields`/`scanUser` include it; new `CreateUnverifiedUser` (`ON CONFLICT (email) DO NOTHING`), `CreateEmailVerificationToken`, `FindEmailVerificationToken`, `ConsumeEmailVerificationToken`, `MarkEmailVerified`, `FindUserVerificationStatusByEmail`
- [ ] 5.3 `auth/service.go`: `Register` (3-case enumeration-safe logic per design.md), `VerifyEmail` (consume token, mark verified, reuse `Login`'s session-creation tail), `ResendVerification` (uniform response), `Login` rejects unverified accounts with `ErrEmailNotVerified`
- [ ] 5.4 `auth/handler.go`: `Register`/`VerifyEmail`/`ResendVerification` handlers, `Login` maps `ErrEmailNotVerified` distinctly, audit events, `metrics.RegisterAttempts`
- [ ] 5.5 `auth/cookie.go`: `applyCookie` gains `case "VerifyEmail"`
- [ ] 5.6 `cmd/server/main.go`: `PerIPRateLimit` wiring for register/resend-verification using the new config values

## 6. Retention job
- [ ] 6.1 5th phase in `RetentionWorker.Work` deleting never-verified `users` past cutoff + lighter cleanup of merely-expired-but-not-yet-cutoff tokens
- [ ] 6.2 `Timeout()` multiplier `4*` → `5*`

## 7. Frontend
- [ ] 7.1 `context/urlState.ts`: `parseVerifyEmailToken`
- [ ] 7.2 `context/AppContext.tsx`: `doRegister`, `doResendVerification`, bootstrap-effect verify-email branch reusing `establishSession`
- [ ] 7.3 `features/auth/components/Register.tsx` + `Login.tsx` toggle + `features/auth/index.ts` export
- [ ] 7.4 `services/serviceLayerReal.ts`: `auth.register/verifyEmail/resendVerification`
- [ ] 7.5 `mocks/handlers.ts` + `mocks/db.ts`: enumeration-safe MSW handlers for all three endpoints
- [ ] 7.6 `i18n/en.ts` + `i18n/de.ts`: register/verify/resend/feature-disabled copy

## 8. Tests
- [ ] 8.1 Backend: `validate.PasswordStrength` byte/rune edge case; `Register` all 3 enumeration cases; `VerifyEmail` valid/expired/consumed/wrong-user token; `Login` unverified rejection (+ verified regression); `ResendVerification` uniform response; `RetentionWorker` never-verified cleanup + `Timeout()` update; `config` SMTP required-when-secure + new env var parsing
- [ ] 8.2 Frontend: `Register.tsx`/`Login.tsx` toggle; `serviceContract.test.ts` new scenarios; `AppContext` bootstrap verify-email branch

## 9. Docs
- [ ] 9.1 `CLAUDE.md` env var table: SMTP_*, `SELF_REGISTRATION_ENABLED`, `EMAIL_VERIFICATION_TTL_HOURS`, `REGISTER_RATE_LIMIT_PER_MIN`, `RESEND_VERIFICATION_RATE_LIMIT_PER_MIN`, `RETENTION_UNVERIFIED_ACCOUNTS_DAYS`

## 10. Verification
- [ ] 10.1 `openspec validate self-service-registration --strict`
- [ ] 10.2 `cd backend && make generate` / repo-root `make generate-ts` — no diff
- [ ] 10.3 `cd backend && make lint`
- [ ] 10.4 `cd backend && make test` (unit + integration)
- [ ] 10.5 `govulncheck`
- [ ] 10.6 `migration-rollback` / `migration-safety` on the new migration(s)
- [ ] 10.7 `backend-openapi-drift`
- [ ] 10.8 `cd frontend && npm run lint && npm run typecheck && npm test && npm run build`
- [ ] 10.9 Grep confirms no raw verification token is ever logged or persisted unhashed
