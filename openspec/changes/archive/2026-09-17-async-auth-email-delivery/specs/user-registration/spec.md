## MODIFIED Requirements

### Requirement: A new user can self-register with email + password
The system MUST allow an unauthenticated visitor to create a new account by
submitting an email address and a password, when self-registration is
enabled. Sending the verification email MUST NOT block the response: the
account is created and a verification token issued synchronously, but
delivery of the verification email happens asynchronously on a durable job
queue, retried with backoff on a transient send failure rather than
silently dropped.

#### Scenario: Valid registration creates an unverified account
- **WHEN** a visitor submits a valid, unused email address and a password
  meeting the strength policy to `POST /auth/register`
- **THEN** a new account is created with the password hashed, marked
  unverified, and a verification email is durably enqueued for delivery to
  that address

#### Scenario: Invalid input is rejected
- **WHEN** a registration is submitted with a malformed email or a password
  outside the accepted length window
- **THEN** the request is rejected with a validation error and no account is
  created

#### Scenario: Verification-email delivery fails transiently
- **WHEN** the verification-email job's send attempt fails transiently
  (e.g. an SMTP relay hiccup)
- **THEN** the job is retried with backoff rather than the email being
  silently dropped, and the account remains available for
  `POST /auth/resend-verification` regardless
