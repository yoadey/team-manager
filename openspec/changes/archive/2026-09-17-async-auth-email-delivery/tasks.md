## 1. Backend: durable job enqueue

- [x] 1.1 `jobs.Client.InsertTx(ctx, tx pgx.Tx, args river.JobArgs, opts
      *river.InsertOpts) error` — inserts a job on the caller's own
      transaction via River's `InsertTx`
- [x] 1.2 `auth.SendVerificationEmailArgs`/`SendPasswordResetEmailArgs`
      (`Kind()` implementations) added to `internal/auth/repository.go`
- [x] 1.3 `auth.Repository` gains a `jobEnqueuer` field/constructor
      parameter (`NewRepository(pool, jobs jobEnqueuer)`), structurally
      satisfied by `*jobs.Client`
- [x] 1.4 `Repository.CreateEmailVerificationToken`/
      `CreatePasswordResetToken` take an `email`/URL parameter and run the
      token INSERT + `InsertTx` job enqueue on one `pgx.Tx`
      (`insertTokenAndEnqueueMailJob`)

## 2. Backend: mail-delivery workers

- [x] 2.1 `backend/internal/jobs/mail_worker.go`: `SendVerificationEmailWorker`
      / `SendPasswordResetEmailWorker`, each holding a `mailer.Mailer` and
      implementing `Work(ctx, job) error` with a bounded timeout
- [x] 2.2 `jobs.NewClient` takes a `mailSender mailer.Mailer` parameter and
      registers both new workers via `river.AddWorker`

## 3. Backend: wire-up

- [x] 3.1 `auth.Service`/`RegistrationConfig` lose the `Mailer` field;
      `issueVerificationToken`/`issuePasswordResetToken` build the link and
      pass it to the repository instead of calling a mailer
- [x] 3.2 `cmd/server/main.go`: `mailSender` construction moved ahead of
      `jobs.NewClient`; `initAuthComponents` takes `*jobs.Client` instead of
      `mailer.Mailer`

## 4. Tests

- [x] 4.1 `service_test.go`'s `mockRepo`/`newTestService`: drop the
      `Mailer` field, update `CreateEmailVerificationToken`/
      `CreatePasswordResetToken` mock signatures
- [x] 4.2 `register_test.go`'s fake repo: record enqueued mail jobs
      (recipient + link) instead of using `mailer.FakeMailer`; all
      `fm.SentCount()`/`fm.LastSentTo()` assertions moved onto the repo
- [x] 4.3 `repository_test.go`: new `newTestRepo` helper backed by a real
      `jobs.Client`/`mailer.FakeMailer`; new
      `TestRepository_CreateEmailVerificationToken_EnqueuesMailJobAtomically`
      / `TestRepository_CreatePasswordResetToken_EnqueuesMailJobAtomically`
      asserting a matching `river_job` row exists with `state = 'available'`
      right after the token insert commits

## 5. Verification

- [x] 5.1 `cd backend && go build ./...`
- [x] 5.2 `cd backend && go test ./internal/auth/... ./internal/jobs/...`
      (the new `river_job`-row integration tests skip without Docker; ran
      in this environment as skips, not passes — see 5.4)
- [x] 5.3 `cd backend && golangci-lint run ./internal/auth/... ./internal/jobs/...`
- [ ] 5.4 `cd backend && make test-integration` (requires Docker; exercises
      the new `river_job`-row assertions in `repository_test.go` end to
      end — not run in this environment, no Docker available)
