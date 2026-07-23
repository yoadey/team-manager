## MODIFIED Requirements

### Requirement: Onboarding docs describe the actual login/registration methods
`docs/end-user/erste-schritte.md` MUST describe the login and account-creation
methods the shipped product actually offers, and MUST NOT describe a login
method that does not exist in the code.

#### Scenario: New member follows the getting-started guide
- **WHEN** a newly invited member reads `docs/end-user/erste-schritte.md`
- **THEN** they learn that login is by email + password, that a new account
  is created via self-service registration (email verification required),
  and that following the invite link joins them to the inviting team after
  registering or logging in
- **AND** the guide makes no claim that login happens via an external
  Identity Provider / SSO
