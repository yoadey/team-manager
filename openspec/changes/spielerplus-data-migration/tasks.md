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
      slug for game/tournament/event is unverified. **Location, fixed after live
      testing found it never imported**: the events list page has no location field
      at all (confirmed from a second, targeted HAR capture of a live event detail
      page) - it only appears on an event's own detail page
      (`GET /training/view?id=...`, confirmed; `/game/`, `/tournament/`, `/event/`
      inferred by the same slug pattern, unconfirmed) as a single `.info-area` block
      labeled "Adresse". `FetchEvents` now fetches each event's detail page as a
      separate per-event request (same accepted N+1 cost as the members/email fetch)
      and sets `Location` from it; a missing/failed fetch is logged and the event is
      still imported, just without a location. **Validated against a real
      account's live dry-run**: paginated correctly through real history via
      `ajaxgetevents`. Found and fixed one gap the HAR captures didn't show: SpielerPlus
      renders an explicit `-:-` placeholder in an `.event-time-value` element for an
      unconfirmed time (e.g. a game's kickoff), rather than omitting the element -
      `eventTimes`/`parseDateTime` now normalize any placeholder value to "not set"
      instead of failing to parse it as `HH:MM`. Also found and fixed: a multi-day
      event's end time can render with a trailing "am DD.MM." (e.g. "17:00 am
      17.11."), not a bare "HH:MM" - `parseHM` parses only the leading
      whitespace-delimited token for the time-of-day. **Extended once
      Teamverwaltung gained multi-day event support** (migration
      `00025_event_multi_day`, adding `events.end_date`): that trailing date is no
      longer discarded - `trailingDate`/`parseDateTime` now also resolve it (via
      the same closest-year `resolveDate` logic, anchored to the start day) into
      `Event.EndDate`, set only when it's genuinely a later day than the start (a
      same-day trailing date, or one that fails to resolve, leaves `EndDate` unset)
      - see 4.4.
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
      **Extended after live testing**: also imports each member's birthday
      (`users.birthday`) from the same profile page fetch (no extra request), read
      from a labeled `.col-md-4.col-sm-6` field block matching the German label
      "Geburtstag" - the same block markup already confirmed for other profile
      fields. **Still open**: no populated birthday was present in the original HAR
      capture, so the "DD.MM.YYYY" value format and the German-only label match are
      unverified against a real value; a present-but-unparseable birthday is logged
      and skipped (not fatal to the member import), matching the tool's existing
      per-record error handling. Only written for newly created users - an existing
      user's birthday is left untouched, consistent with "Existing account is left
      alone" (spec.md). **Also extended**: `Member.PhotoURL` is read straight off the
      roster row's own `.user-icon img` (no extra request, unlike email/birthday) -
      see 4b for the upload side.
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
      historical `date`, plus `end_date` for a multi-day event (see 2.2's
      `EndDate`). **Adapted to the backend's cross-team-events change**
      (migration `00035_event_teams`, developed on a separate branch and merged
      after this task was first written): every event now also needs a matching
      `event_teams (event_id, team_id)` row - since that migration, every
      read/RSVP path scopes visibility via an `EXISTS` join against
      `event_teams` instead of `events.team_id` directly, so an event with no
      row there is invisible to its own team. `db.InsertEvent` now writes both
      rows in a single transaction (never one without the other, so a
      mid-write failure can't strand a permanently invisible event that the
      local state file would then also wrongly mark as already imported).
- [x] 4.5 Upsert `attendance` per event/member (naturally idempotent on
      `UNIQUE(event_id, user_id)`), setting `status` from the mapped enum.
- [x] 4.6 Insert `absences` (using the state file for idempotency), enforcing
      `to_date >= from_date`, span `<= 1095` days, and no overlap with an existing
      absence for that user — report and skip violations rather than aborting the run.

## 4a. Finances (`spielerplus/finances.go`, `spielerplus/dues.go`,
      `spielerplus/punishments.go`, `db/finances.go`)

- [x] 4a.1 Parse the cashbox ledger (`GET /cashbox`, paginated via
      `<ul class="pagination">` `?page=N&per-page=25`) into `Transaction` records
      (title, date, amount, income/expense from the amount's sign). **Confirmed
      from a HAR capture** (initially truncated, recovered via brace-matching
      JSON repair to salvage complete HAR entries): same `.list-item[data-key]`
      convention as `/team`/`/absence`, full "DD.MM.YYYY" dates (unlike events'
      year-less dates). Insert into `transactions` (using the state file for
      idempotency, since it has no natural unique key).
- [x] 4a.2 Parse the membership-dues matrix (`GET /cashbox/dues`) into per-
      (member, due-column) `Due` records (label, amount, paid/unpaid).
      **Confirmed from the same HAR capture**: a `<table>` with one header `<th>`
      per due column (label from a `title` attribute, amount trimmed off the
      header's own text) and one `<tr>` per member, each cell's
      `onclick="toggleCashbox(this, <memberID>, <dueColumnID>)"` giving both the
      member id (a reliable join to the imported roster, unlike penalties - see
      4a.3) and the column's own SpielerPlus id. Insert into `contributions`, one
      row per (member, column) — using the state file for idempotency, since
      **as of the backend's `flexible-membership-fees` change**
      (`00018_flexible_membership_fees`), `contributions` has no "month" column
      or unique constraint left to upsert against at all. `db.InsertContribution`
      writes `name` (the column label) directly; no `due_date` (SpielerPlus gives
      none) and no paid/open status (no longer a stored column - see 4a.4). This
      replaced an earlier version that spread a member's columns across
      synthetic consecutive months to work around the old schema's
      one-row-per-month limit — no longer needed once that limit was removed;
      see design.md.
- [x] 4a.3 Parse the penalty catalog (`GET /punishment-catalog/index`) and
      assigned punishments (`GET /punishments/index`, paginated by analogy to
      `/cashbox` — unconfirmed for this specific page, the captured club had too
      few assignments to trigger pagination). **Confirmed from a second,
      dedicated HAR capture** (the first capture found this club's punishments
      list genuinely empty — a valid state, not a markup gap — so the user
      re-captured after assigning a real punishment): both pages use the
      `.list-item[data-key]` convention; the catalog gives label+amount per
      entry, the assignment list/detail view give **member name only, no id or
      profile link anywhere** (an exception to every other page this tool
      reads). Insert catalog entries into `penalties` and assignments into
      `penalty_assignments` (each via the state file for idempotency), matching
      an assignment's member by exact name against the imported roster and its
      reason text against a catalog label — either one not matching is logged
      and skipped (member name) or falls back to a direct amount/label snapshot
      with a `NULL` `penalty_id` (reason), never guessed.
- [x] 4a.4 **Adapted to the backend's `flexible-membership-fees` change**
      (migrations `00018_flexible_membership_fees`,
      `00020_penalty_assignment_linked_payment`, developed on a separate branch
      and merged after 4a.2/4a.3 were first written): `contributions.status` and
      `penalty_assignments.paid` were both dropped in favor of paid status
      derived from income `transactions` linked via
      `transactions.contribution_id`/`penalty_assignment_id`. Rather than
      heuristically matching an imported cashbox transaction to the due/penalty
      it might pay (rejected as too risky for financial data — see design.md),
      every imported due/penalty is left unlinked; `Summary.DuesPaidNotLinked`/
      `PenaltiesPaidNotLinked` count how many were actually paid on SpielerPlus
      so the run summary can tell an operator how many to reconcile by hand.
- [x] 4a.5 Write a SpielerPlus-source-id breadcrumb into each imported
      transaction/contribution/penalty assignment's free-text note/description
      field (`transactions.note` since migration `00024_transaction_note`;
      `contributions.description` since `00022_contribution_archive_description`;
      `penalty_assignments.note` has existed since `00008_penalty_assignment_note`
      but was unused by this importer until now), so a treasurer reconciling
      imported records can trace one back to its SpielerPlus id.

## 4b. Member photos (`storage/`, `spielerplus/client.go`'s `FetchAsset`,
      `importrun/run.go`'s `importMemberPhoto`)

- [x] 4b.1 Parse each roster member's photo URL directly off the `/team` page
      (`.user-icon img`'s `src`) — no extra request needed, confirmed from the
      same HAR capture as 2.4's roster markup. SpielerPlus's `default.svg`
      placeholder for a member with no custom photo is detected by substring and
      treated as "no photo", not imported.
- [x] 4b.2 Add a minimal, write-only S3-compatible object store client
      (`storage.Store`, `Put` only) using the same `S3_*` env vars/semantics as
      the backend's own object storage, since this tool's separate Go module
      can't import `backend/internal/storage` directly (Go's `internal/`
      visibility rule) — see design.md.
- [x] 4b.3 Fetch a member's photo bytes (`spielerplus.Client.FetchAsset`, a
      generic absolute-URL GET reused for any future asset fetch, still
      throttled), validate them (JPEG/PNG only, 2 MB cap, decompression-bomb
      pixel-count guard — matching the backend's own upload validation, no
      resize since SpielerPlus's rendition is already small), and upload under
      the same object key the backend itself would use (`users/<id>/photo`),
      only for newly created users (existing accounts' photos are left alone,
      same as birthday). Entirely optional: unset `S3_ENDPOINT`/`S3_BUCKET`
      skips photo import for the whole run without failing it. Any per-member
      failure (fetch/validate/upload/DB write) is logged and skipped, not fatal
      to that member's import.

## 5. Orchestration and dry-run (`import/`, `main.go`)

- [x] 5.1 Orchestrate the full run in order: members/roles (including a photo
      upload per newly created member, when configured) → events (including a
      per-event location fetch) → attendance → absences → transactions → dues →
      penalties, each step logging counts of created/skipped/failed records.
      Finances steps and photo uploads are treated as supplementary: a
      fetch/parse/upload failure there is logged and skipped rather than
      aborting the whole run, unlike members/events which the rest of the run
      depends on.
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
