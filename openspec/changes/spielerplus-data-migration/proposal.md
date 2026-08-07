## Why

The club currently runs its calendar and attendance tracking in a third-party tool
(SpielerPlus) and wants to move to Teamverwaltung. A cold-start migration would lose
the club's history: past training/game dates, who attended which training, and
planned absences (including past ones) that members and coaches rely on for
statistics. SpielerPlus has no public API, so this has to be a one-off, operator-run
import against data scraped from a logged-in SpielerPlus session, executed before real
users log in for the first time.

## What Changes

- Add a new, standalone Go tool (`tools/spielerplus-import/`, its own module, not part
  of the `backend` build/CI) that:
  - Authenticates to SpielerPlus using a session cookie the operator captures from
    their own logged-in browser session (no SpielerPlus password stored anywhere).
  - Scrapes SpielerPlus's server-rendered pages for team members/roles, events
    (trainings/games), per-event attendance, planned absences (including past
    ones), and finances: the general cashbox ledger, membership dues, and
    penalties (catalog + assigned punishments).
  - Writes directly to the Teamverwaltung Postgres database (same `DATABASE_URL` as
    the backend), bypassing the backend HTTP API, since there is no bulk-import
    endpoint and some of what's needed (pre-verified accounts with no usable password,
    historical dates) falls outside normal user-facing flows.
  - Pre-creates a `users` row per SpielerPlus member (matched by email) with no usable
    password and email already marked verified, so the person's later "forgot
    password" flow (`POST /auth/forgot-password`) works the first time they try to log
    in, instead of forcing self-registration.
  - Maps SpielerPlus roles to existing Teamverwaltung roles in the target team via a
    fixed configuration table, and creates the corresponding `memberships` /
    `membership_roles` rows.
  - Is safe to re-run: previously-imported users (by email), events/absences (via a
    local state file mapping SpielerPlus IDs to Teamverwaltung UUIDs), and attendance
    (already unique per event+user) are not duplicated on a second run.
  - Supports a `--dry-run` mode that reports what would be written without touching
    the database.

This does not change any backend/frontend product code, the OpenAPI contract, or the
database schema — it is an operational tool run once (or a few times, iteratively)
per club migrating off SpielerPlus.

## Capabilities

### New Capabilities
- `spielerplus-migration-tooling`: a standalone, re-runnable import tool that pulls
  members, events, attendance, absences, and finances (cashbox transactions,
  membership dues, penalties) out of a SpielerPlus account and loads them into
  Teamverwaltung ahead of real user logins.

### Modified Capabilities
<!-- none -->

## Impact

- New directory `tools/spielerplus-import/` (own `go.mod`, not wired into
  `backend/Makefile` or `.github/workflows/ci.yml`).
- Reads/writes the shared Postgres database directly: `users`, `memberships`,
  `membership_roles`, `event_series`, `events`, `attendance`, `absences`,
  `transactions`, `penalties`, `penalty_assignments`, `contributions` (see
  `backend/internal/db/migrations/00001_init.sql`) — no migration/schema change.
  Existing business rules normally enforced by `backend/internal/absences/service.go`
  and `backend/internal/auth/service.go` (absence overlap/span checks, verified-email
  requirement, non-empty `password_hash` for `ForgotPassword` to act) must be
  reproduced by the tool itself, since it does not go through those services.
  Its own idempotency state (a local mapping file) lives outside the repo/DB — no
  Teamverwaltung schema change.
- No changes to `backend/openapi/openapi.yaml`, `internal/gen`, or the frontend.
- Operational/manual: requires a real SpielerPlus session and a target team already
  created in Teamverwaltung (`TEAM_ID`) before the tool runs.
