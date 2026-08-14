## Why

The statistics page's "Fehlzeiten" (absence table) tab
(`frontend/src/pages/Stats.tsx`'s `'absences'` tab, backed by `GET
/teams/{teamId}/stats/absences`) lists every member absence with its event
and date, alongside the quota and matrix views that already surface the
same underlying attendance data in a more useful, aggregated form. It
provides no actionable information beyond what those two views already
show and should be removed outright, not just hidden.

Separately, the entire statistics area (`internal/stats`, `stats-view-preferences`)
is currently gated by the `events` module (`x-rbac-module: events` on
`getStatsOverview`/`getAttendanceMatrix`/`getMemberStats`) or not gated at
all (`x-rbac-module: public` on the `stats-preferences`/`stats-presets`
endpoints added by `stats-personal-presets`). A team that wants to show
event details/RSVPs to everyone (`events: read`) but keep attendance
statistics restricted to coaches has no way to do that — the two are
inseparable today. Statistics needs its own RBAC module.

## What Changes

- **Remove the "Fehlzeiten" absence-table tab entirely**: the frontend tab,
  its query hook, the `GET /teams/{teamId}/stats/absences` endpoint, and
  the backend service/repository code and schemas that exist solely to
  serve it. The quota and matrix tabs are unaffected — they use their own,
  separate queries.
- **New `stats` RBAC module** (`none | read | write`), alongside the
  existing `events | members | finances | news | polls | settings`. It
  gates the statistics overview, attendance matrix, per-member statistics,
  and the personal preferences/presets endpoints (currently `events` or
  `public`).
- **The default `Member` role grants `stats: read`** on new teams, same
  tier as its `events`/`members`/`news`/`polls`/`settings` defaults. The
  default `Admin` role grants `stats: write`.
- **Existing teams are backfilled**: a migration adds `stats` to every
  existing role's permissions JSONB — `write` for the system `Admin` role,
  `read` for the system `Member` role, `none` for any other existing
  custom role (unset, admin-adjustable afterwards) — so installed teams
  don't silently lose statistics access the moment this ships.
- **`stats: write` gates defining personal statistics presets.** Viewing
  the statistics area (including the caller's own saved presets and their
  last-selected date range) only requires `stats: read`. Creating,
  renaming, or deleting a named custom preset — "eigene feste Bereiche",
  introduced by `stats-personal-presets` — additionally requires `stats:
  write`. Saving the caller's last-selected range is an automatic
  side effect of viewing, not a deliberate "define a preset" action, so it
  stays at `read`.

## Capabilities

### New Capabilities
- `stats-access-control`: the `stats` RBAC module gating the statistics
  area, its default role grants, the existing-team backfill, and the
  read/write split between viewing statistics (incl. presets) and defining
  new presets.

### Modified Capabilities
- `attendance-statistics`: the attendance-matrix endpoint's authorization
  requirement now names the `stats` module instead of `events`.

### Removed Capabilities
- `attendance-absence-table`: the absence-table tab and its endpoint are
  removed; no replacement.

## Impact

- Database: new migration `backend/internal/db/migrations/00032_stats_rbac_module.sql`
  backfilling `roles.permissions` for existing rows.
- API contract: `backend/openapi/openapi.yaml` — `Permissions` schema gains
  `stats`; `getStatsOverview`/`getAttendanceMatrix`/`getMemberStats` move
  from `x-rbac-module: events` to `stats`; `getStatsPreferences`/
  `listStatsPresets` move from `public` to `stats` (read); `setStatsPreferences`
  moves to `stats` + `x-rbac-self-service: true` (stays read-gated, not
  write); `createStatsPreset`/`updateStatsPreset`/`deleteStatsPreset` move
  to `stats` (write, not self-service); path `/teams/{teamId}/stats/absences`
  and schemas `AttendanceAbsenceTable`/`AttendanceAbsenceRow` are deleted.
  Regenerated `internal/gen/api.gen.go`, `frontend/src/api/types.gen.ts`.
- Backend: `internal/middleware/{authz.go,rbac_table.gen.go}`,
  `cmd/genrbac/main.go` (`validModules`), `internal/teams/{model.go,repository.go}`
  (`PermissionsJSON`, `CreateTeam` role seeding, `MergePermissions`),
  `internal/roles/repository.go` (`getEffectivePermissionsByUserQ`,
  `foldPermissions`, `enforceNoRoleEscalation`), `internal/roles/service.go`
  (`toInternalPermissions`); `internal/stats/{handler.go,service.go,repository.go}`
  lose `GetStatsAbsences`/`GetAbsences`/`AbsenceStats`;
  `internal/server/server.go` loses the `GetStatsAbsences` delegation.
- Frontend: `types/index.ts` (`ModuleKey`, drop `AttendanceAbsenceTable`/`Row`),
  `services/index.ts` (`MODULE_LABELS`), `context/urlState.ts`
  (`ROUTE_MODULE.stats`), `pages/Stats.tsx` (drop the absences tab, gate
  preset create/rename/delete on `stats:write`), `pages/hooks/useStatsQueries.ts`
  (drop `useAbsenceTableQuery`), `query/keys.ts`, `services/serviceLayerReal.ts`,
  `api/map.ts`, `features/team/components/RoleSheets.tsx` (automatic via
  `MODULE_LABELS`), `mocks/{db.ts,handlers.ts}`, `i18n/{de.ts,en.ts}`.
