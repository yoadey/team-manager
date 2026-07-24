## Why

`CalExportSheet.tsx` (`frontend/src/features/events/components/CalExportSheet.tsx`)
already advertises a calendar-subscription URL
(`https://teamverwaltung.app/cal/{teamId}.ics`) and a "copy link" action
(`useCalExportActions.ts::copyCalUrl`), but the URL is a hardcoded,
non-functional placeholder — no backend route serves it, and the component
displays a permanent warning (`t('events.calPrototypeNote')`) admitting this.
A real one-time `.ics` download already works entirely client-side
(`buildIcs()`/`downloadIcs()`), but a calendar *subscription* — which a
calendar app polls on its own schedule — requires a stable server URL,
since calendar apps cannot run this app's JavaScript or use its
cookie-based session.

There is no existing precedent in this codebase for an unauthenticated,
secret-token-bearing GET route: every genuinely unauthenticated route
(`security: []`) takes its secret in the request body (`/auth/login`,
`/auth/register`, `/auth/verify-email`), and every path-embedded secret
(`/invites/{code}/accept`) still requires an authenticated caller on top of
it. A calendar feed is a new kind of route for this backend — auth lives
entirely in an opaque, revocable, per-user token in the URL, checked inside
the handler itself rather than via the cookie/RBAC middleware chain.

## What Changes

- New `calendar_feed_tokens` table: one row per (user, team), holding a
  high-entropy random token, `created_at`, and a nullable `revoked_at`.
- New `internal/calendarfeed` package: `Service.IssueToken` (mints/rotates
  a token for a `(userID, teamID)` pair — revokes any existing active token
  for that pair first) and `Service.ServeFeed` (resolves a bare token to its
  `(userID, teamID)`, re-checks current team membership and `events`-module
  read permission *at request time*, then renders that user's visible
  events as an `.ics` feed).
- Server-side ICS rendering ported from the existing client-side generator
  (`useCalExportActions.ts::buildIcs()`) into Go — line folding, escaping,
  stable per-event `UID`, `DTSTAMP`, `Europe/Berlin`-anchored `DTSTART`/`DTEND`
  (the backend has no per-team timezone concept today; this preserves the
  frontend's existing hardcoded assumption rather than introducing a new one).
- New OpenAPI operations:
  - `POST /teams/{teamId}/calendar-feed/token` (issue/rotate; `x-rbac-module:
    events`, `x-rbac-self-service: true` — any member with events read access
    can mint their own feed link) returning the ready-to-use `webcal://`/
    `https://` URL.
  - `DELETE /teams/{teamId}/calendar-feed/token` (revoke).
  - `GET /calendar-feed/{token}.ics` (`security: []`, registered outside the
    authenticated router group in `cmd/server/main.go`, the same way
    `/auth/login`/`/auth/register` already are) — returns `text/calendar`.
- `CalExportSheet.tsx` calls the real issue endpoint instead of building a
  fake URL, and the placeholder warning is removed once the link is real.

## Capabilities

### New Capabilities
- `calendar-feed`: per-user, per-team, revocable ICS subscription feed
  serving that user's currently-visible events to any standards-compliant
  calendar client.

### Modified Capabilities
<!-- none -->

## Impact

- Backend: new `internal/calendarfeed/{model.go,repository.go,service.go,
  handler.go,ics.go}`, `cmd/server/main.go` (unauthenticated route
  registration, matching the existing `/auth/*` pattern), `internal/metrics/business.go`
  (feed request counters, optional).
- Database: new migration `internal/db/migrations/00031_calendar_feed_tokens.sql`.
- API contract: `backend/openapi/openapi.yaml` (three new operations, one new
  schema), regenerated `internal/gen/api.gen.go` and
  `frontend/src/api/types.gen.ts`.
- Frontend: `features/events/components/CalExportSheet.tsx`,
  `features/events/hooks/useCalExportActions.ts` (`copyCalUrl` now fetches a
  real token instead of formatting a placeholder), `services/serviceLayerReal.ts`,
  `mocks/{handlers.ts,db.ts}`, `services/serviceContract.test.ts`,
  `i18n/{en.ts,de.ts}` (drop `calPrototypeNote` copy or repurpose it for a
  genuine caveat if one remains).
- Docs: none beyond the OpenAPI spec itself (no new env vars — the feed's
  base URL reuses the existing `PUBLIC_BASE_URL`).
