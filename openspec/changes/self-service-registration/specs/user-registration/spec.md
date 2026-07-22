## ADDED Requirements

### Requirement: A new user can self-register with email + password
The system MUST allow an unauthenticated visitor to create a new account by
submitting an email address and a password, when self-registration is
enabled.

#### Scenario: Valid registration creates an unverified account
- **WHEN** a visitor submits a valid, unused email address and a password
  meeting the strength policy to `POST /auth/register`
- **THEN** a new account is created with the password hashed, marked
  unverified, and a verification email is sent to that address

#### Scenario: Invalid input is rejected
- **WHEN** a registration is submitted with a malformed email or a password
  outside the accepted length window
- **THEN** the request is rejected with a validation error and no account is
  created

### Requirement: Registration does not leak account existence
`POST /auth/register` MUST return the same response regardless of whether
the submitted email is available, already registered and verified, or
already registered and still pending verification.

#### Scenario: Email is available
- **WHEN** registration is submitted for an email with no existing account
- **THEN** a new unverified account is created and the generic success
  response is returned

#### Scenario: Email already has a verified account
- **WHEN** registration is submitted for an email that already has a
  verified account
- **THEN** the existing account and its password are left unchanged, and the
  same generic success response is returned

#### Scenario: Email already has a pending (unverified) registration
- **WHEN** registration is submitted for an email that already has an
  unverified account
- **THEN** the existing password hash is not overwritten, a fresh
  verification token is issued to that address, and the same generic success
  response is returned

### Requirement: An unverified account cannot log in
`Login` MUST reject a correct email/password pair when the account has never
completed email verification, with a response distinguishable from wrong
credentials.

#### Scenario: Correct credentials, unverified account
- **WHEN** a login is attempted with the correct email and password for an
  account whose email has never been verified
- **THEN** the login is rejected with a response indicating the account is
  unverified, not a generic invalid-credentials error

#### Scenario: Correct credentials, verified account
- **WHEN** a login is attempted with the correct email and password for a
  verified account
- **THEN** the login succeeds as before (no regression)

### Requirement: Email verification establishes a session
`POST /auth/verify-email` MUST, given a valid unexpired unused token, mark
the account verified and return a session equivalent to a successful login.

#### Scenario: Valid token
- **WHEN** a verification request is submitted with a valid, unexpired,
  unused token
- **THEN** the account is marked verified and the response contains a
  session token and user profile, identical in shape to `login`'s response

### Requirement: Verification tokens are single-use and expire
A verification token MUST be rejected once it has already been consumed or
once its expiry has passed.

#### Scenario: Reused token
- **WHEN** a verification request is submitted with a token that was already
  successfully consumed
- **THEN** the request is rejected and no further state change occurs

#### Scenario: Expired token
- **WHEN** a verification request is submitted with a token past its expiry
- **THEN** the request is rejected

### Requirement: Resend verification never confirms account existence
`POST /auth/resend-verification` MUST return the same generic response
regardless of whether the submitted email has no account, an already
verified account, or a still-unverified account — only the last case
actually triggers a new email.

#### Scenario: Uniform response across all account states
- **WHEN** resend-verification is requested for an email with no account, an
  already-verified account, and a still-unverified account, in turn
- **THEN** all three requests receive the same response body, and only the
  still-unverified case results in an email being sent

### Requirement: Self-registration can be disabled server-side
When `SELF_REGISTRATION_ENABLED` is set to false, `POST /auth/register` MUST
reject every request, even otherwise-valid ones, without creating an
account.

#### Scenario: Feature disabled
- **WHEN** self-registration is disabled and a visitor submits an otherwise
  valid registration
- **THEN** the request is rejected and no account is created

### Requirement: Never-verified accounts are eventually cleaned up
The daily retention job MUST delete accounts whose email was never verified
once they are older than the configured retention window, so the email
address becomes available for a fresh registration.

#### Scenario: Cleanup after the retention window
- **WHEN** the retention job runs and finds an unverified account older than
  `RETENTION_UNVERIFIED_ACCOUNTS_DAYS`
- **THEN** the account and its verification tokens are deleted

#### Scenario: Recent unverified and verified accounts are untouched
- **WHEN** the retention job runs
- **THEN** unverified accounts still within the retention window, and all
  verified accounts, are left untouched
