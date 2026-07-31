## ADDED Requirements

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
