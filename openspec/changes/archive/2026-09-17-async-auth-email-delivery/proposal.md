## Why

`POST /auth/forgot-password` promises identical behavior for "no account",
"account with no password" (OIDC-only), and "account with a password" (see
`openspec/specs/password-reset/spec.md`'s "Forgot-password does not leak
account existence or auth method"). Before this change, the third branch
(`auth.Service.issuePasswordResetToken`) called `mailer.Mailer.
SendPasswordResetEmail` synchronously and waited for it to return — a full
SMTP round trip (dial, STARTTLS, auth, transmit; `internal/mailer/smtp.go`
dials with a 10s timeout) — before the handler responded, while the first
two branches returned as soon as `FindUserByEmail` came back empty or found
a passwordless account. The three branches' response latency was therefore
never actually identical: an attacker who timed `POST /auth/forgot-password`
responses could distinguish "has a password" (slow, real SMTP call) from
the other two (fast, DB-only), defeating the endpoint's stated
enumeration-safety contract.

`auth.Service.issueVerificationToken` (backing `POST /auth/register` and
`POST /auth/resend-verification`) had the same synchronous-SMTP-call shape,
though `openspec/specs/user-registration/spec.md`'s enumeration-safety
requirement only covers account existence, not response timing — it is
included here because it shares the same code path and the same fix.

## What Changes

- `auth.Repository.CreateEmailVerificationToken`/`CreatePasswordResetToken`
  now enqueue a durable River job (`send_verification_email`/
  `send_password_reset_email`) in the same DB transaction as the token-row
  insert, via a new `jobs.Client.InsertTx` (generic over any
  `river.JobArgs`, unlike the existing `EnqueueNotification`). The insert
  and the enqueue commit or roll back together.
- `auth.Service` no longer holds a `mailer.Mailer` at all —
  `RegistrationConfig.Mailer` is removed. `issueVerificationToken`/
  `issuePasswordResetToken` build the verification/reset link and hand it to
  the repository; they never touch SMTP or block on it.
- New `jobs.SendVerificationEmailWorker`/`SendPasswordResetEmailWorker`
  (`backend/internal/jobs/mail_worker.go`) consume those jobs and hold the
  actual `mailer.Mailer`, registered in `jobs.NewClient` (which now takes a
  `mailer.Mailer` parameter) and wired up from `cmd/server/main.go`.
  A send failure is returned as an error so River's own retry/backoff
  applies, instead of the old behavior of logging and swallowing it.
- Net effect: every `ForgotPassword` branch now does the same DB-only work
  (one `FindUserByEmail`, and for the real-account branch, one token
  insert + one job enqueue in a single transaction) with nothing resembling
  a network call, closing the timing side channel. The same applies to
  `Register`/`ResendVerification`.

## Capabilities

### Modified Capabilities
- `password-reset`: forgot-password's three branches now have equivalent
  response latency by construction (email delivery is asynchronous), not
  merely equivalent response bodies.
- `user-registration`: register/resend-verification's email delivery is
  likewise asynchronous and durable (retried on transient SMTP failure
  instead of silently dropped).

## Impact

- `backend/internal/auth/repository.go` (token-insert methods gain an
  `email`/URL parameter and enqueue a job transactionally;
  `NewRepository` takes a `jobEnqueuer`)
- `backend/internal/auth/service.go` (`RegistrationConfig.Mailer` removed;
  `issueVerificationToken`/`issuePasswordResetToken` no longer call a
  mailer)
- `backend/internal/jobs/mail_worker.go` (new:
  `SendVerificationEmailWorker`, `SendPasswordResetEmailWorker`)
- `backend/internal/jobs/notification_worker.go` (`Client.InsertTx`;
  `NewClient` takes a `mailer.Mailer` and registers the two new workers)
- `backend/cmd/server/main.go` (`initAuthComponents` takes `*jobs.Client`
  instead of `mailer.Mailer`; `mailSender` is now built before
  `jobs.NewClient` instead of before `initAuthComponents`)
- Test coverage: `backend/internal/auth/service_test.go`,
  `register_test.go`, `repository_test.go` (new DB-backed tests asserting a
  `river_job` row is enqueued atomically with the token row)
- No API shape change (`POST /auth/forgot-password` /
  `POST /auth/register` / `POST /auth/resend-verification` responses are
  unchanged); no new migration (River's own `river_job` table already
  exists via `jobs.MigrateRiver`, used by the existing notification/push
  jobs).
