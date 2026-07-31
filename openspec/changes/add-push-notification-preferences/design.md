## Context

Push subscriptions are per-browser and team-agnostic by design (one
subscription "covers push delivery for every team the user belongs to",
per the original `add-web-push-notifications` proposal) — that part is not
changing. What's missing is a per-(team, member) *preference* layer on top:
which notification categories, within a given team, should actually reach
that subscription. This sits alongside, not instead of, the existing
module-permission gate in `notifications.HasReadAccess` — permission is
about what a member is authorized to see at all; preference is about what
they've chosen to be interrupted for.

## Goals / Non-Goals

- Goal: a member can, per team, turn off push for one or more of:
  attendance responses, event lifecycle changes, news, polls, absences —
  independently of their module permissions and independently of any other
  team they belong to.
- Goal: default behavior for a member who never visits the new settings is
  identical to today (all categories pushed, subject to the existing
  permission gate) — no migration/backfill surprises.
- Goal: preference categories line up with the notification feed's own
  grouping (`NotificationsSheet.tsx`'s filter chips: attendance / events /
  other, where "other" further splits into news/poll/absence for push
  purposes) so the settings UI reads naturally against the feed a member
  already sees.
- Non-goal: no per-device preferences (e.g. "push events on my phone but
  not my laptop") — preferences are per (team, user), matching how
  permissions themselves are scoped, not per subscription row.
- Non-goal: the in-app notification feed itself is unaffected — it already
  shows everything the member's module permissions allow; preferences only
  affect the push channel.

## Decisions

- **New table, not a JSONB column on `memberships`.** `push_preferences`
  is entirely `internal/push`'s to own (schema, repository, tests), no
  different from `push_subscriptions` itself or `notifications.notif_seen`
  (also a standalone `(team_id, user_id)`-keyed table owned by its
  package). Piggybacking a column onto `memberships` would make the
  `members` package responsible for data it never reads, and would need a
  cross-package import for `push` to interpret its JSONB shape.
- **A missing row means "everything enabled", not "everything disabled".**
  Every existing subscriber has no row today. Defaulting a missing row to
  all-`false` would silently stop all push delivery for every current
  subscriber the moment this ships. The repository's `GetPreferences`
  returns an in-memory default-all-true struct on `pgx.ErrNoRows`; `PUT`
  upserts an explicit row only when a member actually changes something.
- **Five boolean columns, not a JSONB blob.** Mirrors `roles.permissions`'
  precedent of "small, fixed, known set of keys" but as plain columns
  (rather than JSONB) since — unlike RBAC modules, which can grow — the
  five categories are fixed by `gen.NotificationType`'s enum and a JSONB
  default-merge dance (like `teams.PermissionsJSON`) buys nothing here; a
  straight `SELECT`/`UPDATE` of five booleans is simpler to reason about
  and index.
- **Preference gate is independent of, and applied after, the permission
  gate.** `enqueuePushDeliveries` already computes `module` (permission
  gate) once per notification and re-checks per-subscriber; the category
  (preference gate) is likewise computed once per notification
  (`push.NotificationCategory(a.Type)`) and checked per-subscriber
  alongside the existing `allowedCache` permission cache, added as a
  second cache keyed the same way. A subscriber must pass both to receive
  a push; failing either is silent (matches the existing "no push, no
  error" behavior for a permission failure).
- **`push.NotificationCategory` is a separate mapping from
  `notifications.NotificationModule`, not a reuse.** The RBAC module gate
  collapses `attendance` into the `events` module (there's no
  `events:attendance` sub-permission), but the *preference* UI wants
  attendance responses separately toggleable from event-lifecycle
  changes, matching the notification feed's own "attendance" vs "events"
  filter chips. Reusing `NotificationModule` would conflate the two and
  remove the one distinction this feature is for.
- **Endpoint lives under `/teams/{teamId}/push-preferences`, not
  `/teams/{teamId}/notifications/push-preferences`.** It's a `push`-owned
  resource (tag `push`, handled by `push.Handler`), not a `notifications`
  one — consistent with `/users/me/push-subscriptions` already being
  `push`-tagged rather than folded under `/teams/{teamId}/notifications`.

## Risks / Trade-offs

- Five fixed boolean columns means a sixth notification category in the
  future needs its own migration + column + UI toggle, rather than an
  automatically-extensible JSONB map. Accepted: the same is already true
  of `roles.permissions`' module set, and `gen.NotificationType` changes
  rarely enough that this isn't a real cost in practice.
- Two independent per-subscriber gates (permission + preference) inside
  `enqueuePushDeliveries`'s existing loop add one more DB round trip in
  the worst case (a cache miss on both). Both are already scoped to "once
  per distinct user in this notification's recipient list", so the added
  cost is bounded and consistent with the existing permission-cache
  pattern.

## Migration Plan

- New migration `00016_push_preferences.sql`: `CREATE TABLE
  push_preferences (team_id UUID NOT NULL REFERENCES teams(id) ON DELETE
  CASCADE, user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  attendance BOOLEAN NOT NULL DEFAULT true, events BOOLEAN NOT NULL DEFAULT
  true, news BOOLEAN NOT NULL DEFAULT true, polls BOOLEAN NOT NULL DEFAULT
  true, absence BOOLEAN NOT NULL DEFAULT true, updated_at TIMESTAMPTZ NOT
  NULL DEFAULT now(), PRIMARY KEY (team_id, user_id))`. No index migration
  expected (the primary key already covers the only lookup pattern,
  `WHERE team_id = $1 AND user_id = $2`), but the migration-safety CI gate
  gets the final say per the existing `CREATE INDEX CONCURRENTLY` +
  `NO TRANSACTION` pattern if it disagrees.
- No backfill needed — see "a missing row means enabled" above.
