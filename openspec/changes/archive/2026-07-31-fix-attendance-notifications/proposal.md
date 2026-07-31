## Why

Attendance responses ("Rückmeldungen" — a member RSVPing yes/no/maybe to an
event) never show up in the team's notification feed, and never trigger a
Web Push, even though the feature looks fully built end-to-end: the
`NotificationType` enum has an `attendance` value
(`backend/openapi/openapi.yaml`), `notifications.NotificationModule` already
maps it to the `events` RBAC module (`backend/internal/notifications/service.go:57-62`),
the `notifications` table has a `status` column for it
(`backend/internal/db/migrations/00001_init.sql:256`), the frontend's
`NotificationsSheet` fully renders it including its own "Attendance" filter
chip (`frontend/src/features/notifications/components/NotificationsSheet.tsx:56-73,141`),
and the frontend's own mock backend already creates one on every RSVP
(`frontend/src/mocks/handlers.ts:1043-1045`).

The one missing piece is the real Go backend: `events.Service.SetAttendance`
(`backend/internal/events/service.go:558-616`) persists the attendance row
but never calls `jobEnqueuer.EnqueueNotification`, unlike `CreateEvent` and
`SetStatus` in the same file, which do enqueue `event_created`/
`event_cancelled` notifications. Compounding this, `jobs.NotificationArgs`
(`backend/internal/jobs/notification_worker.go:34-43`) has no `Status`
field at all and the worker's `INSERT INTO notifications` (line 114-119)
never sets the `status` column — so even a caller that tried to enqueue an
`attendance` notification today would end up with a row the feed can't
render correctly (`NotifMeta` dereferences `n.status!` for `attendance`
rows in `NotificationsSheet.tsx:57`).

## What Changes

- `jobs.NotificationArgs` gains a `Status *string` field; `NotificationWorker.Work`'s
  insert includes it.
- `events.Service.SetAttendance` enqueues an `attendance` notification
  (team, actor = the member whose attendance this is, event id/title/date,
  status) whenever the new status is an actual response (`yes`/`no`/`maybe`)
  — mirroring the mock's existing behavior. A reset to `pending` (no
  response yet) does not notify, matching the frontend's rendering, which
  only distinguishes yes/no/maybe for this type. Enqueuing is best-effort
  (logged, not fatal), matching the existing `CreateEvent`/`SetStatus` code
  path.
- No API contract, schema, or migration change — `NotificationType.attendance`,
  the `status` column, and the frontend rendering all already exist; this
  closes the one gap that stopped them from ever being exercised.

## Capabilities

### New Capabilities
- `attendance-notifications`: an event's own attendance responses generate
  team notifications (and, transitively, Web Push deliveries) the same way
  event/news/poll activity already does.

## Impact

- Backend: `backend/internal/jobs/notification_worker.go` (`NotificationArgs.Status`,
  insert column), `backend/internal/events/service.go` (`SetAttendance`),
  plus tests (`events/service_test.go`, `jobs/notification_worker_test.go`).
- No frontend change needed (already correct); no OpenAPI/migration change.
