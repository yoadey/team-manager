## Context

Calendar clients subscribe to an HTTPS `.ics` URL and re-poll it on their own schedule (hours). They cannot authenticate with the app's session cookie, so the feed must be reachable by a URL-embedded secret. Today only a client-side one-time `.ics` exists (`useCalExportActions.ts`); the subscribe link is a dead prototype. Member `birthday` and team events already exist server-side. `PUBLIC_BASE_URL` is available for building absolute links. Google/Apple/Outlook all consume the **same** HTTPS feed (`webcal://` is just Apple's trigger scheme for the same URL) — so one endpoint serves all three.

## Goals / Non-Goals

**Goals:**
- One stable URL a user subscribes to once; the calendar app keeps it fresh.
- Works in Google Calendar (Android, "From URL"), Apple Calendar ("Add subscribed calendar"), and Outlook ("Subscribe from web").
- Feed carries the team's events + member birthdays, timezone-correct.
- The URL is a secret, scoped to what the user may see, and revocable.

**Non-Goals:**
- Two-way sync / writing back from the calendar app (feeds are read-only by nature).
- Cross-team merging (that's `cross-team-events` / `shared-team-calendar`).
- Controlling client refresh frequency (clients decide; we only set sensible cache headers).

## Decisions

- **Auth model:** a high-entropy random token (≥256 bit) per team membership, stored **hashed** (look up by hash of the presented token). Endpoint e.g. `GET /calendar/{token}.ics` sits **outside** the session-auth + RBAC middleware — the token *is* the credential. The token resolves to a (user, team) and the feed serializes exactly the events/birthdays that member may see (same visibility rules as in-app). Rotating the token invalidates old subscriptions.
- **Scope:** per-membership feed (this user, this team) — matches the current per-team export UX and keeps visibility simple. A merged all-teams feed can follow later.
- **Content/format:** reuse the existing `.ics` shape (VCALENDAR + VEVENTs) but generate it **server-side** so it's authoritative and identical across clients; emit a proper `VTIMEZONE` for Europe/Berlin rather than only UTC stamps, so all-day birthdays and local times import correctly. Birthdays = yearly all-day `VEVENT` (`DTSTART;VALUE=DATE`, `RRULE:FREQ=YEARLY`), skipping members without a birthday or that the caller may not see.
- **Configurable content:** the feed includes **all** event types by default; the subscriber can restrict it to any subset of the event `type` values and toggle birthdays. Store the selection **server-side, tied to the feed token** (e.g. a `types text[]` + `include_birthdays bool` on `calendar_feed_tokens`), edited via the app — so the subscription URL never changes and the calendar app reflects a changed selection on its next refresh. Rationale for server-side over query params: the URL a user already pasted into their calendar app can't be edited afterwards, so a stored selection is the only way "change what's in my feed" works without re-subscribing. An unknown/removed type simply yields no events of that type.
- **Caching:** send `Cache-Control`/`ETag` so clients poll efficiently; content is generated per request (small teams).
- **Frontend:** the calExport sheet shows the real URL, a copy button, a `webcal://` deep link, per-platform instructions (i18n keys already exist), and a "regenerate link" action; the one-time download stays as a fallback.

## Risks / Trade-offs

- **The URL is a bearer credential** (DSGVO-relevant: it exposes event titles/locations and birthdays to anyone holding it). Mitigate: high-entropy token, stored hashed, revocable via regenerate, scoped to the member's own visibility, HTTPS-only, and no other PII (no emails/phones) in the feed. Document that sharing the link shares the calendar.
- **Unauthenticated route** must be carefully placed around the auth/RBAC middleware and rate-limited to resist token brute-forcing (constant-time-ish lookup by hash, plus the global per-IP limiter). It must not become an open enumeration surface.
- Timezone correctness across clients (Apple/Google/Outlook parse `VTIMEZONE` differently) — test the generated feed against real client import, not just a byte comparison.
- New migration + storage for tokens; keep it minimal (membership_id, token_hash, created_at).
