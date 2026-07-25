## ADDED Requirements

### Requirement: Subscribe a browser for web push
A user MUST be able to register a browser push subscription so that notifications can be delivered as web-push messages.

#### Scenario: User enables push
- **WHEN** a user grants notification permission and subscribes
- **THEN** their push subscription (endpoint + keys) is stored server-side, unique per endpoint

#### Scenario: Feature not configured
- **WHEN** VAPID keys are not configured on the server
- **THEN** the subscribe endpoint is unavailable and the client shows push as unavailable rather than failing silently

### Requirement: Deliver notifications via web push
When a notification is created for a user who has an active push subscription, the server MUST attempt to deliver it as a web-push message subject to the same permission filtering as the in-app feed.

#### Scenario: Notification delivered
- **WHEN** a notification is created for a subscribed user who is permitted to see it
- **THEN** a web-push message is sent to that user's subscription and clicking it opens the relevant place in the app

#### Scenario: Dead subscription pruned
- **WHEN** the push service responds 404 or 410 for a subscription
- **THEN** that subscription is deleted and no longer receives pushes

#### Scenario: Notification the user may not see
- **WHEN** a notification originates from a module the user lacks permission on
- **THEN** no web-push message for it is sent to that user
