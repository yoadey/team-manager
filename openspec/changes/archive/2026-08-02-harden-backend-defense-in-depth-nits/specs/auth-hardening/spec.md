## ADDED Requirements

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
