## ADDED Requirements

### Requirement: No plaintext email in the audit log
Audit log entries MUST NOT store a member's email address in plaintext. A keyed hash MAY be stored to allow correlation.

#### Scenario: Login is audited
- **WHEN** a login attempt (success or failure) is recorded in the audit log
- **THEN** the stored attributes contain a keyed hash of the email, not the plaintext address

### Requirement: Over-length passwords are rejected
The system MUST reject passwords longer than 72 bytes with a validation error rather than silently truncating them.

#### Scenario: Password exceeds bcrypt limit
- **WHEN** a password longer than 72 bytes is submitted for login or password set
- **THEN** the request is rejected with a validation error before hashing

### Requirement: CSRF fallback for header-less mutations
For a cookie-authenticated state-changing request, if neither an `Origin` nor a `Sec-Fetch-Site` header is present, the request MUST be rejected.

#### Scenario: Mutating request with no fetch metadata
- **WHEN** a POST/PUT/PATCH/DELETE arrives with no `Origin` and no `Sec-Fetch-Site`
- **THEN** it is rejected
- **AND** a request with a whitelisted `Origin` is still allowed
