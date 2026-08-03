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
