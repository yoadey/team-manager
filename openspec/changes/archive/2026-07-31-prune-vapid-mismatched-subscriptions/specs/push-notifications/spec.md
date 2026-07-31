## MODIFIED Requirements

### Requirement: Permanently invalid subscriptions are pruned automatically
The system MUST delete a subscription once the push service reports it can
never be delivered to again, so failed sends don't accumulate indefinitely.

#### Scenario: Push service reports the subscription is gone
- **WHEN** a delivery attempt receives a 404 or 410 response from the push
  service
- **THEN** the corresponding `push_subscriptions` row is deleted and no
  further deliveries are attempted to it

#### Scenario: Push service reports a VAPID key mismatch for this subscription
- **WHEN** a delivery attempt receives a 401 or 403 response whose body
  identifies this specific subscription as created against a VAPID public
  key the server no longer signs with (e.g. Mozilla autopush's errno 109
  "VAPID public key mismatch", or FCM's "credentials ... do not correspond
  to the credentials used to create the subscription")
- **THEN** the corresponding `push_subscriptions` row is deleted and no
  further deliveries are attempted to it, the same as a 404/410

#### Scenario: A 401/403 does not identify a specific stale subscription
- **WHEN** a delivery attempt receives a 401 or 403 response whose body
  does not match a known key-mismatch signature (e.g. a malformed
  `VAPID_SUBJECT` rejected server-wide)
- **THEN** the subscription is left in place and the push delivery job is
  retried through the existing job-queue retry mechanism, since a
  configuration fix may recover it
