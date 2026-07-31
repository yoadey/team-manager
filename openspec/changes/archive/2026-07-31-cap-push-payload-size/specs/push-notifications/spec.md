## MODIFIED Requirements

### Requirement: Delivery failures do not block the notification pipeline
A push delivery failure MUST NOT prevent the underlying notification from
being recorded, and MUST NOT crash or stall the worker processing other
notifications. The push notification body MUST be bounded to a fixed
length before it is enqueued, so no source field's length can produce an
encrypted payload the push service rejects as too large.

#### Scenario: Push service is temporarily unavailable
- **WHEN** the browser's push service returns a transient error (e.g. a 5xx
  or network failure)
- **THEN** the notification row itself is unaffected, and the push delivery
  job is retried through the existing job-queue retry mechanism

#### Scenario: Notification source text exceeds the push body length cap
- **WHEN** a notification's title/note/event-title text (e.g. a poll
  question with no length limit) exceeds the push body's length cap
- **THEN** the enqueued push body is truncated to the cap with a trailing
  ellipsis, rather than sent in full

#### Scenario: Push service reports the payload is too large
- **WHEN** a delivery attempt receives a 413 response from the push
  service
- **THEN** the push delivery job is cancelled (not retried) and the
  corresponding `push_subscriptions` row is left in place, since the
  subscription itself is still valid — only that message was too large
