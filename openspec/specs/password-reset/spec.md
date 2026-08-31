# password-reset Specification

## Purpose
Defines self-service password recovery by email: an unauthenticated visitor can request a reset link via `POST /auth/forgot-password`, which always returns the same generic response regardless of whether the email has no account, an account with no password, or an account with a password, so account existence and auth method are never leaked; a resulting reset token is single-use and expires, `POST /auth/reset-password` accepts it only with a password meeting the strength policy and then returns a session equivalent to login, a successful reset invalidates every other existing session on the account, the endpoint is per-IP rate-limited like the codebase's other public auth endpoints, and expired tokens are eventually cleaned up by the daily retention job.

## Requirements

### Requirement: A user can request a password reset link by email
The system MUST allow an unauthenticated visitor to request a password
reset link by submitting an email address to `POST /auth/forgot-password`.

#### Scenario: Account with a password receives a reset link
- **WHEN** a visitor submits an email address that has an account with a
  password set
- **THEN** a single-use, time-limited reset token is issued and a
  password-reset email is sent to that address

### Requirement: Forgot-password does not leak account existence or auth method
`POST /auth/forgot-password` MUST return the same response regardless of
whether the submitted email has no account, has an account with no password
set, or has an account with a password.

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
- **THEN** a reset token is issued and emailed, and the same generic success
  response is returned

### Requirement: Reset tokens are single-use and expire
A password reset token MUST be rejected once it has already been consumed or
once its expiry has passed.

#### Scenario: Reused token
- **WHEN** a reset-password request is submitted with a token that was
  already successfully consumed
- **THEN** the request is rejected and no password change occurs

#### Scenario: Expired token
- **WHEN** a reset-password request is submitted with a token past its
  expiry
- **THEN** the request is rejected and no password change occurs

### Requirement: A valid reset token changes the password and establishes a session
`POST /auth/reset-password` MUST, given a valid unexpired unused token and a
password meeting the strength policy, set the account's new password and
return a session equivalent to a successful login.

#### Scenario: Valid token and valid password
- **WHEN** a reset-password request is submitted with a valid, unexpired,
  unused token and a password meeting the strength policy
- **THEN** the account's password is updated and the response contains a
  session token and user profile, identical in shape to `login`'s response

#### Scenario: Weak new password is rejected
- **WHEN** a reset-password request is submitted with a valid token but a
  password outside the accepted strength policy
- **THEN** the request is rejected and the account's password is unchanged

### Requirement: Resetting a password invalidates every existing session
Successfully resetting a password MUST invalidate all of that account's
existing sessions, not only the session created by the reset itself.

#### Scenario: Other active sessions are logged out
- **WHEN** an account with one or more active sessions completes a password
  reset
- **THEN** every session that existed before the reset is invalidated, and
  only the new session created by the reset remains valid

### Requirement: Forgot-password requests are rate-limited
`POST /auth/forgot-password` MUST be rate-limited per client IP, mirroring
this codebase's other public auth endpoints.

#### Scenario: Rate limit exceeded
- **WHEN** a single client IP exceeds `FORGOT_PASSWORD_RATE_LIMIT_PER_MIN`
  requests to `POST /auth/forgot-password` within a minute
- **THEN** further requests from that IP are rejected until the window
  resets

### Requirement: Expired reset tokens are eventually cleaned up
The daily retention job MUST delete password reset token rows once their
expiry has passed.

#### Scenario: Cleanup of expired tokens
- **WHEN** the retention job runs and finds password reset tokens past
  their expiry
- **THEN** those token rows are deleted

#### Scenario: Unexpired tokens are untouched
- **WHEN** the retention job runs
- **THEN** password reset tokens that have not yet expired are left
  untouched
