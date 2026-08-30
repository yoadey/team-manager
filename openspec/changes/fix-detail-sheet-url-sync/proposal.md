## Why

Reported bug: "Die URL wird nicht bei jeder Navigation aktualisiert, z. B.
Termine. Dadurch kann man nicht überall einfach über die history zurück
gehen." (The URL isn't updated on every navigation, e.g. appointments —
so the browser Back button doesn't work everywhere.)

`frontend/src/context/urlState.ts`'s `buildPath` only emits the
`/events/<id>` or `/members/<id>` detail path when the *current
top-level route* (`s.route`) already equals `'events'`/`'members'`:

```ts
if (s.detail && (s.route === 'events' || s.route === 'members')) {
  path += '/' + encodeURIComponent(s.detail.id);
}
```

But `openEventDetail`/`openMemberDetail` open their sheet without
switching `state.route` when called from a different route — e.g.
`EventCard` on the Home route (`components/cards.tsx`) or an
event-linked entry in the notifications sheet
(`features/notifications/components/NotificationsSheet.tsx`), both
reachable from any route via the always-visible bell icon. In that case
`s.route` is still `'home'` (or whatever route the sheet was opened
from), the `if` above never matches, and `buildPath` silently falls
through to producing the unchanged `/home` path — the URL is never
updated and no history entry is pushed. Pressing Back then skips past
the app entirely instead of closing the detail sheet, exactly the
reported symptom.

## What Changes

- `buildPath`: key a detail sheet's path off `detail.kind` (`event` →
  `/events/<id>`, `member` → `/members/<id>`) instead of requiring the
  current `route` to already match. This makes the URL update correctly
  regardless of which route the detail sheet was opened from.
- `AppContext.tsx`'s state→URL sync effect: refine push-vs-replace so
  opening a detail sheet always pushes a new history entry (even across
  a route switch) and closing one always replaces (so Back doesn't
  resurrect a sheet that was already closed) — today's logic only
  handled push/replace correctly for the same-route case.

## Capabilities

### Added Capabilities
- `state-based-routing`: opening an event/member detail sheet always
  updates the browser URL and creates a Back-able history entry,
  regardless of which route it was opened from.

## Impact

- `frontend/src/context/urlState.ts` (+ tests).
- `frontend/src/context/AppContext.tsx` (state→URL sync effect).
- No backend changes.
