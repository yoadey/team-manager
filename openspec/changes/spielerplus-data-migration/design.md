## Context

Teamverwaltung's backend (`backend/`) has no bulk-import endpoint, no "pending member
without a user account" concept (`memberships.user_id` is `NOT NULL`), and no
external-ID columns for reconciling re-imports (see `backend/internal/db/migrations/00001_init.sql`).
SpielerPlus has no documented/public API: several community projects
(`christianwehe/calendar-sync`, `janic0/autospieler`, `DrTobe/dauerzusagesendung`)
confirm it's a server-rendered Yii2 app, scraped via a logged-in HTTP session rather
than called via JSON API. From those projects we already know concrete details for
login and events/attendance; absences and the member/role list have no reference and
must be reverse-engineered live against a real account during implementation.

A first HAR capture of a live session (browsing events, a training's participation
widget, and the absences pages) confirmed several endpoints directly, though it did
not include response bodies (headers/URLs only): `GET /events` (initial page)
followed by `POST /events/ajaxgetevents` (`offset`/`old` form fields) is how the real
frontend pages both forward and backward through history - answering what was
previously an open question about reaching historical events. `POST
/events/ajaxgetparticipation` (`eventid`/`eventtype` form fields) is the real
per-event attendance endpoint. `GET /absence` (not `/absences`) is the real absences
list, with `GET /absence/update?id=...` as a per-record detail/edit page. `GET /team`
is the best candidate for the member roster (a `GET /site/team` was also observed and
returns 200, role unconfirmed).

A second HAR capture (of the same kind of session, this time exported with response
bodies included) confirmed the actual markup for events and attendance, closing what
were previously open questions there: events are `div.panel[id="event-{type}-{id}"]`
with the title under `.panel-heading-text .panel-title`, the year-less date ("DD.MM",
no trailing dot - the earlier guessed format had one) under `.panel-heading-info
.panel-subtitle`, and up to three `.event-time-item .event-time-value` elements for
meet/start/end times, read positionally rather than by their German label text so
parsing doesn't depend on the account's display language. `ajaxgetevents` and
`ajaxgetparticipation` both turned out to return a JSON envelope
(`{"html": "...", "count": N}` / `{"html": "..."}`), not raw HTML - `count` is the
authoritative end-of-pagination signal. Critically, `ajaxgetparticipation`'s HTML
groups *every* team member by status under `<div id="{code}-parti-collapse">`,
confirming it's the trainer/full-roster view this importer needs, not a self-only one
as the earlier open question worried - the numeric group code (not the localized label
text) maps 1:1 onto Teamverwaltung's `attendance.status` enum, including SpielerPlus's
own "Nicht nominiert" group mapping directly onto `not_nominated`. `spielerplus/events.go`
and `spielerplus/attendance.go` were rewritten to match. This second capture was
truncated before reaching the member roster or absences pages, so those two remain
unverified guesses (see tasks.md 2.4/2.5).

A third round of HAR captures (`/team`, `/user/view?id=...`, and `/absence`, all with
response bodies) closed those two remaining gaps. `GET /team` is the roster
(`.team-list-item` rows), but - unexpectedly - it does not show member email
addresses at all; those only appear on a member's own profile page
(`GET /user/view?id=...`) as a `mailto:` link, so `FetchMembers` now does one roster
fetch plus one profile fetch per member. Only the account owner's own profile was
captured, so it's unverified whether viewing another member's profile shows their
email the same way (role/privacy-based visibility could differ) - a member with no
visible email is skipped and logged rather than imported without one.
`GET /absence` turned out to render every absence type as tab-panes
(`#absence-tab0`..`#absence-tab5`) in a single page load, with no separate pagination
call: tab0 ("Aktuell") is a filtered "currently relevant" subset - confirmed by
cross-checking row counts, tab0's count exactly equals the type tabs' total minus the
type tabs' own count, i.e. every tab0 row is a duplicate of one already in its type
tab - so the importer reads tabs 1-5 (which together hold the complete history) and
skips tab0. Each absence's date range ("02.08 - 09.08.26") has an inconsistent format:
the end date carries a 2-digit year, the start date doesn't and is resolved against
the end date's year (adjusted back a year if that would put start after end, handling
a Dec-to-Jan range). The "1 Tag pro Woche" (weekly-recurring) type was empty for this
club, so there's no confirmed example of how a populated recurring entry renders -
`expandAbsences` best-effort-detects a German weekday name/abbreviation in the
reason text and expands by that; if none is found, it imports a single literal
one-off range instead of guessing at a recurrence pattern with zero evidence.

The user has decided this is a standalone tool, not backend-integrated, written in Go,
authenticating to SpielerPlus with a manually-captured browser session cookie, using a
fixed SpielerPlus-role → Teamverwaltung-role mapping table, and tracking its own
import state locally instead of extending the Teamverwaltung schema.

## Goals / Non-Goals

**Goals:**
- Re-runnable, dry-run-capable import of members, events, attendance, and absences
  (including historical/past-dated ones) from one SpielerPlus team into one
  Teamverwaltung team.
- Imported members can complete Teamverwaltung's existing "forgot password" flow on
  first login, without going through self-registration/email verification.
- No changes to backend product code, OpenAPI contract, or DB schema.

**Non-Goals:**
- No ongoing/continuous sync between SpielerPlus and Teamverwaltung — this is a
  one-time (or few-time, while iterating) cutover tool, not a background job.
- No UI in Teamverwaltung for running or configuring the import.
- No attempt to also *write back* to SpielerPlus (e.g. via the known
  `ajax-participation-form` endpoint) — read-only against SpielerPlus.
- No general-purpose SpielerPlus API client library — only what this import needs.

## Decisions

**Standalone Go module under `tools/spielerplus-import/`.** Kept out of
`backend/go.mod` and `.github/workflows/ci.yml` deliberately: it depends on scraping
an external, undocumented site with credentials that don't belong in CI, and its
correctness isn't part of the product's normal build/test/deploy gates. `go build`/`go
vet`/unit tests for this tool are run manually, not as a CI gate (see tasks.md
Verification).

**Auth via a captured browser session cookie, not stored credentials.** The operator
logs into SpielerPlus in their own browser and passes the session cookie via env var
(`SPIELERPLUS_SESSION_COOKIE`). This avoids persisting a SpielerPlus password anywhere
in the tool's config/state, at the cost of the operator having to refresh the cookie
if it expires mid-import (acceptable for a low-frequency, operator-driven tool).
Requests need a browser-like `User-Agent` — SpielerPlus 403s the default `requests`
(and presumably Go `net/http`) UA per the community projects' notes.

**Direct Postgres writes, bypassing the backend HTTP API/services.** There's no bulk
endpoint, and two things the import needs are outside what the app-facing services
support at all: (a) creating a `users` row that's pre-verified with no real password
yet (self-registration always starts unverified), and (b) inserting historical
`attendance`/`absences` rows with real past dates and — for attendance — an
`at` timestamp that isn't `now()`. The tool therefore talks to Postgres directly via
`pgx`, and reimplements the invariants those services would otherwise enforce:
- `users`: non-empty but unusable placeholder `password_hash` (e.g. a bcrypt hash of a
  random UUID) so `ForgotPassword` (`backend/internal/auth/service.go:419`) doesn't
  no-op it as "OIDC-only"; `email_verified_at` set at insert time so the retention job
  (`backend/internal/jobs/retention.go:109`) doesn't purge it before 7 days pass.
- `attendance.status` restricted to the same enum the DB CHECK constraint expects
  (`yes|no|maybe|pending|not_nominated`); SpielerPlus's three states plus "no response"
  map onto four of those (`not_nominated` is never produced by this import).
- `absences`: `to_date >= from_date`, span `<= 1095` days, and no overlapping absence
  per user — checked by the tool before insert (matching what
  `backend/internal/absences/repository.go:218`'s `userHasOverlappingAbsence` does at
  the app layer), reporting and skipping violations rather than aborting the whole run.

**Idempotency via a local state file, not a schema change.** The user explicitly
chose not to add external-ID columns to the backend schema. Instead:
- `users` are naturally idempotent on the unique `email` column — no extra state
  needed.
- `attendance` is naturally idempotent on the existing `UNIQUE(event_id, user_id)`
  constraint — re-import upserts.
- `events` and `absences` have no natural external key, so the tool persists a local
  JSON state file (`spielerplus_id -> teamverwaltung_uuid`, per entity type) next to
  wherever it's run, read on startup and updated after each successful write. This
  file is operator-local (not committed, not part of the DB) and is the only thing
  that makes a second run against the same SpielerPlus data safe.

**Fixed role-mapping table, not automatic inference.** SpielerPlus role names (e.g.
Trainer/Co-Trainer/Spieler) are club-specific text, and Teamverwaltung roles are
per-team custom rows created by the club through the normal UI. The tool takes a small
config file mapping SpielerPlus role name → an existing Teamverwaltung role name in the
target team; unmapped SpielerPlus roles fail the run loudly (not silently defaulted)
so the operator fixes the mapping instead of getting a wrong permission level.

**Recurring weekly absences get expanded to concrete date ranges at import time.**
SpielerPlus supports a "fixed weekday" recurring absence; Teamverwaltung's `absences`
table only has concrete `from_date`/`to_date`. The importer expands each recurring
absence into the set of concrete past (and up to the import date) ranges it actually
covers, rather than inventing a recurrence concept in Teamverwaltung.

**Absences and member/role scraping have no existing reference and are reverse-engineered
during implementation.** Login and events/attendance endpoints are already known from
community projects (see proposal.md/Impact). The absences and member-list pages are
not covered by any of them, so their URLs/selectors are determined by inspecting the
real SpielerPlus account during implementation (with the operator's help if needed),
not assumed up front.

**Request throttling and active-team confirmation, added after initial review.**
Two operational gaps surfaced once the tool was otherwise complete: nothing kept a
full run's many requests (one per member's profile, one per event's attendance) from
firing back-to-back, and nothing checked that the SpielerPlus session's currently
*active* team (there is no team id in any URL this tool calls - SpielerPlus scopes
every page by a session-level "active team" instead) was actually the one intended,
which matters for any account that manages more than one team. Both are now handled
before any data is written: `spielerplus.Client` enforces a configurable minimum gap
between requests (`SPIELERPLUS_REQUEST_DELAY`, default 500ms), and `main.go` fetches
the active team's display name (confirmed markup: the sidebar's "Team wechseln" nav
card) and requires it be confirmed - either interactively (`y`/`yes` on stdin) or
against `SPIELERPLUS_EXPECTED_TEAM_NAME` for repeat/non-interactive runs - before
proceeding.

## Risks / Trade-offs

- **Scraping fragility**: SpielerPlus can change its markup at any time and silently
  break parsing (same risk the reference community projects call out). Mitigated by
  keeping HTML parsing isolated behind a small interface with fixture-based unit tests,
  so breakage is caught by re-running tests against a freshly captured page rather than
  discovered mid-import.
- **Bypassing app-layer validation**: writing directly to Postgres means bugs in the
  tool's re-implemented invariants (enum values, absence overlap/span) could corrupt
  data in ways the backend would normally reject. Mitigated by dry-run-first and by
  testing against a disposable/staging database before any production run.
- **Session cookie expiry**: a long-running import (many teams/seasons) may outlive the
  captured cookie. The tool fails fast with a clear "re-authenticate" error on a 401/
  redirect-to-login rather than silently importing partial/empty data.
- **ToS considerations**: scraping a third-party site outside its documented API may be
  against SpielerPlus's terms of service. This is the club's own account and data, same
  as the precedent community projects; flagged here, not blocking implementation.
