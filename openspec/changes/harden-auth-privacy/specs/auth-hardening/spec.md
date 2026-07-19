## ADDED Requirements

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
