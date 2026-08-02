## Why

`notifications.Service.List` filters every returned item server-side by
the caller's permission on that item's originating module, since
`notifications` itself carries no route-level RBAC gate
(`x-rbac-module: public`). The module lookup (`NotificationModule`,
`backend/internal/notifications/service.go:70-77`) returns `""` — meaning
"no module gate, visible to everyone" — for any notification type it
doesn't recognize in its `switch`. Its sibling `HasReadAccess` explicitly
fails *closed* for the same "unclassified" situation. The `default:` case
in `NotificationModule` has its own comment calling this a "safety net for
a malformed/future DB row", but a fail-open default is exactly backwards
for a safety net: during a rolling deploy where a newer server version
writes a notification of a type an older, still-serving pod's `switch`
doesn't recognize, that item is shown to every team member for the
duration of the rollout — regardless of their actual `read`/`none`
permission on the module it really belongs to.

## What Changes

- `NotificationModule`'s `default:` case returns a sentinel that
  `HasReadAccess` treats as denied, instead of `""` (public/unrestricted).
- Add a regression test that simulates an unrecognized/future notification
  type and asserts it is excluded from `List` for a member who does not
  hold blanket (e.g. settings-admin) access.

## Capabilities

### Added Capabilities
- `authorization`: notification items with an unclassifiable module fail
  closed (denied) rather than open (shown to everyone), matching the
  fail-closed default the rest of the RBAC system uses.

## Impact

- `backend/internal/notifications/service.go` (+ `service_test.go`).
- No API or migration changes; no change to `HasReadAccess`'s existing
  fail-closed behavior, which this now matches.
