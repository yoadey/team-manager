## Why

Web Push today (`backend/internal/push/`, capability `push-notifications`)
is all-or-nothing per browser: a member either has a subscription registered
(and gets pushed every notification type their current module permissions
allow, across *every* team they belong to) or has none at all
(`push.Repository.ListForTeamExcludingUser` joins `push_subscriptions` to
`memberships` purely to find current members — there is no concept of "push
for team A but not team B", let alone "push for events but not news"). A
member active in one team's day-to-day events but only loosely following
another team's news has no way to quiet one without losing the other, and no
way to mute a category (e.g. poll pushes) they find noisy in a specific team
while keeping it in another.

## What Changes

- New `push_preferences` table (`team_id`, `user_id`, one boolean column per
  notification category: `attendance`, `events`, `news`, `polls`,
  `absence`), owned entirely by `internal/push` — a missing row means "all
  categories enabled" (today's behavior), so existing subscribers are
  unaffected until they explicitly change something.
- `push.NotificationCategory(notifType string) string` maps a
  `gen.NotificationType` to one of those five categories (mirrors
  `notifications.NotificationModule`'s switch, but only push cares about
  `absence` and `attendance` as distinct categories from `events`/`polls`,
  matching the frontend's own notification-feed filter groups
  (`NotificationsSheet.tsx`'s `attendance`/`events`/`other` chips)).
- `internal/jobs/notification_worker.go`'s `enqueuePushDeliveries` gains a
  second, independent gate alongside the existing module-permission check:
  after a subscription passes the permission gate, it's also checked against
  the recipient's per-team category preference; either gate failing skips
  that push (permissions gate what a member is *allowed* to see; preferences
  gate what they've *asked* to be pushed).
- New self-service endpoints `GET /teams/{teamId}/push-preferences` and
  `PUT /teams/{teamId}/push-preferences`, `x-rbac-module: public` (a member
  manages only their own preferences within a team they belong to).
- Frontend: five per-team toggles (one per category), added inline to
  `ProfileSheet` directly below the existing global push opt-in toggle,
  gated on the same "push already enabled for this browser" state that
  toggle already tracks — the per-team toggles are meaningless (and hidden)
  until a subscription exists, and scoped to the currently active team.

## Capabilities

### Modified Capabilities
- `push-notifications`: delivery is now additionally gated by the
  recipient's own per-team, per-category opt-out, on top of the existing
  module-permission gate.

## Impact

- Backend: new `internal/push/{model.go,repository.go,service.go,handler.go}`
  additions (`CategoryPreferences`, `GetPreferences`/`UpsertPreferences`,
  handler methods), `internal/jobs/notification_worker.go` (second gate in
  `enqueuePushDeliveries`), `cmd/server/main.go` (no new wiring — reuses the
  existing `push.Repository` instance), tests across all of these.
- Database: new migration `internal/db/migrations/00014_push_preferences.sql`
  (+ its index migration if required by the migration-safety lint).
- API contract: `backend/openapi/openapi.yaml` (two new operations, one new
  schema `PushCategoryPreferences`), regenerated `internal/gen/api.gen.go`,
  `internal/middleware/rbac_table.gen.go`, `frontend/src/api/types.gen.ts`.
- Frontend: new preferences sheet + hook under `features/notifications/`,
  `services/serviceLayerReal.ts`, `mocks/{handlers.ts,db.ts}`,
  `services/serviceContract.test.ts`, `i18n/{en.ts,de.ts}`.
