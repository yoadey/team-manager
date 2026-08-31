# client-error-monitoring Specification

## Purpose
Defines how client-side error events are scrubbed before being sent to Sentry: the user context carries only an opaque user id, with email address, IP address, and display name never included anywhere in the event.
## Requirements
### Requirement: Sentry events carry no personally identifying user data
Error events sent to Sentry MUST NOT include the user's email address, IP
address, or display name.

#### Scenario: Authenticated user triggers an error report
- **WHEN** an authenticated user's action produces an error event sent to
  Sentry
- **THEN** the event's user context includes only an opaque user id
- **AND** no email, IP address, or display name is present anywhere in the
  event

