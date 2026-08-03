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
      **Confirmed from a HAR capture of a live session** (headers/URLs only, no
      response bodies): the real frontend follows `GET /events` with `POST
      /events/ajaxgetevents` (`offset` incrementing by 5, `old=true`) to page further
      into history; `spielerplus/events.go`'s `FetchEvents` now implements this loop,
      stopping when a page parses zero events. The year-less-date ambiguity is
      resolved via `resolveDate`, which picks whichever year keeps a date closest to
      the previously-resolved event in the same sequence (works walking forward *or*
      backward, unlike the old fixed "assume next year" rule). **Still open**: whether
      `old=true` truly needs to be resent on every page vs. only once (sent on every
      page as the safer assumption - harmless if wrong), and whether the fragment's
      date format/selectors match the full page's (unverified, no response body seen).
- [ ] 2.3 Parse per-event attendance into a per-member status; map SpielerPlus states →
      Teamverwaltung's `attendance.status` enum (`Zugesagt`→`yes`, `Unsicher`→`maybe`,
      `Absagen/Abwesend`→`no`, no response→`pending`). **Confirmed from the HAR
      capture**: `POST /events/ajaxgetparticipation` (`eventid`/`eventtype` form
      fields) is the real endpoint, called from a `GET /training/view?id=...` detail
      page - `spielerplus/attendance.go`'s `FetchAttendance` now calls it directly.
      **Still open and important**: the capture only shows it being called from a
      single training's detail page, so it's still unconfirmed whether the response
      lists *every* team member's status (what this importer needs) or only the
      caller's own (what the known community reference clients use this family of
      endpoint for) - if it's self-only, full-roster attendance needs a different,
      trainer-facing source. Row selectors also remain unverified (no response body).
- [ ] 2.4 Reverse-engineer and parse the team member/role list (names, emails,
      SpielerPlus role). **Partially confirmed from the HAR capture**: `GET /team`
      returned 200 when the roster was viewed and is now `spielerplus/members.go`'s
      `membersPath` (previously an unrelated guess, `/squad/members`) - but a `GET
      /site/team` was also observed (200), so it's not fully certain `/team` is the
      roster and not `/site/team`. Row selectors remain unverified (no response body).
- [ ] 2.5 Reverse-engineer and parse planned absences, including past ones and any
      "recurring weekday" absences. **Confirmed from the HAR capture**: `GET /absence`
      (not `/absences`, the previous guess) is the real list page, with `GET
      /absence/update?id=...` as a per-absence detail/edit page (not yet used - worth
      checking first if the list page doesn't expose enough, e.g. the recurring-weekday
      fields). `spielerplus/absences.go`'s `absencesPath` corrected accordingly. Row
      selectors and the recurring-weekday expansion remain unverified (no response
      body).
- [x] 2.6 Unit tests for all of the above against saved HTML fixtures, so future
      SpielerPlus markup changes fail a test run instead of corrupting an import.
      Fixtures are currently synthetic (matching this code's own selectors), **not
      yet captured from a real account** — swap in real captured markup once 2.2-2.5
      are verified live.

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
