## MODIFIED Requirements

### Requirement: Delivery failures do not block the notification pipeline
A push delivery failure MUST NOT prevent the underlying notification from
being recorded, and MUST NOT crash or stall the worker processing other
notifications. A non-2xx response from the push service (other than a
404/410, which prunes the subscription) MUST carry a bounded snippet of
the push service's own response body in the error returned by
`push.WebPusher.Send`, so the job-queue error log gives an operator enough
detail to diagnose the failure (e.g. a VAPID authentication rejection)
without adding separate request logging.

#### Scenario: Push service is temporarily unavailable
- **WHEN** the browser's push service returns a transient error (e.g. a 5xx
  or network failure)
- **THEN** the notification row itself is unaffected, and the push delivery
  job is retried through the existing job-queue retry mechanism

#### Scenario: Push service rejects VAPID authentication
- **WHEN** the push service responds with a non-2xx status other than
  404/410 (e.g. a 401 from a VAPID key mismatch)
- **THEN** the error returned by `push.WebPusher.Send` includes a bounded
  snippet of the response body alongside the status code, and the push
  delivery job is retried through the existing job-queue retry mechanism
