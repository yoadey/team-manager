## Context

`auth.Service.issueVerificationToken`/`issuePasswordResetToken` used to
persist a token row and then call `mailer.Mailer.SendVerificationEmail`/
`SendPasswordResetEmail` inline, awaiting the SMTP round trip before
returning. `ForgotPassword`'s enumeration-safety contract (see
`openspec/specs/password-reset/spec.md`) requires all three branches — no
account, account with no password, account with a password — to be
indistinguishable to the caller. Identical response *bodies* were already
true; identical response *timing* was not, since only the third branch
waited on SMTP.

The codebase already has a durable job queue (River, via
`backend/internal/jobs`) used for notification fan-out and Web Push
delivery, including retry/backoff on failure and a `PushDeliveryWorker`
pattern of "build the row, hand it to a worker."

## Goals

- Make `ForgotPassword`'s three branches do equivalent work end-to-end, not
  just return equivalent JSON — no branch may perform a network call (SMTP)
  before responding.
- Never lose a verification/reset email to a transient SMTP failure: a
  worker error should retry via River's backoff, not silently log-and-drop
  as the old inline call did.
- Keep the token-row insert and the "we will send this email" commitment
  atomic: a crash between the two, or a rollback of one without the other,
  must not be possible.
- Minimize blast radius: no change to the OpenAPI contract, no new
  migration, no behavior change visible to `Register`/`ForgotPassword`
  callers beyond timing.

## Decisions

- **Reuse River, not a new queue.** `internal/jobs` already runs a
  `river.Client[pgx.Tx]` against the same Postgres pool, with an existing
  `river_job` table (`jobs.MigrateRiver`) and an established
  worker-registration pattern (`river.AddWorker` in `jobs.NewClient`).
  Adding two more `river.Worker` implementations is a much smaller change
  than standing up a second delivery mechanism.
- **Atomicity via `river.Client.InsertTx`, not two separate writes.** River
  supports inserting a job as part of a caller-supplied `pgx.Tx`; the job
  row only becomes visible (and thus workable) once that transaction
  commits, and disappears with it on rollback. A new `jobs.Client.InsertTx`
  wraps this (generic over `river.JobArgs`, `EnqueueNotification` stays
  hardcoded to `NotificationArgs` and non-transactional since it has no
  atomicity requirement of its own). `auth.Repository`'s
  `insertTokenAndEnqueueMailJob` runs the token INSERT and `InsertTx` on
  the same `tx`, sharing one `Begin`/`Commit`.
  - Alternative considered: enqueue after commit, outside the transaction.
    Rejected — a crash or failure between "commit the token" and "enqueue
    the job" would durably persist a token with no corresponding email ever
    sent, and the user has no way to notice or retry (only
    `POST /auth/resend-verification` exists for the verification case; there
    is no equivalent for a stuck reset token besides waiting for TTL expiry
    and asking again).
- **`auth.Repository` depends on a narrow `jobEnqueuer` interface, not
  `internal/jobs` directly.** `internal/jobs` already imports
  `internal/notifications`, which imports `internal/auth` (for
  `HashEmailForAudit`/session-adjacent helpers used by audit logging) —
  `internal/auth` importing `internal/jobs` back would cycle. `jobEnqueuer`
  (`InsertTx(ctx, tx, river.JobArgs, *river.InsertOpts) error`) is
  structurally satisfied by `*jobs.Client` without either package needing
  to import the other's full surface; `river.JobArgs`/`river.InsertOpts`
  are plain types from the external `river` module, safe for `internal/auth`
  to name directly.
- **The job-args types (`SendVerificationEmailArgs`, `
  SendPasswordResetEmailArgs`) live in `internal/auth`, not
  `internal/jobs`.** `internal/auth` is their sole producer; putting them
  in `internal/jobs` would need `internal/auth` to import `internal/jobs`
  for `Kind()` compile-time checks anyway, hitting the same cycle above.
  `jobs.SendVerificationEmailWorker`/`SendPasswordResetEmailWorker` import
  `internal/auth` for the args types (the same direction `internal/jobs`
  already imports `internal/auth` transitively via `internal/notifications`
  — no new edge).
- **`RegistrationConfig.Mailer` is removed rather than left unused.** A
  `mailer.Mailer` field nobody reads would silently invite a future
  regression (someone "fixing" a bug by calling it inline again, right back
  where this change started). The real `mailer.Mailer` now lives solely on
  `jobs.NewClient`'s new parameter, consumed only by the two new workers.
- **`jobs.NewClient` takes `mailSender mailer.Mailer` as a required
  parameter**, not optional/nilable like `pushDeps`/`eventReminderWorker`.
  Unlike push (which has a real "team has push disabled entirely" off
  state), auth's mail-sending is not optional in any deployment — every
  environment configures a mailer (real SMTP in prod, `mailer.
  NewFakeMailer` in dev/tests per `cmd/server/main.go`'s `initMailer`), so
  there is no meaningful "nil mailer" case to support here.
- **Where mailSender is constructed in `main.go` moved earlier** (before
  `jobs.NewClient`, instead of just before `initAuthComponents`), since
  `jobs.NewClient` now needs it. `initAuthComponents` takes `*jobs.Client`
  instead of `mailer.Mailer`.

## Risks

- **A stuck/failed `river_job` row for mail delivery is now the only
  record of "we owe this user an email".** Mitigated the same way push
  delivery already is: River's built-in retry/backoff on worker error, plus
  the job row is inspectable via `river_job` for ops if delivery keeps
  failing (e.g. persistent SMTP misconfiguration) — this is strictly better
  observability than the old behavior (a `WarnContext` log line and
  nothing else).
- **Verification/reset emails are no longer guaranteed sent by the time the
  HTTP response returns**, which is now true by design (that's the fix),
  but is a behavior change worth calling out: a user who submits
  `POST /auth/register` and immediately checks their inbox may see a few
  seconds' delay before River's worker picks up the job, versus the old
  synchronous send. This is expected and acceptable — the endpoint's
  contract was never "email delivered by the time this call returns," only
  "an email will be sent."
- **`jobs.Client.InsertTx` bypasses `EnqueueNotification`'s existing
  validation/logging wrapper.** Accepted: `InsertTx` is a lower-level
  primitive by design (generic over `river.JobArgs`), and its two current
  callers (`auth.Repository`'s two token-insert methods) are the only
  producers of `SendVerificationEmailArgs`/`SendPasswordResetEmailArgs`, so
  there is no cross-package misuse surface today.
