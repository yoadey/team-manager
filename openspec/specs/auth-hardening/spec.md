# auth-hardening Specification

## Purpose
Defines a set of targeted hardening measures for authentication and its surrounding infrastructure: audit log entries store a one-way hash of a member's email instead of plaintext, over-length passwords are rejected before hashing or lookup rather than silently truncated, cross-site mutating requests are blocked based on `Sec-Fetch-Site` even without a disallowed `Origin`, the mailer independently rejects header-injection attempts via CRLF in `to`/`from`/`subject`, and a missing auth context on an authenticated route reports 401 rather than 404.
## Requirements
### Requirement: No plaintext email in the audit log
Audit log entries MUST NOT store a member's email address in plaintext. A one-way hash MAY be stored so repeated attempts for the same address stay correlatable.

#### Scenario: Login is audited
- **WHEN** a login attempt (success or failure) is recorded in the audit log
- **THEN** the stored attributes contain a one-way hash of the email, not the plaintext address

### Requirement: Over-length passwords are rejected
The system MUST reject passwords longer than 72 bytes with a validation error rather than silently truncating them (bcrypt's input limit).

#### Scenario: Password exceeds bcrypt limit at hashing
- **WHEN** a password longer than 72 bytes is submitted to be hashed
- **THEN** it is rejected before hashing

#### Scenario: Password exceeds bcrypt limit at login
- **WHEN** a login is attempted with a password longer than 72 bytes
- **THEN** it is rejected as invalid credentials without a database lookup

### Requirement: Cross-site mutating requests are blocked
For a cookie-authenticated state-changing request that the browser marks as cross-site (`Sec-Fetch-Site: cross-site`), the request MUST be rejected, even when a disallowed `Origin` header is absent. Requests without cross-site metadata and without a disallowed Origin (non-browser API clients, same-origin) remain allowed.

#### Scenario: Cross-site fetch metadata without Origin
- **WHEN** a POST/PUT/PATCH/DELETE arrives with `Sec-Fetch-Site: cross-site` and no `Origin`
- **THEN** it is rejected

#### Scenario: Same-origin request
- **WHEN** a mutating request arrives with `Sec-Fetch-Site: same-origin`
- **THEN** it is allowed

### Requirement: Outbound email headers are defended at the mailer layer
The mailer MUST reject any `to`, `from`, or `subject` value containing a
carriage return or line feed before constructing the message, independent
of any validation performed by the caller.

#### Scenario: A field contains a CRLF sequence
- **WHEN** `to`, `from`, or `subject` passed to the mailer contains `\r`
  or `\n`
- **THEN** the mailer returns an error and does not send a message with
  injected headers

### Requirement: A missing auth context on an authenticated route reports 401
A handler that requires an authenticated user but finds none in the
request context MUST respond 401 Unauthorized, not 404 Not Found.

#### Scenario: Auth context missing on the own-photo endpoint
- **WHEN** `GET /me/photo` is handled with no authenticated user in the
  request context
- **THEN** the response is 401, not 404

