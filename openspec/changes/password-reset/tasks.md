## 1. Database
- [x] 1.1 New migration `00014_password_reset_tokens.sql`: `password_reset_tokens` table (id, `user_id` FK `ON DELETE CASCADE`, `token_hash` UNIQUE, `expires_at`, `consumed_at`, `created_at`), `CREATE INDEX CONCURRENTLY` on `(user_id)` and `(expires_at)`
      Split into two migrations (`00014_password_reset_tokens.sql` for the table, `00015_password_reset_tokens_indexes.sql` with `-- +goose NO TRANSACTION` for the `CONCURRENTLY` indexes), matching this repo's established convention (see `00002`/`00003`, `00004`/`00005` pairs) rather than combining both in one file as `00001_init.sql`'s single bootstrap migration does.
- [ ] 1.2 `make migrate` locally; confirm `migration-rollback` (up→down→up) and `migration-safety` gates pass
      Could not run in this sandbox: no Postgres instance and no Docker daemon available, so neither `make migrate` nor the Docker-backed rollback/safety CI gates can execute here. CI (with Docker + Postgres) should confirm.

## 2. Mailer
- [x] 2.1 `internal/mailer/mailer.go`: `Mailer` interface gains `SendPasswordResetEmail(ctx, toEmail, resetURL string) error`
- [x] 2.2 `internal/mailer/smtp.go`: `SMTPMailer.SendPasswordResetEmail`
- [x] 2.3 `internal/mailer/fake.go`: `FakeMailer.SendPasswordResetEmail` (logs the link, test accessor)

## 3. Config
- [x] 3.1 `PASSWORD_RESET_TTL_HOURS` (default 1), `FORGOT_PASSWORD_RATE_LIMIT_PER_MIN` (default 3) added to `registrationSettings`/`Config`/`loadRegistrationConfig`
      Also wired into `helm/team-manager/{values.yaml,values.schema.json,templates/_env.tpl,README.md}` alongside the existing self-registration settings.

## 4. OpenAPI
- [x] 4.1 `POST /auth/forgot-password` (`security: []`, new schemas `ForgotPasswordRequest`, reuses `RegisterResponse`), `POST /auth/reset-password` (`security: []`, new schema `ResetPasswordRequest`, reuses `LoginResponse`)
- [x] 4.2 `cd backend && make generate` (commit `internal/gen/api.gen.go`)
- [x] 4.3 repo-root `make generate-ts` (commit `frontend/src/api/types.gen.ts`)

## 5. Backend auth: model/repository/service/handler
- [x] 5.1 `auth/model.go`: `PasswordResetTokenRow`
- [x] 5.2 `auth/repository.go`: `CreatePasswordResetToken`, `FindPasswordResetToken`, `ConsumePasswordResetToken`, `UpdateUserPassword`, `DeleteSessionsForUser`
      `UpdateUserPassword` additionally scopes `WHERE ... AND deleted_at IS NULL`, mirroring `UpdateUserPhoto`, so a race between account erasure and an in-flight reset can't resurrect a credential on an already-anonymized user.
- [x] 5.3 `auth/service.go`: `RegistrationConfig` gains `PasswordResetTTL`; `ErrInvalidResetToken`; `ForgotPassword` (3-case enumeration-safe logic per design.md, reusing `issueVerificationToken`'s pattern for a new `issuePasswordResetToken` helper); `ResetPassword` (consume token, validate/hash new password, `UpdateUserPassword`, `DeleteSessionsForUser`, reuse `createSessionAndSign`)
- [x] 5.4 `auth/handler.go`: `ForgotPassword`/`ResetPassword` handlers, audit events, `metrics.PasswordResetAttempts`
- [x] 5.5 `auth/cookie.go`: `applyCookie` gains `case "ResetPassword"`
- [x] 5.6 `internal/audit/audit.go`: `EventPasswordResetRequest`, `EventPasswordReset`
- [x] 5.7 `internal/metrics/business.go`: `PasswordResetAttempts` counter
- [x] 5.8 `cmd/server/main.go`: `PerIPRateLimit` wiring for `/auth/forgot-password`; unauthenticated `/auth/reset-password` route registered after the generated mux
      Also added `internal/server/server.go` delegations (`ForgotPassword`/`ResetPassword`) so `*server.Server` satisfies `gen.StrictServerInterface` -- not anticipated in design.md but required for the aggregator pattern this repo already uses for every other auth operation.

## 6. Retention job
- [x] 6.1 New phase in `RetentionWorker.Work` deleting expired `password_reset_tokens` (mirrors the existing expired-`email_verification_tokens` phase)
- [x] 6.2 `Timeout()` multiplier bumped for the new phase
      6× → 7× `retentionPhaseTimeout` (seven phases total: notifications, sessions, invites, audit_log, never-verified users, expired email-verification tokens, expired password-reset tokens).

## 7. Frontend
- [x] 7.1 `context/urlState.ts`: `parseResetPasswordToken`
- [x] 7.2 `context/AppContext.tsx`: `doForgotPassword`, `doResetPassword`; bootstrap-effect reset-password branch (shows the reset form pre-populated with the token, does NOT auto-submit); `AppState.resetPasswordToken`
- [x] 7.3 `features/auth/components/ForgotPassword.tsx` (mirrors `Register`'s check-your-email confirmation UX) + `ResetPassword.tsx` (new password + confirm, submits via `doResetPassword`) + `Login.tsx` "Forgot password?" link + `features/auth/index.ts` exports
- [x] 7.4 `services/serviceLayerReal.ts`: `auth.forgotPassword`/`auth.resetPassword`
- [x] 7.5 `mocks/handlers.ts` + `mocks/db.ts`: enumeration-safe MSW handlers for both endpoints, `passwordResetTokens` store
- [x] 7.6 `i18n/en.ts` + `i18n/de.ts`: forgot/reset-password copy

## 8. Tests
- [x] 8.1 Backend: `Repository` new methods; `Service.ForgotPassword` all 3 enumeration cases (no account / no password / has password); `Service.ResetPassword` valid/expired/consumed/wrong token, session invalidation, weak-password rejection; `Handler` endpoints incl. audit + metrics; `RetentionWorker` new phase + `Timeout()`; `config` new env var parsing
      Repository and RetentionWorker integration tests are written (`testutil.NewTestDB`) but auto-skip in this sandbox (no Docker); CI (with Docker + Postgres) should confirm they pass.
- [x] 8.2 Frontend: `ForgotPassword.tsx`/`ResetPassword.tsx`/`Login.tsx` link; `serviceContract.test.ts` new scenarios

## 9. Docs
- [x] 9.1 `CLAUDE.md` env var table: `PASSWORD_RESET_TTL_HOURS`, `FORGOT_PASSWORD_RATE_LIMIT_PER_MIN`

## 10. Verification
- [x] 10.1 `openspec validate password-reset --strict`
      Passes: "Change 'password-reset' is valid".
- [x] 10.2 `cd backend && make generate` / repo-root `make generate-ts` — no diff
      Both ran clean as part of implementation; regenerated `internal/gen/api.gen.go`, `frontend/src/api/types.gen.ts`, `frontend/src/api/zod.gen.ts`, and `internal/db/gen/models.go` (sqlc) are committed alongside the source changes that require them.
- [x] 10.3 `cd backend && make lint`
      `golangci-lint run ./...` → "0 issues."
- [x] 10.4 `cd backend && make test`
      `go build ./...`, `go vet ./...`, and `go test ./... -short` all pass. Integration tests (`testutil.NewTestDB`) auto-skip without Docker; not run here (same limitation as 1.2).
- [x] 10.5 `cd frontend && npm run lint && npm run typecheck && npm test && npm run build`
      ESLint (`--max-warnings 0`) clean, `tsc -b --noEmit` clean, 1326/1326 tests passed across 94 files, production build succeeded, `npm run check:bundle` within budget (272.7 KB / 600 KB total gzipped).
