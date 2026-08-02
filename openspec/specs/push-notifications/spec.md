# push-notifications Specification

## Purpose
TBD - created by archiving change add-web-push-notifications. Update Purpose after archive.
## Requirements
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
notifications. The push notification body MUST be bounded to a fixed
length before it is enqueued, so no source field's length can produce an
encrypted payload the push service rejects as too large. A non-2xx
response from the push service (other than a 404/410, which prunes the
subscription) MUST carry a bounded snippet of the push service's own
response body in the error returned by `push.WebPusher.Send`, so the
job-queue error log gives an operator enough detail to diagnose the
failure (e.g. a VAPID authentication rejection) without adding separate
request logging.

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

#### Scenario: Push service rejects VAPID authentication
- **WHEN** the push service responds with a non-2xx status other than
  404/410/413 (e.g. a 401 from a VAPID key mismatch, or a 5xx that isn't a
  transient network failure)
- **THEN** the error returned by `push.WebPusher.Send` includes a bounded
  snippet of the response body alongside the status code, and the push
  delivery job is retried through the existing job-queue retry mechanism

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

### Requirement: A member can configure which notification categories are pushed, per team
The system MUST let an authenticated team member independently enable or
disable Web Push delivery for each notification category (attendance,
events, news, polls, absence) within a specific team, without affecting any
other team they belong to.

#### Scenario: Member disables push for one category in one team
- **WHEN** a member sets `news: false` for team A via
  `PUT /teams/{teamId}/push-preferences`
- **THEN** future `news` notifications in team A are not pushed to that
  member's subscriptions
- **AND** `news` notifications in any other team the member belongs to are
  still pushed, and every other category in team A is still pushed

#### Scenario: Reading current preferences
- **WHEN** a member calls `GET /teams/{teamId}/push-preferences`
- **THEN** the response reflects their last-saved preferences for that team,
  or all categories enabled if they've never changed anything

### Requirement: Preferences default to fully enabled
A team member who has never configured push preferences MUST receive push
notifications for every category their module permissions already allow —
identical to the pre-existing, non-configurable behavior.

#### Scenario: Member has no stored preferences
- **WHEN** a notification is created for a member who has never called
  `PUT /teams/{teamId}/push-preferences` for that team
- **THEN** a push is sent for it exactly as if every category were enabled,
  subject only to the existing module-permission gate

### Requirement: The preference gate is independent of the permission gate
A push MUST be sent only when the recipient both has read access to the
notification's module (or it is self-standing) AND has not disabled that
notification's category for that team; either condition failing suppresses
the push.

#### Scenario: Permission allows but preference disables
- **WHEN** a recipient has `events:read` in a team but has disabled the
  `events` category there
- **THEN** no push is sent for an `event_created` notification in that team,
  even though the in-app feed still shows it

#### Scenario: Preference allows but permission denies
- **WHEN** a recipient has enabled the `polls` category in a team but their
  current `polls` module permission is `none`
- **THEN** no push is sent for a `poll` notification in that team, matching
  the existing permission-gate behavior

### Requirement: A member can configure a reminder push before an event starts, per team
The system MUST let an authenticated team member enable or disable a push
reminder for upcoming events in a specific team, and configure how many
hours before an event's start the reminder is sent, independent of the
existing per-category push preferences and of any other team the member
belongs to. The reminder is delivered by push only and does not create an
in-app notification-feed entry.

#### Scenario: Reading the default reminder preference
- **WHEN** a member calls `GET /teams/{teamId}/push-preferences` having
  never configured anything
- **THEN** the response reports `eventReminderEnabled: true` and
  `eventReminderHoursBefore: 6`

#### Scenario: Member customizes the lead time
- **WHEN** a member sets `eventReminderHoursBefore: 24` for team A via
  `PUT /teams/{teamId}/push-preferences`
- **THEN** future reminders for events in team A are evaluated against a
  24-hour lead time for that member
- **AND** the setting has no effect on any other team the member belongs to

#### Scenario: Member disables event reminders
- **WHEN** a member sets `eventReminderEnabled: false` for a team
- **THEN** no reminder push is sent to that member for events in that team,
  even though other push categories (attendance, events, news, polls,
  absence) are unaffected

#### Scenario: Configured lead time out of range
- **WHEN** a member submits `eventReminderHoursBefore` outside 1–72
- **THEN** the request is rejected with 400 and no preference is saved

### Requirement: A reminder push is sent once, near the configured lead time before an event starts
The system MUST deliver, at most once per (event, member) pair, a push
notification once the current time has reached the member's configured
`eventReminderHoursBefore` window before the event's computed start instant,
provided the event is not cancelled and the member currently has at least
read access to the events module.

#### Scenario: Reminder becomes due
- **WHEN** the current time reaches a member's configured
  `eventReminderHoursBefore` before a non-cancelled event's start instant,
  and reminders are enabled for that member/team
- **THEN** a push notification referencing the event is sent to each of the
  member's registered subscriptions

#### Scenario: Reminder already sent
- **WHEN** a reminder has already been sent for a given (event, member) pair
- **THEN** it is never sent again for that pair, even though the periodic
  check that discovers due reminders keeps re-evaluating the same event on
  every run until it starts

#### Scenario: Event is cancelled before the reminder window
- **WHEN** an event's status is `cancelled` at the time the reminder would
  become due
- **THEN** no reminder is sent for it

#### Scenario: Member's module permission denies read access
- **WHEN** a member's current `events` module permission is `none` at the
  time the reminder would become due
- **THEN** no reminder is sent to that member for that event, matching how
  other push categories are gated by module read access

