## 1. Project scaffolding

- [x] 1.1 Create `tools/spielerplus-import/` as its own Go module (`go.mod`, Go
      1.25+), not referenced from `backend/go.mod`, `backend/Makefile`, or
      `.github/workflows/ci.yml`.
- [x] 1.2 Add a `README.md` documenting: how to capture `SPIELERPLUS_SESSION_COOKIE`
      from a browser, required env vars (`DATABASE_URL`, `TEAM_ID`, role-mapping file
      path), and `--dry-run` usage.

## 2. SpielerPlus client (`spielerplus/`)

- [x] 2.1 HTTP client: cookie-jar session, browser-like `User-Agent` header, base URL
      `https://www.spielerplus.de`, session-cookie auth via
      `SPIELERPLUS_SESSION_COOKIE`; fail fast with a clear error on a redirect-to-login
      (expired/invalid session) instead of silently scraping an empty/login page.
- [x] 2.2 Parse the events list (`GET /events`) into structured events (type, title,
      date, time, location); resolve the "no year in the date" ambiguity and determine
      how to reach historical (non-upcoming) events, not just the default view.
      **Fully confirmed from a second HAR capture that included response bodies**:
      real markup is `div.panel[id="event-{type}-{id}"]`, title under
      `.panel-heading-text .panel-title`, year-less date ("DD.MM", no trailing dot)
      under `.panel-heading-info .panel-subtitle`, and up to three
      `.event-time-item .event-time-value` elements read positionally (meet/start/end,
      or start/end, or just start) rather than by German label text, so it doesn't
      depend on the account's display language. `spielerplus/events.go` fully rewritten
      to match. `POST /events/ajaxgetevents` (`offset` incrementing by 5, `old=true`)
      pages further into history, confirmed to return a JSON envelope
      (`{"html": ..., "count": N}`) - `count` is the authoritative end-of-history
      signal, used directly instead of guessing from parsed-row counts. The year-less
      date is resolved via `resolveDate`, which picks whichever year keeps a date
      closest to the previously-resolved event in the same sequence (works walking
      forward *or* backward). **Still open**: whether `old=true` truly needs to be
      resent on every page vs. only once (sent on every page as the safer assumption);
      only "training" events were present in the capture, so the row-id-derived type
      slug for game/tournament/event is unverified; no location field was observed in
      the capture (selector left as an unverified guess). **Validated against a real
      account's live dry-run**: paginated correctly through real history via
      `ajaxgetevents`. Found and fixed one gap the HAR captures didn't show: SpielerPlus
      renders an explicit `-:-` placeholder in an `.event-time-value` element for an
      unconfirmed time (e.g. a game's kickoff), rather than omitting the element -
      `eventTimes`/`parseDateTime` now normalize any placeholder value to "not set"
      instead of failing to parse it as `HH:MM`.
- [x] 2.3 Parse per-event attendance into a per-member status; map SpielerPlus states →
      Teamverwaltung's `attendance.status` enum. **Fully confirmed from the second HAR
      capture, including three full response bodies**: `POST
      /events/ajaxgetparticipation` (`eventid`/`eventtype` form fields) returns a JSON
      envelope (`{"html": ...}`) whose HTML groups *every* team member under
      `<div id="{code}-parti-collapse">` by status - **this resolves the earlier open
      question: it is the full-roster trainer view, not self-only**. The numeric group
      code maps 1:1 onto Teamverwaltung's own enum (`1`→yes, `2`→maybe, `0`→no,
      `99`→pending, and SpielerPlus's own `3`→"Nicht nominiert" maps directly onto
      Teamverwaltung's `not_nominated`) - matched via the code, not the group's German
      label text. Each member's id comes from their `.participation-list-user-photo`
      link (`/user/view?id=...`), and an optional decline reason from
      `.participation-list-user-reason .reason-text`, now also imported into
      `attendance.reason`. `spielerplus/attendance.go` fully rewritten to match.
- [x] 2.4 Reverse-engineer and parse the team member/role list (names, emails,
      SpielerPlus role). **Fully confirmed from a third HAR capture (`/team` and
      `/user/view?id=...`, response bodies included)**: `GET /team` is the roster,
      each entry a `.team-list-item` with the member's id/name/role
      (`.list-item-link` href, `.list-label-section .list-label`,
      `.user-role .user-role-item`) - but **the roster page does NOT show email
      addresses**. Each member's email only appears on their own profile page
      (`GET /user/view?id=...`) as a `mailto:` link, so `FetchMembers` now does one
      roster fetch plus one profile fetch per member (accepted N+1 cost for a
      one-off import). **Still open**: only the account owner's own profile was
      captured, so it's unverified whether another member's email renders the same
      way on their profile (role/privacy-based visibility could hide it) - a member
      with no visible email is skipped and logged rather than imported without one.
- [x] 2.5 Reverse-engineer and parse planned absences, including past ones and any
      "recurring weekday" absences. **Fully confirmed from a third HAR capture
      (`/absence`, response body included)**: `GET /absence` renders every absence
      type as tab-panes (`#absence-tab0`..`#absence-tab5`) in one page load, no
      separate pagination call; tab0 ("Aktuell") is a filtered "currently relevant"
      subset (confirmed by row-count cross-check) and is skipped in favor of tabs
      1-5, which together hold the complete history. Row markup
      (`.list-item.wrapmode`, absence id from `.list-item-link`, member id from a
      `/user/view?id=...` link, date range from `.list-value` in a
      "DD.MM - DD.MM.YY" format - the end date carries a 2-digit year, the start
      date doesn't and is resolved against it) is now implemented in
      `spielerplus/absences.go` to match. **Still open**: the "1 Tag pro Woche"
      (weekly-recurring, tab5) type was empty for this club, so there's no real
      example of how a populated recurring entry renders - `expandAbsences` still
      best-effort-detects a German weekday name in the reason text and falls back to
      importing a literal one-off range (logged) if none is found, rather than
      guessing wrong.
- [x] 2.6 Unit tests for all of the above against saved HTML fixtures, so future
      SpielerPlus markup changes fail a test run instead of corrupting an import. All
      four areas (events, attendance, members, absences) now use fixtures built to
      match the real confirmed markup structure (element/class names, JSON
      envelopes, the two-request member+email flow, the tab-scoped absence list) -
      not literal captured HTML (no real member data is committed to the repo), but
      structurally grounded rather than guessed. The one remaining gap is the
      "1 Tag pro Woche" recurring-absence markup itself (2.5), since no populated
      real example exists to build a fixture from.

## 3. Role mapping and idempotency state (`mapping/`)

- [x] 3.1 Load a config file mapping SpielerPlus role name → existing Teamverwaltung
      role name for the target team; fail the run with a clear error listing any
      SpielerPlus role with no mapping entry.
- [x] 3.2 Local JSON state file: load/save `spielerplus_id -> teamverwaltung_uuid`
      per entity type (events, absences); used to make re-runs idempotent for the two
      entities without a natural unique key.

## 4. Database writes (`db/`)

- [x] 4.1 `pgx` pool setup from `DATABASE_URL`; look up the target team's roles (by
      name, from `roles`/`membership_roles`) to resolve the role-mapping config against
      real role IDs.
- [x] 4.2 Upsert users by email: skip existing; for new users, insert with a
      non-empty placeholder `password_hash` and `email_verified_at` set immediately.
- [x] 4.3 Upsert `memberships` + `membership_roles` for the target `TEAM_ID` using the
      resolved role mapping; skip if the membership already exists.
- [x] 4.4 Insert `events` (using the state file for idempotency) with their real
      historical `date`.
- [x] 4.5 Upsert `attendance` per event/member (naturally idempotent on
      `UNIQUE(event_id, user_id)`), setting `status` from the mapped enum.
- [x] 4.6 Insert `absences` (using the state file for idempotency), enforcing
      `to_date >= from_date`, span `<= 1095` days, and no overlap with an existing
      absence for that user — report and skip violations rather than aborting the run.

## 5. Orchestration and dry-run (`import/`, `main.go`)

- [x] 5.1 Orchestrate the full run in order: members/roles → events → attendance →
      absences, each step logging counts of created/skipped/failed records.
- [x] 5.2 `--dry-run` flag: run the full read/scrape + mapping pipeline and print what
      would be written, without opening a DB write transaction.
- [x] 5.3 Summary report at the end of a real run (counts per entity, any skipped
      records with reasons) so the operator can sanity-check before onboarding real
      users.
- [x] 5.4 Throttle requests to SpielerPlus (configurable minimum gap, default 500ms,
      `SPIELERPLUS_REQUEST_DELAY`) so a full run's many per-member/per-event requests
      don't hammer it in a tight loop.
- [x] 5.5 Confirm the active SpielerPlus team before writing anything: SpielerPlus
      scopes every page this tool reads by a session-level "active team" with no team
      id in the URL to double check against, so an account managing more than one team
      could have the wrong one active. Detect the active team's name (confirmed
      markup: the sidebar's "Team wechseln" nav card) and either compare it against
      `SPIELERPLUS_EXPECTED_TEAM_NAME` (non-interactive) or prompt for an explicit
      `y`/`yes` on stdin (interactive) before proceeding.

## 6. Verification

- [x] 6.1 `openspec validate spielerplus-data-migration --strict` passes.
- [x] 6.2 `cd tools/spielerplus-import && go build ./... && go vet ./...` passes.
- [x] 6.3 `go test ./...` passes (HTML-fixture parsing tests, role-mapping/idempotency
      logic tests).
- [ ] 6.4 Manual dry-run against a local/staging Postgres (e.g. `docker compose`) with
      a disposable team; inspect the would-write summary. **Not run** — this sandbox
      has no Docker/Postgres daemon available; do this against a real staging DB
      before any production run.
- [ ] 6.5 Manual real run against the same disposable team/DB; verify rows via `psql`;
      re-run and confirm no duplicates are created. **Not run**, same reason as 6.4.
- [ ] 6.6 End-to-end: a pre-created user completes `POST /auth/forgot-password` →
      reset → login, and sees their imported events/attendance/absences in the
      Teamverwaltung frontend. **Not run** — depends on 6.4/6.5 having created real
      data first.
- [x] 6.7 Confirm existing backend CI gates (`make test`, `make lint`, `make build`,
      frontend `npm run lint`/`typecheck`/`test`/`build`) are untouched — this change
      adds no files under `backend/` or `frontend/` (verified via `git status`).
