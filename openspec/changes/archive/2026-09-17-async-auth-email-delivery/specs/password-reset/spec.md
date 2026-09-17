## MODIFIED Requirements

### Requirement: Forgot-password does not leak account existence or auth method
`POST /auth/forgot-password` MUST return the same response regardless of
whether the submitted email has no account, has an account with no password
set, or has an account with a password. This equivalence MUST hold for
response latency, not only for the response body: no branch may perform a
network call (e.g. sending the reset email over SMTP) before responding, so
that response timing cannot be used to distinguish which branch was taken.

#### Scenario: No account for the email
- **WHEN** forgot-password is requested for an email with no existing
  account
- **THEN** no token is issued, no email is sent, and the generic success
  response is returned

#### Scenario: Account exists but has no password set
- **WHEN** forgot-password is requested for an email whose account has no
  password set
- **THEN** no token is issued, no email is sent, and the same generic
  success response is returned

#### Scenario: Account exists with a password
- **WHEN** forgot-password is requested for an email whose account has a
  password set
- **THEN** a reset token is issued and its email durably enqueued for
  asynchronous delivery, and the same generic success response is returned

#### Scenario: Response timing does not distinguish the branches
- **WHEN** forgot-password is requested for an email in any of the three
  states above
- **THEN** the handler returns as soon as its DB work completes, without
  waiting on any email delivery — sending the reset email happens
  out-of-band on a durable job queue, after the response has already gone
  out

#### Scenario: Reset-email delivery fails transiently
- **WHEN** the reset-email job's send attempt fails transiently (e.g. an
  SMTP relay hiccup)
- **THEN** the job is retried with backoff rather than the email being
  silently dropped
