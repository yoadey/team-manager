## 1. Database
- [ ] 1.1 New migration `00014_password_reset_tokens.sql`: `password_reset_tokens` table (id, `user_id` FK `ON DELETE CASCADE`, `token_hash` UNIQUE, `expires_at`, `consumed_at`, `created_at`), `CREATE INDEX CONCURRENTLY` on `(user_id)` and `(expires_at)`
- [ ] 1.2 `make migrate` locally; confirm `migration-rollback` (up→down→up) and `migration-safety` gates pass

## 2. Mailer
- [ ] 2.1 `internal/mailer/mailer.go`: `Mailer` interface gains `SendPasswordResetEmail(ctx, toEmail, resetURL string) error`
- [ ] 2.2 `internal/mailer/smtp.go`: `SMTPMailer.SendPasswordResetEmail`
- [ ] 2.3 `internal/mailer/fake.go`: `FakeMailer.SendPasswordResetEmail` (logs the link, test accessor)

## 3. Config
- [ ] 3.1 `PASSWORD_RESET_TTL_HOURS` (default 1), `FORGOT_PASSWORD_RATE_LIMIT_PER_MIN` (default 3) added to `registrationSettings`/`Config`/`loadRegistrationConfig`

## 4. OpenAPI
- [ ] 4.1 `POST /auth/forgot-password` (`security: []`, new schemas `ForgotPasswordRequest`, reuses `RegisterResponse`), `POST /auth/reset-password` (`security: []`, new schema `ResetPasswordRequest`, reuses `LoginResponse`)
- [ ] 4.2 `cd backend && make generate` (commit `internal/gen/api.gen.go`)
- [ ] 4.3 repo-root `make generate-ts` (commit `frontend/src/api/types.gen.ts`)

## 5. Backend auth: model/repository/service/handler
- [ ] 5.1 `auth/model.go`: `PasswordResetTokenRow`
- [ ] 5.2 `auth/repository.go`: `CreatePasswordResetToken`, `FindPasswordResetToken`, `ConsumePasswordResetToken`, `UpdateUserPassword`, `DeleteSessionsForUser`
- [ ] 5.3 `auth/service.go`: `RegistrationConfig` gains `PasswordResetTTL`; `ErrInvalidResetToken`; `ForgotPassword` (3-case enumeration-safe logic per design.md, reusing `issueVerificationToken`'s pattern for a new `issuePasswordResetToken` helper); `ResetPassword` (consume token, validate/hash new password, `UpdateUserPassword`, `DeleteSessionsForUser`, reuse `createSessionAndSign`)
- [ ] 5.4 `auth/handler.go`: `ForgotPassword`/`ResetPassword` handlers, audit events, `metrics.PasswordResetAttempts`
- [ ] 5.5 `auth/cookie.go`: `applyCookie` gains `case "ResetPassword"`
- [ ] 5.6 `internal/audit/audit.go`: `EventPasswordResetRequest`, `EventPasswordReset`
- [ ] 5.7 `internal/metrics/business.go`: `PasswordResetAttempts` counter
- [ ] 5.8 `cmd/server/main.go`: `PerIPRateLimit` wiring for `/auth/forgot-password`; unauthenticated `/auth/reset-password` route registered after the generated mux

## 6. Retention job
- [ ] 6.1 New phase in `RetentionWorker.Work` deleting expired `password_reset_tokens` (mirrors the existing expired-`email_verification_tokens` phase)
- [ ] 6.2 `Timeout()` multiplier bumped for the new phase

## 7. Frontend
- [ ] 7.1 `context/urlState.ts`: `parseResetPasswordToken`
- [ ] 7.2 `context/AppContext.tsx`: `doForgotPassword`, `doResetPassword`; bootstrap-effect reset-password branch (shows the reset form pre-populated with the token, does NOT auto-submit); `AppState.resetPasswordToken`
- [ ] 7.3 `features/auth/components/ForgotPassword.tsx` (mirrors `Register`'s check-your-email confirmation UX) + `ResetPassword.tsx` (new password + confirm, submits via `doResetPassword`) + `Login.tsx` "Forgot password?" link + `features/auth/index.ts` exports
- [ ] 7.4 `services/serviceLayerReal.ts`: `auth.forgotPassword`/`auth.resetPassword`
- [ ] 7.5 `mocks/handlers.ts` + `mocks/db.ts`: enumeration-safe MSW handlers for both endpoints, `passwordResetTokens` store
- [ ] 7.6 `i18n/en.ts` + `i18n/de.ts`: forgot/reset-password copy

## 8. Tests
- [ ] 8.1 Backend: `Repository` new methods; `Service.ForgotPassword` all 3 enumeration cases (no account / no password / has password); `Service.ResetPassword` valid/expired/consumed/wrong token, session invalidation, weak-password rejection; `Handler` endpoints incl. audit + metrics; `RetentionWorker` new phase + `Timeout()`; `config` new env var parsing
- [ ] 8.2 Frontend: `ForgotPassword.tsx`/`ResetPassword.tsx`/`Login.tsx` link; `serviceContract.test.ts` new scenarios

## 9. Docs
- [ ] 9.1 `CLAUDE.md` env var table: `PASSWORD_RESET_TTL_HOURS`, `FORGOT_PASSWORD_RATE_LIMIT_PER_MIN`

## 10. Verification
- [ ] 10.1 `openspec validate password-reset --strict`
- [ ] 10.2 `cd backend && make generate` / repo-root `make generate-ts` — no diff
- [ ] 10.3 `cd backend && make lint`
- [ ] 10.4 `cd backend && make test`
- [ ] 10.5 `cd frontend && npm run lint && npm run typecheck && npm test && npm run build`
