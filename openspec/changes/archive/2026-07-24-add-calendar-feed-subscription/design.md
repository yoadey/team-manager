## Context

Standard API auth in this backend is a cookie-based JWT session
(`auth.UserFromContext`), enforced by `authHandler.AuthMiddleware` on the
entire `/api/v1` router group in `cmd/server/main.go`, before
`RequireMembership`/`RequirePermission` even run. Calendar apps (Google
Calendar, Apple Calendar, Outlook, etc.) poll a subscription URL on their own
schedule and cannot present a session cookie or run this app's login flow —
the URL itself must be the credential.

The closest existing "secret in the URL" precedent, `/invites/{code}/accept`,
still requires the redeemer to be logged in — it links a code to an
already-authenticated caller, which doesn't help here. `pagination.Paginator`'s
HMAC-signed opaque cursors are the closest *mechanism* precedent (a
tamper-evident secret derived from a server key) but are designed to encode
short-lived pagination state, not a long-lived, individually revocable
credential — an HMAC-only design can only be invalidated by rotating the
global signing key, which would silently break every user's feed link at
once. A DB-backed token (mirroring how `invites.code` already works) is
revocable per row instead.

`events` has no timezone concept at all (`internal/events/model.go`'s
`Date`/`MeetTime`/`StartTime`/`EndTime` are bare `DATE`/wall-clock strings,
no `TIMESTAMPTZ`, no per-team TZ column). The frontend already commits to
`Europe/Berlin` everywhere it turns those into absolute instants
(`useCalExportActions.ts`'s `zonedTimeToUtc(..., 'Europe/Berlin')`,
called out explicitly in a comment there). The server-side renderer keeps
that same hardcoded assumption rather than inventing new backend TZ
handling as a side effect of this change.

## Goals / Non-Goals

**Goals:**
- A calendar app can subscribe once and see events created, updated, or
  cancelled after that point, without the user re-sharing a link.
- The feed reflects only events the token's user can currently see (their
  team membership + `events` module read permission), checked fresh on every
  request — not frozen at token-issue time.
- A user can revoke a leaked link without operator involvement, and get a
  fresh one immediately.
- No new backend timezone modeling — reuse the frontend's existing
  `Europe/Berlin` assumption.

**Non-Goals:**
- No per-event-type or per-role filtering of the feed in v1 — it mirrors
  exactly what the user would see in the app's own event list.
- No change to the existing client-side one-time `.ics` download
  (`downloadIcs()`) — it keeps working unchanged; this change only makes the
  *subscribe* link real.
- No general-purpose "public API token" mechanism — this token is scoped
  narrowly to serving one ICS feed, not a bearer credential for other routes.

## Decisions

**Token shape and storage.** `calendar_feed_tokens.token` is a 32-byte
`crypto/rand` value, hex-encoded (matching `teams.generateCode`'s existing
`rand.Read` + `hex.EncodeToString` pattern in `internal/teams/repository.go`),
stored in the clear (unlike session tokens, which are hashed) — a leaked feed
token is meant to be trivially revocable and replaceable, not something the
DB needs to protect against its own compromise the way a password or session
token does; this mirrors `invites.code`, which is also stored in the clear.
One active token per `(user_id, team_id)`: issuing a new one sets the old
row's `revoked_at` (or deletes it outright — decided during implementation;
either satisfies "old link stops working immediately").

**Authorization happens inside the handler, not the middleware chain.**
`GET /calendar-feed/{token}.ics` is declared `security: []` in the OpenAPI
spec (same as `/auth/*`) and, like those routes, is registered in
`cmd/server/main.go` as a manual `r.Get(...)` call *after* the generated
mux — chi's "last registration wins" means this override replaces
whatever the generated mux registered for that operation inside the
authenticated group, keeping the unauthenticated route from ever passing
through `AuthMiddleware`. Inside the handler, `Service.ServeFeed` does the
authorization the middleware chain would normally do: look up the token,
confirm `revoked_at IS NULL`, re-derive the bound user's current team
membership and `events` permission via the same `members.Repository`
methods `RequireMembership`/`RequirePermission` use, and return 404 (not
403, to avoid confirming a token *exists* but is merely unauthorized-now) if
either check fails.

**Feed content.** `Service.ServeFeed` reuses `events.Service`'s existing
`ListEvents`-equivalent query path (scoped to the token's team) rather than
introducing a second events query — filtered to non-cancelled events, same
as `buildIcs()` already filters (`e.status !== 'cancelled'`). ICS rendering
(`internal/calendarfeed/ics.go`) is a direct Go port of
`useCalExportActions.ts::buildIcs()`'s escaping/folding/UID rules, so the
two outputs stay recognizably equivalent even though a one-time download and
a live subscription now come from different code paths.

**URL shape.** The issued URL is
`{PUBLIC_BASE_URL-derived API origin}/api/v1/calendar-feed/{token}.ics`,
built server-side from the existing `PUBLIC_BASE_URL`/`ALLOWED_ORIGINS`
config (`internal/config/config.go`) rather than a new env var — the
`.ics` suffix is kept in the path (not just cosmetic) since some calendar
clients sniff the extension to decide whether a URL is a calendar feed.

## Risks / Trade-offs

- **Token in a URL is inherently shareable/loggable** (browser history,
  proxy access logs, calendar-app sync logs) — this is the accepted nature
  of every calendar subscription link (Google/Apple links work the same
  way); the mitigation is revocability, not secrecy-by-obscurity alone.
- **No rate limiting specific to this route beyond the existing global
  `RATE_LIMIT_RPS`.** Calendar clients typically poll hourly to daily, so
  this is judged sufficient; a per-token limiter can be added later if abuse
  is observed.
- **Re-checking permissions on every request** adds a DB round-trip per feed
  fetch (membership + permission lookup) on top of the events query itself —
  accepted, since it's the only way to honor "none must hide events" for a
  URL that, unlike a cookie session, can't be invalidated by logging out.
