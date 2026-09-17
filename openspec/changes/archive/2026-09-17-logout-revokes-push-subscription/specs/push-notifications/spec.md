## MODIFIED Requirements

### Requirement: A user can disable Web Push notifications
The system MUST let an authenticated user remove a previously registered
subscription so no further pushes are sent to it. Logging out MUST also
revoke the browser's current Web Push subscription, so a shared/kiosk
device stops receiving the logged-out account's push notifications
without requiring the user to remember to disable push separately.

#### Scenario: Unregistering a subscription
- **WHEN** a user disables push (or the browser unsubscribes locally) and the
  frontend calls `DELETE /users/me/push-subscriptions` with that endpoint
- **THEN** the subscription row is deleted and no further pushes are sent to
  it

#### Scenario: A user cannot remove another user's subscription
- **WHEN** a delete request names an endpoint that belongs to a different
  user's subscription
- **THEN** the request has no effect on that other user's subscription

#### Scenario: Logging out with an active push subscription
- **WHEN** a user with an active Web Push subscription on this browser logs
  out
- **THEN** the browser's `PushSubscription` is unsubscribed locally and
  `DELETE /users/me/push-subscriptions` is called for it, so no further
  pushes are delivered to this browser for that account

#### Scenario: Logout with no active subscription, or an unsupported browser
- **WHEN** a user logs out and either the browser has no active
  `PushSubscription` or the browser doesn't support Web Push
- **THEN** logout proceeds normally with no push-related network call

#### Scenario: Push unsubscribe fails during logout
- **WHEN** the local unsubscribe or the backend delete call fails (e.g. the
  device is offline) during logout
- **THEN** logout still completes and the user is returned to the login
  screen — the failure is reported for diagnostics but never blocks or
  fails the logout flow
