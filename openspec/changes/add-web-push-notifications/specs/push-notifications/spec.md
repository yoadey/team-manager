## ADDED Requirements

### Requirement: A user can enable Web Push notifications for their browser
The system MUST let an authenticated user register their browser's push
subscription so they receive push notifications independent of whether the
app is open.

#### Scenario: Registering a subscription
- **WHEN** a user grants notification permission and the frontend POSTs the
  resulting `PushSubscription` (endpoint + encryption keys) to
  `POST /users/me/push-subscriptions`
- **THEN** the subscription is stored against that user's account and future
  qualifying notifications are pushed to it

#### Scenario: Re-registering the same browser
- **WHEN** a subscription is registered whose endpoint already exists for
  that user (e.g. the browser re-subscribed after a key rotation)
- **THEN** the existing row is updated in place rather than duplicated

### Requirement: A user can disable Web Push notifications
The system MUST let an authenticated user remove a previously registered
subscription so no further pushes are sent to it.

#### Scenario: Unregistering a subscription
- **WHEN** a user disables push (or the browser unsubscribes locally) and the
  frontend calls `DELETE /users/me/push-subscriptions` with that endpoint
- **THEN** the subscription row is deleted and no further pushes are sent to
  it

#### Scenario: A user cannot remove another user's subscription
- **WHEN** a delete request names an endpoint that belongs to a different
  user's subscription
- **THEN** the request has no effect on that other user's subscription

### Requirement: Push delivery respects the recipient's current module permissions
A push notification MUST NOT be sent for a notification whose originating
module the recipient does not currently have at least "read" on — the same
gate `notifications.Service.List` applies to the in-app feed.

#### Scenario: Recipient has read access to the module
- **WHEN** a notification is created for a module the recipient has "read" or
  "write" on
- **THEN** a push is sent to each of the recipient's registered subscriptions

#### Scenario: Recipient's module permission is "none"
- **WHEN** a notification is created for a module the recipient's current
  permission is "none" on
- **THEN** no push is sent to the recipient for that notification

#### Scenario: Self-standing notification types are always pushed
- **WHEN** a notification of a type not gated by any module (e.g. an absence
  notice) is created for a team member
- **THEN** a push is sent regardless of module permissions, matching how the
  in-app feed always shows it

### Requirement: Delivery failures do not block the notification pipeline
A push delivery failure MUST NOT prevent the underlying notification from
being recorded, and MUST NOT crash or stall the worker processing other
notifications.

#### Scenario: Push service is temporarily unavailable
- **WHEN** the browser's push service returns a transient error (e.g. a 5xx
  or network failure)
- **THEN** the notification row itself is unaffected, and the push delivery
  job is retried through the existing job-queue retry mechanism

### Requirement: Permanently invalid subscriptions are pruned automatically
The system MUST delete a subscription once the push service reports it can
never be delivered to again, so failed sends don't accumulate indefinitely.

#### Scenario: Push service reports the subscription is gone
- **WHEN** a delivery attempt receives a 404 or 410 response from the push
  service
- **THEN** the corresponding `push_subscriptions` row is deleted and no
  further deliveries are attempted to it

### Requirement: Push delivery is disableable per environment
Web Push MUST degrade gracefully to a no-op in environments without VAPID
keys configured, and MUST be required when the deployment is otherwise
production-configured.

#### Scenario: VAPID keys not configured, cookies not secure (dev)
- **WHEN** the server starts without `VAPID_PUBLIC_KEY`/`VAPID_PRIVATE_KEY`/
  `VAPID_SUBJECT` set and `COOKIE_SECURE=false`
- **THEN** the server starts successfully and push sends are logged instead
  of actually delivered

#### Scenario: VAPID keys missing, cookies secure (production)
- **WHEN** the server starts with `COOKIE_SECURE=true` and any of
  `VAPID_PUBLIC_KEY`/`VAPID_PRIVATE_KEY`/`VAPID_SUBJECT` unset
- **THEN** startup fails with a clear configuration error, matching the
  existing `S3_*`/`SMTP_*` required-when-secure behavior
