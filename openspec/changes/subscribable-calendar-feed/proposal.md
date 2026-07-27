## Why

The calendar "subscribe" flow is a **prototype**: `frontend/src/features/events/hooks/useCalExportActions.ts` (`copyCalUrl`) hands out a dead `webcal://teamverwaltung.app/cal/{id}.ics` URL, and the only working path is a **one-time** `.ics` download (`downloadIcs`) that never updates. Users want a real calendar they can **subscribe to once** and have it **refresh automatically** in Google Calendar (Android), Apple Calendar (iOS/macOS), and Outlook. Birthdays should be part of that feed too.

## What Changes

- Add a **server-side, subscribable iCal feed**: a stable per-membership URL returning live `text/calendar`, re-fetched periodically by the calendar client — no manual re-export.
- Authenticate the feed by an **unguessable, revocable token** in the URL (calendar apps can't send a session cookie), scoped to the requesting user's membership so it only exposes events/birthdays that member may see.
- Feed content: **all** of the team's events — every event `type` (training, matches/tournaments, performances, …) — **plus member birthdays** (yearly all-day), with correct timezone handling (`VTIMEZONE` Europe/Berlin) so imports land at the right time.
- **Configurable content:** the subscriber chooses which categories the feed contains — any subset of the event types and whether birthdays are included — stored per feed so the URL stays stable and the calendar app picks up the change on its next refresh. Default: everything on.
- Frontend: replace the prototype link with the real feed URL + copy/`webcal://` actions and the per-platform subscribe instructions already drafted in i18n (`calGoogleDesc`/`calAppleDesc`); let the user **regenerate** the token to revoke old subscriptions. Keep the one-time `.ics` download as a secondary option.

## Capabilities

### New Capabilities
- `calendar-feed`: a subscribable, auto-refreshing team calendar feed (events + birthdays) importable into Google/Apple/Outlook.

## Impact

- Spec/backend: `openapi.yaml` — a token-authenticated feed endpoint returning `text/calendar` (outside the normal session-auth/RBAC path) plus endpoints to get/rotate a membership's feed token; new `calendar_feed_tokens` storage (+ migration); a server-side `.ics` builder (events + birthdays, `VTIMEZONE` Europe/Berlin) in a new `internal/calendar`; regenerate clients; tests.
- Frontend: `useCalExportActions.ts` + the calendar-export sheet (real URL, copy/webcal, regenerate-token, per-platform instructions), `frontend/src/i18n/{de,en}.ts`, MSW.
- Ops: document the feed URL as a **bearer/capability link** (secret, revocable) and its cache/refresh cadence in `docs/`.
- CI: openapi-drift, migration gates, backend + frontend gates. **API + migration-affecting; the feed route bypasses session auth by design — cover it explicitly.**
