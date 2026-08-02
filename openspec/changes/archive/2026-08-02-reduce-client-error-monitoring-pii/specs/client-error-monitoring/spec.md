## ADDED Requirements

### Requirement: Sentry events carry no personally identifying user data
Error events sent to Sentry MUST NOT include the user's email address, IP
address, or display name.

#### Scenario: Authenticated user triggers an error report
- **WHEN** an authenticated user's action produces an error event sent to
  Sentry
- **THEN** the event's user context includes only an opaque user id
- **AND** no email, IP address, or display name is present anywhere in the
  event
