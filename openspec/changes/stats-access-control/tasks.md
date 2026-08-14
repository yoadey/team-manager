## 1. OpenAPI
- [x] 1.1 `Permissions` schema: add `stats` property (`PermLevel`), add to
      `required`
- [x] 1.2 `getStatsOverview`/`getAttendanceMatrix`/`getMemberStats`:
      `x-rbac-module: events` → `stats`
- [x] 1.3 `getStatsPreferences`, `listStatsPresets`: `x-rbac-module:
      public` → `stats`
- [x] 1.4 `setStatsPreferences`: `x-rbac-module: public` → `stats` +
      `x-rbac-self-service: true`
- [x] 1.5 `createStatsPreset`/`updateStatsPreset`/`deleteStatsPreset`:
      `x-rbac-module: public` → `stats` (no self-service flag)
- [x] 1.6 Delete path `/teams/{teamId}/stats/absences` (`getStatsAbsences`)
      and schemas `AttendanceAbsenceTable`, `AttendanceAbsenceRow`
- [x] 1.7 `cd backend && make generate` (commit `internal/gen/api.gen.go`,
      `internal/middleware/rbac_table.gen.go`)
- [x] 1.8 repo-root `make generate-ts` (commit
      `frontend/src/api/types.gen.ts`)

## 2. Backend: RBAC module plumbing
- [x] 2.1 `cmd/genrbac/main.go`: add `"stats": true` to `validModules`
- [x] 2.2 `internal/teams/model.go`: `PermissionsJSON` gains `Stats string
      \`json:"stats"\``
- [x] 2.3 `internal/teams/repository.go`: `CreateTeam`'s `adminPerms` gains
      `Stats: "write"`, `memberPerms` gains `Stats: "read"`;
      `MergePermissions`'s `best` init and per-role merge loop gain `Stats`
- [x] 2.4 `internal/roles/repository.go`: `getEffectivePermissionsByUserQ`'s
      `eff` init, `foldPermissions`, and `enforceNoRoleEscalation`'s
      `ceilings`/`granted` slices gain a `Stats` entry
- [x] 2.5 `internal/roles/service.go`: `toInternalPermissions` gains the
      `Stats` mapping from `gen.Permissions.Stats`
- [x] 2.6 `internal/middleware/authz.go`: `hasWritePermission`/
      `hasAnyPermission` gain `case "stats": level = p.Stats`

## 3. Database
- [x] 3.1 `00032_stats_rbac_module.sql`: backfill existing `roles.permissions`
      — `stats: write` for system `Admin` roles, `stats: read` for system
      `Member` roles, `stats: none` for every other role row still missing
      the key (`WHERE NOT (permissions ? 'stats')`, run last so it only
      catches what the first two updates didn't)
- [ ] 3.2 `make migrate` locally if Docker is available; otherwise rely on
      CI's `backend-migration-rollback`/`backend-migration-safety` gates
      (no Docker in this dev sandbox — deferred to CI)

## 4. Backend: remove the absence-table code
- [x] 4.1 `internal/stats/repository.go`: delete `AbsenceStats` and its
      interface entry
- [x] 4.2 `internal/stats/service.go`: delete `GetAbsences` and its
      interface entry
- [x] 4.3 `internal/stats/handler.go`: delete `GetStatsAbsences`
- [x] 4.4 `internal/server/server.go`: delete both `GetStatsAbsences`
      delegation methods (`StrictUnimplemented` and `Server`)
- [x] 4.5 Delete the corresponding cases from
      `internal/stats/{handler_test.go,service_test.go,repository_test.go}`

## 5. Backend: tests
- [x] 5.1 `internal/middleware/authz_test.go`: cover `stats` read/write/none
      gating (mirroring the existing per-module cases), and any
      `PermissionsJSON{...}` literals that need a `Stats` value to exercise
      the new module
- [x] 5.2 `internal/teams/repository_test.go`: `CreateTeam` asserts the
      new team's Admin role has `stats: write` and Member role has
      `stats: read`
- [x] 5.3 `internal/roles/repository_test.go`: role-escalation test covers
      `stats` alongside the existing five modules
- [x] 5.4 Migration test/assertion (or a `stats` package integration test)
      confirming a pre-migration role row without `stats` reads back as
      `none` via the API, and post-migration the system roles read back as
      documented

## 6. Frontend: RBAC module plumbing
- [x] 6.1 `types/index.ts`: `ModuleKey` gains `'stats'`; delete
      `AttendanceAbsenceTable`/`AttendanceAbsenceRow` types
- [x] 6.2 `services/index.ts`: `MODULE_LABELS` gains `stats: 'Statistik'`
      (role editor UI updates automatically via `RoleSheets.tsx`)
- [x] 6.3 `context/urlState.ts`: `ROUTE_MODULE.stats` changes from
      `'events'` to `'stats'`; widen the `Record` value type union
- [x] 6.4 `mocks/db.ts`: `perms()` default gains `stats: 'none'`,
      `MODULES` gains `'stats'`, `defaultRoles()`'s Admin/Trainer role
      gains `stats: 'write'` and the default Member role gains
      `stats: 'read'`

## 7. Frontend: remove the absence-table tab
- [x] 7.1 `pages/Stats.tsx`: remove the `'absences'` tab from `StatsTab`,
      the tab button, `AbsenceTableView`, its loading branch, and the
      `useAbsenceTableQuery` call
- [x] 7.2 `pages/hooks/useStatsQueries.ts`: remove `useAbsenceTableQuery`
- [x] 7.3 `query/keys.ts`: remove `statsAbsences`
- [x] 7.4 `services/serviceLayerReal.ts`, `services/apiContract.ts`:
      remove `stats.absenceTable`
- [x] 7.5 `api/map.ts`: remove the absence-table mapping
- [x] 7.6 `mocks/handlers.ts`: remove the `GET
      /teams/:teamId/stats/absences` handler
- [x] 7.7 `i18n/de.ts`, `i18n/en.ts`: remove `tabAbsences`,
      `absenceTableTitle`, `emptyAbsences` keys
- [x] 7.8 Delete/update the corresponding cases in `pages/Stats.test.tsx`,
      `pages/hooks/useStatsQueries.test.ts`, `services/serviceLayerReal.test.ts`,
      `api/map.test.ts`

## 8. Frontend: gate preset write actions
- [x] 8.1 `pages/Stats.tsx`: disable/hide the "new preset" affordance and
      each saved preset's rename/delete actions unless `app.can('stats',
      'write')`; leave range selection and the automatic last-selection
      save unaffected by the write check
- [x] 8.2 Corresponding `pages/Stats.test.tsx` coverage for a `stats: read`
      user (can view, cannot create/rename/delete a preset) vs. a `stats:
      write` user

## 9. Spec
- [x] 9.1 `openspec/specs/attendance-absence-table/spec.md`: capability
      removed (delta below)
- [x] 9.2 `openspec/specs/attendance-statistics/spec.md`: "Matrix range and
      authorization" now names `stats` instead of `events`

## 10. Verification
- [x] 10.1 `openspec validate stats-access-control --strict`
- [x] 10.2 `cd backend && make test && make lint`
- [x] 10.3 `cd frontend && npm run typecheck && npm run lint && npm test`
- [x] 10.4 `make generate && make generate-ts` produce no drift
- [ ] 10.5 Manual: as a `Member` (default `stats: read`), open the
      statistics page — quota/matrix visible, Fehlzeiten tab gone, existing
      presets visible, "new preset" disabled; as an `Admin` (`stats:
      write`), the same page allows creating/renaming/deleting presets; as
      a role with `stats: none`, the nav item and page are hidden/403
      (no running app/DB in this dev sandbox — deferred to reviewer/CI)
