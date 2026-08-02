## Context

Two existing mechanisms this change builds on:

- `push.CategoryPreferences` / `push_preferences` table: per-(team, user)
  opt-out, one boolean per notification category, defaulting to fully
  enabled when no row exists (`push.DefaultCategoryPreferences`).
- `jobs.RetentionWorker`: the only existing River *periodic* job (as
  opposed to a job enqueued in reaction to a request), registered via
  `river.NewPeriodicJob(river.PeriodicInterval(24*time.Hour), ...)` in
  `jobs.NewClient`.

Events don't carry a single "start" column — `events.Date` (a DATE) plus
optional `StartTime`/`MeetTime` (`"HH:MM"` strings, Europe/Berlin) combine via
`events.EventStartInstant` into the absolute UTC instant already used for the
calendar feed's `DTSTART` and the `cancel_lead_minutes` cutoff. Reminders
reuse that same function rather than inventing a second notion of "when does
this event start".

## Goals

- A member can opt in/out of event reminders and pick a lead time (1–72h),
  per team, using the same UI surface as the existing push category toggles.
- A reminder is delivered exactly once per (event, member) pair, even though
  the periodic job re-scans the same upcoming events on every tick.
- No polling loop needs to know about every team member's individual lead
  time up front — the job discovers who's due on each tick.
- Stay inside the existing push-delivery pipeline (`push.Pusher`,
  `PushDeliveryWorker`, `push_subscriptions`) rather than building a second
  delivery path.

## Decisions

### Extend `PushCategoryPreferences`, don't add a new resource
`eventReminderEnabled`/`eventReminderHoursBefore` live on the same
`PushCategoryPreferences` object as `attendance`/`events`/`news`/`polls`/
`absence`, saved via the existing `PUT /teams/{teamId}/push-preferences`.
Alternative considered: a dedicated `/teams/{teamId}/event-reminder-preferences`
endpoint. Rejected — it's the same per-team, per-user, whole-object-PUT shape
the frontend (`usePushPreferencesActions`) already has, and splitting it
would mean two round trips and two loading states in a UI panel that already
reads as one settings block.

### A bounded, discovery-based periodic job, not per-user scheduled jobs
`EventReminderWorker` runs every 5 minutes and, on each tick, re-derives
"who is due a reminder right now" from three inputs: (1) events starting
within a fixed forward-looking window, (2) each candidate recipient's stored
`eventReminderHoursBefore`, (3) an idempotency table recording who's already
been reminded for which event. This avoids the alternative of enqueuing a
one-shot delayed job per (event, member) at event-creation time (River
supports `ScheduledAt`), which was rejected because: a member can change
their `eventReminderHoursBefore` *after* the event was created (the
scheduled job's fire time would already be wrong and there's no clean way to
requeue it), events can be edited (date/time change → every already-scheduled
reminder job is now wrong), and membership/preferences can change between
event creation and the event itself (a member who joins the team next week
should still get reminded about an event created before they joined).
Re-deriving from current state on every tick sidesteps all of that at the
cost of a periodic scan, which is cheap at this app's scale (a sports club's
teams, not a global multi-tenant SaaS).

Five minutes balances promptness (worst-case a reminder fires 5 minutes later
than the configured lead time) against load; the retention job's 24h cadence
is far too coarse for this, and firing every minute is unnecessary precision
for a "few hours before" reminder.

### Idempotency via a `(event_id, user_id)` marker table, not `notifications`
A new `event_reminders_sent (event_id, user_id, sent_at)` table, written with
`INSERT ... ON CONFLICT (event_id, user_id) DO NOTHING RETURNING 1` —
mirroring `NotificationWorker`'s own `river_job_id` `ON CONFLICT DO NOTHING`
idempotency trick, but keyed on the actual business identity (event ×
recipient) rather than a job ID, since this row's only job is "has this pair
already been notified", checked fresh on every tick rather than once per job
run. Only an actual insert (row returned) triggers the push enqueue, closing
the race between concurrent workers/replicas processing the same tick.
`ON DELETE CASCADE` from both `events` and `users` keeps it from
accumulating orphaned rows.

Reusing the `notifications` table (a new `event_reminder` type) was
considered and rejected: it would require extending
`gen.NotificationType`, `notifications.NotificationModule`,
`push.NotificationCategory`, and the frontend's `NotificationsSheet.tsx`
filter chips — machinery built for member-authored/system events shown in an
activity feed, not a purely time-triggered push with no feed presence (see
the proposal's explicit push-only scope decision).

### Window bound: query events within `[now-1d, now+96h]`, filter precisely in Go
`events.Repository`'s new lookahead query filters on the `date` column
(coarse, day granularity, cheap index range scan) to a window comfortably
covering the maximum configurable lead time (72h) plus a day of slack either
side for `EventStartInstant`'s Europe/Berlin conversion and events whose time
component pushes them across a date boundary. The *precise* "is this event's
start instant inside this member's window right now" check happens in Go
using `EventStartInstant`, exactly like `SetAttendance`'s
`cancel_lead_minutes` cutoff already does — there's no reason to duplicate
that timezone-aware arithmetic in SQL.

### Fan-out reuses the `permsChecker` gate, preference check is new
For each candidate event, `EventReminderWorker` lists team members via a new
`push.Repository.ListForTeam` (like `ListForTeamExcludingUser` but without
excluding an actor — a reminder has no "actor" to exclude), then for each
member: (1) `permsChecker.GetPermissions` + `notifications.HasReadAccess(...,
"events")` — a member who's lost read access to events shouldn't be reminded
about one, same as the existing notification path; (2)
`push.Repository.GetPreferences` — check `EventReminderEnabled` and read
`EventReminderHoursBefore`; (3) the idempotency insert. This is the same
three-gate shape `NotificationWorker.enqueuePushDeliveries` already uses
(permission → preference → dedupe-insert), just triggered by a clock tick
instead of a notification row.

### Reminder push payload has no `notifications` row behind it
The `PushDeliveryArgs` job enqueued directly carries a title/body built from
the event (e.g. "Reminder" / "<event title> in Xh") — there's no
`NotificationArgs` insert to derive it from, unlike every existing push path.

## Risks

- **Missed ticks (deploy restart, worker downtime) can skip a reminder
  entirely** if the qualifying window (lead-time hour) passes while the
  worker isn't running. Accepted: a missed push is a degraded, not broken,
  experience (the member still sees the event in-app), and this mirrors how
  the retention job already tolerates missed periodic runs.
- **Clock-tick granularity means "N hours before" is approximate**
  (±5 minutes). Documented in the UI copy as "about N hours before", not
  promised to the minute.
- **A 72h cap on `eventReminderHoursBefore`** bounds the lookahead query and
  is enforced by both a DB `CHECK` constraint and request validation
  (400 on an out-of-range value) — chosen to comfortably cover "remind me
  the night before an early morning event" without unbounded window growth.
