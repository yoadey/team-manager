## Context

`stats` isn't a module today — CLAUDE.md documents this deliberately:
"`stats` isn't a module of its own: its GET operations carry
`x-rbac-module: events`, since its data is event/attendance-derived." That
was a reasonable simplification while statistics was just three read
endpoints piggybacking on `events`. It stops working once a team wants
different visibility for "who's coming to practice" (events) versus "how
often has each member actually shown up" (stats) — the latter is more
sensitive (performance/attendance tracking) and several teams using this
app have asked to restrict it to coaches. `stats-personal-presets` also
added six new endpoints as `x-rbac-module: public` (membership-only),
which was fine when there was no `stats` module to gate them with, but now
needs revisiting alongside the new module.

Every RBAC module in this codebase is threaded through the same six
places, enumerated per module (not data-driven from a slice/list), because
`genrbac`, `authz.go`, and `teams`/`roles`' Go structs are all
independently-hand-written mirrors of the OpenAPI `Permissions` schema —
see `absence-stats-relevance`'s design.md for the equivalent precedent
(`notifications.Service`'s `HasReadAccess` deliberately duplicating
`authz.go`'s switch, "per its own comment"). Adding a module is
consequently a checklist change across those places, not a single-point
edit; `cmd/genrbac`'s `validModules` map is the fail-closed gate that
catches a forgotten `x-rbac-module: stats` at `make generate` time, but
nothing catches a forgotten Go-side switch/struct field except tests and
runtime behavior (an unrecognized module falls through every `switch`'s
`default: return false` — fail-closed, not a crash, but silently denies
everyone including Admins if a case is missed).

## Goals / Non-Goals

**Goals:**
- Statistics visibility becomes independently controllable from `events`,
  defaulting to the same tier `Member` already gets for `events` (`read`)
  so this change is invisible to a team that doesn't touch its role
  config.
- Existing teams keep exactly the access they have today immediately after
  deploy — no re-configuration required for the common case (system
  Admin/Member roles), via a backfill migration.
- Defining a personal statistics preset (create/rename/delete) requires
  `stats: write`; viewing statistics, including the caller's own presets
  and remembered last-selected range, only requires `stats: read`.
- Remove the Fehlzeiten tab and all code that exists solely to serve it,
  leaving no dead handler/service/repository/schema behind.

**Non-Goals:**
- No change to `events`-module permissions or to any other existing
  module's semantics.
- No UI for bulk-editing existing custom (non-system) roles' new `stats`
  permission — they're backfilled to `none` and an admin adjusts them
  manually via the existing role editor, same as any other module.
- Not reconsidering whether presets should be team-shared instead of
  private-per-user — that's `stats-personal-presets`' decision, unchanged
  here.

## Decisions

**Presets (`create`/`update`/`delete`) require `stats: write`; preferences
(`get`/`put` last-selection) and listing presets stay at `stats: read`,
not self-service-exempt-from-read but also not write-gated.** This is a
deliberate asymmetry, not an oversight: a saved custom preset
("Saison 2026/27") is a named, reusable artifact the user is *defining*
for the area, analogous to other module `write` actions (e.g. `news:write`
to publish an article) even though it's private to its creator and never
visible to teammates. Saving the last-selected range, by contrast, is a
transparent side effect of merely looking at the page — gating it behind
`write` would mean a read-only member's stats page silently fails to
remember their last-viewed range every visit, which is a regression a
`read` grant shouldn't cause. `setStatsPreferences` is marked
`x-rbac-self-service: true` for this reason (per CLAUDE.md: self-service
exempts a caller from `write` on their own module, not from `none`) — it
still requires `stats: read` like every other stats GET, just never
`write`. `createStatsPreset`/`updateStatsPreset`/`deleteStatsPreset` are
deliberately *not* marked self-service, so their mutating methods fall
through to the normal `write`-required rule despite operating only on the
caller's own rows.

**Existing-team backfill only special-cases the two system roles.** The
migration runs
`UPDATE roles SET permissions = permissions || '{"stats":"write"}'::jsonb WHERE system AND name = 'Admin'`,
then the equivalent for `Member` → `'read'`, then a catch-all
`UPDATE roles SET permissions = permissions || '{"stats":"none"}'::jsonb WHERE NOT (permissions ? 'stats')`
for every other existing row (custom roles, and any future system role
this migration doesn't yet know about). Without a catch-all, a role row
missing the `stats` key would still work correctly at runtime — Go's
`json.Unmarshal` leaves `PermissionsJSON.Stats` as `""`, and every
`permLevelRank`/`foldMax`/`hasWritePermission`/`hasAnyPermission` treats
an unrecognized level the same as `"none"` (rank 0, fails both the read
and write checks) — but leaving it implicit means a future reader can't
tell "this role was never given stats access" from "this role predates
the module and was never migrated" by looking at the data. Writing `none`
explicitly makes every role row self-describing for all six-going-on-seven
modules.

**The absence table is removed, not deprecated or feature-flagged.** It
has no dependents (its `AttendanceStats.AbsenceStats` repository query and
`AttendanceAbsenceTable`/`Row` schemas are referenced nowhere else) and
CLAUDE.md's "don't use feature flags... when you can just change the code"
applies directly — there's no migration path to design because nothing
downstream consumes the removed data.

## Risks

- **Missing one of the six Go-side switch/struct sites for the new module
  silently fail-closes instead of erroring.** Mitigated by exercising
  `stats: read`/`write`/`none` explicitly in `authz_test.go` and
  `roles`/`teams` package tests, and by `cmd/genrbac`'s `validModules`
  catching a forgotten OpenAPI-side `x-rbac-module` at generate time.
- **Backfill migration touches every team's role rows in production.** The
  three `UPDATE ... WHERE` statements are narrow (system-role-name match,
  then a `NOT (permissions ? 'stats')` guard) and idempotent — re-running
  the migration is a no-op since every row already has the key after the
  first run.
