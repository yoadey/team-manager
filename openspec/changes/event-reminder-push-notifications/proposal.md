## Why

Web Push today only fires reactively — a notification row (news, event
lifecycle change, poll, attendance, absence) is created and, if the
recipient's module permission and per-category preference both allow it,
pushed (`backend/internal/jobs/notification_worker.go`,
`backend/internal/push/`). There is no proactive reminder: a member who RSVP'd
"yes" to a training three days ago gets no nudge as it approaches, and has to
remember to check the app. Members have asked for a "remind me before it
starts" push, with a configurable lead time since a goalkeeper who needs to
arrive early wants a longer reminder window than someone who just shows up on
time.

## What Changes

- New per-(team, user) push preference, alongside the existing category
  toggles: `eventReminderEnabled` (default `true`) and
  `eventReminderHoursBefore` (integer hours, 1–72, default `6`) — extends
  `PushCategoryPreferences` / `GET|PUT /teams/{teamId}/push-preferences`
  rather than adding new endpoints, since it's the same per-team push-opt-in
  surface members already know.
- New periodic backend job (`jobs.EventReminderWorker`, River periodic job
  like the existing daily `RetentionWorker`) that runs every 5 minutes,
  finds non-cancelled upcoming events whose start instant
  (`events.EventStartInstant`) has just entered a team member's configured
  reminder window, and enqueues one Web Push delivery per qualifying
  subscription — gated by the same `events` module read-permission check
  `NotificationWorker` already applies, plus the new preference.
- New `event_reminders_sent` table recording (event_id, user_id) once a
  reminder has been enqueued for that pair, so a member is reminded exactly
  once per event regardless of how many 5-minute ticks their window spans or
  how many replicas run the periodic job concurrently.
- Settings → Notifications gets a new toggle + hours input under the
  existing per-team push preferences panel.
- Deliberately push-only: no new entry is added to the in-app notification
  feed / `notifications` table for reminders, since the ask is specifically
  "delivered by push notification" and the feed's `NotificationType` enum,
  `notifications.Service`, and `NotificationsSheet.tsx` rendering are a
  materially larger surface than this change needs to touch.

## Capabilities

### Modified Capabilities
- `push-notifications`: adds a new, time-triggered (rather than
  event-triggered) push delivery path with its own opt-in, opt-out and
  configurable lead time, on top of the existing category-preference and
  module-permission gates.

## Impact

- `backend/openapi/openapi.yaml` — extend `PushCategoryPreferences` schema
  (`eventReminderEnabled`, `eventReminderHoursBefore`); regenerate
  `internal/gen/api.gen.go` and `frontend/src/api/types.gen.ts`
  (`make generate` / `make generate-ts`).
- `backend/internal/db/migrations/` — new migration: `push_preferences`
  gets two new columns; new `event_reminders_sent` table.
- `backend/internal/push/` — `model.go` (`CategoryPreferences` fields,
  default), `repository.go` (Get/UpsertPreferences SQL, new
  `ListForTeam` for reminder fan-out), `handler.go` (wire the two new
  fields through `toGenPreferences`/`SetPushPreferences`).
- `backend/internal/jobs/` — new `event_reminder_worker.go` (+test);
  `NewClient` registers it as a `river.PeriodicJob`; `notification_worker.go`
  unchanged (reminders are not notification rows).
- `backend/internal/events/` — small repository addition to list
  non-cancelled events starting within a bounded lookahead window across all
  teams (reminders are computed by a global periodic job, not per-team).
- `backend/cmd/server/main.go` — construct and register the new worker.
- `frontend/src/features/notifications/` — `types.ts`
  (`PushCategoryPreferences` fields), `hooks/usePushPreferencesActions.ts`
  (default object, setter for the new fields).
- `frontend/src/features/settings/components/NotificationsPanel.tsx` — new
  toggle + numeric hours input.
- `frontend/src/i18n/` (de/en) — new strings.
- `frontend/src/mocks/db.ts` / `handlers.ts` — mock parity for the contract
  test (`serviceContract.test.ts`).
- No RBAC/`x-rbac-module` change: `push-preferences` is already
  `x-rbac-module: public`.
