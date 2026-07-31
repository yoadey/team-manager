## Context

`internal/notifications` and its frontend counterpart were built assuming an
`attendance` notification type would be produced somewhere; the plumbing
(RBAC module mapping, DB column, rendering, push label in
`jobs.pushPayloadForNotification`) all reference it already. Only the
producer — `events.Service.SetAttendance` — was never wired up. This is a
gap-closing bugfix, not a new subsystem.

## Goals / Non-Goals

- Goal: every attendance response (yes/no/maybe) a member submits produces a
  notification row visible to teammates with `events` module read access
  (and a push, subject to the existing module-permission gate and the
  in-flight `add-push-notification-preferences` category gate).
- Goal: match the mock's existing semantics exactly (`frontend/src/mocks/handlers.ts:1043-1045`)
  so `serviceContract.test.ts` continues to hold both implementations to the
  same behavior once a contract test is added for this.
- Non-goal: this does not address the *other* event-lifecycle notification
  gaps noticed while investigating (`UpdateEvent`/`DeleteEvent`/reactivation
  don't enqueue `event_updated`/`event_deleted`/`event_reactivated` either,
  despite those types existing and being rendered). Those are pre-existing,
  separate issues, out of scope for the specific "Rückmeldungen fehlen"
  report this change addresses.
- Non-goal: no change to who can *see* an attendance notification — the
  existing `NotificationModule`/`HasReadAccess` gate (module `events`)
  already applies unchanged.

## Decisions

- **Actor is the responding member, not the caller.** `SetAttendance(eventID,
  callerID, userID, teamID, ...)` lets `callerID != userID` (an organizer
  setting someone else's attendance, gated by `events:write`). The
  notification's actor must be `userID` — "Alex said yes" — regardless of who
  clicked the button, matching the mock (`actorId: body.userId`) and what the
  frontend line renders (`n.actorName + ' ' + verb`).
- **Only yes/no/maybe notify, not pending.** `not_nominated` is already
  rejected earlier in `SetAttendance` (`ErrAttendanceStatusNotNominated`) and
  can't reach this point. `pending` is a valid target status (e.g. an
  organizer resetting someone's response) but isn't "feedback" — nothing to
  tell the team — and the frontend's `notifMeta` has no rendering branch for
  it (falls through to the "maybe" wording), so enqueuing it would render
  incorrectly. The mock encodes the identical rule.
- **`Status` added to `NotificationArgs` rather than reusing `Title`.** The
  `notifications.status` column already exists and is already read by
  `notifications.Repository.ListByTeamAndUser` and surfaced as
  `AppNotification.status` — only the write path (`NotificationArgs` /
  the worker's `INSERT`) was missing it. Adding the field is additive: every
  other call site continues to pass `nil`, which the nullable column already
  accepts.

## Risks / Trade-offs

- Existing notification rows created before this change have no `status`
  and are of a different `type` (`event_*`/`news`/`poll`/`absence`), so
  there is nothing to backfill.
- Enqueuing a notification on every RSVP change (including a member
  repeatedly toggling their own response) could be noisy in a very active
  team; this matches the mock's existing behavior and is consistent with
  how every other write in this codebase notifies unconditionally (no
  debouncing anywhere else in `internal/jobs`), so no debouncing is added
  here either.

## Migration Plan

None — no schema or API change.
