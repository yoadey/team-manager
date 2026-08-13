## 1. Database
- [x] 1.1 `00030_stats_view_preferences.sql` + `00031_stats_view_presets_index.sql`:
      split into two migrations vs. the single one sketched below, to satisfy
      CI's migration-safety lint (unconditionally flags a bare
      `CREATE INDEX` and an `ADD CONSTRAINT ... FOREIGN KEY` without
      `NOT VALID`, even on a table created in the same migration): 00030
      creates both tables with the `preset_id` FK declared inline on
      `stats_last_selection` (`stats_view_presets` created first so the
      inline reference resolves), 00031 adds the lookup index via
      `CREATE INDEX CONCURRENTLY` under `-- +goose NO TRANSACTION`.
      ```sql
      CREATE TABLE stats_last_selection (
        team_id uuid NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
        user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
        from_date date, to_date date, preset_id uuid,
        updated_at timestamptz NOT NULL DEFAULT now(),
        PRIMARY KEY (team_id, user_id)
      );
      CREATE TABLE stats_view_presets (
        id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
        team_id uuid NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
        user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
        name text NOT NULL, from_date date NOT NULL, to_date date NOT NULL,
        created_at timestamptz NOT NULL DEFAULT now()
      );
      ALTER TABLE stats_last_selection ADD CONSTRAINT stats_last_selection_preset_id_fkey
        FOREIGN KEY (preset_id) REFERENCES stats_view_presets(id) ON DELETE SET NULL;
      ```
- [x] 1.2 `make migrate` locally if Docker is available; otherwise rely on CI's
      `backend-migration-rollback`/`backend-migration-safety` gates (no local
      Docker in this environment; relying on CI)

## 2. OpenAPI
- [x] 2.1 `GET /teams/{teamId}/stats/members/{userId}`: add `from`/`to` query
      params (`type: string, format: date`), matching the other three stats
      endpoints
- [x] 2.2 `GET`/`PUT /teams/{teamId}/stats-preferences`: last-selection
      resource (`fromDate`, `toDate`, `presetId` nullable), `x-rbac-module: public`
- [x] 2.3 `GET`/`POST /teams/{teamId}/stats-presets`,
      `PATCH`/`DELETE /teams/{teamId}/stats-presets/{presetId}`:
      `{ id, name, fromDate, toDate }`, `x-rbac-module: public`
- [x] 2.4 `cd backend && make generate` (commit `internal/gen/api.gen.go`)
- [x] 2.5 repo-root `make generate-ts` (commit `frontend/src/api/types.gen.ts`)

## 3. Backend: stats module (GetMemberStats fix)
- [x] 3.1 `internal/stats/handler.go`: `GetMemberStats` reads `req.Params.From`/
      `req.Params.To` instead of passing `nil, nil`

## 4. Backend: new statsprefs package
- [x] 4.1 `internal/statsprefs/model.go`: `LastSelection` (`FromDate`,
      `ToDate` `*time.Time`, `PresetID *uuid.UUID`), `Preset` (`ID`, `Name`,
      `FromDate`, `ToDate`); `maxPresetsPerTeamUser = 20`,
      `ErrTooManyPresets`
- [x] 4.2 `repository.go`: `GetLastSelection`/`UpsertLastSelection`
      (mirrors `push.Repository.GetPreferences`/`UpsertPreferences`),
      `ListPresets`/`CreatePreset`/`UpdatePreset`/`DeletePreset`
      (team+user scoped)
- [x] 4.3 `service.go`: thin pass-through + `maxPresetsPerTeamUser` check on
      create
- [x] 4.4 `handler.go` + `internal/server/server.go`: wire the 6 new routes

## 5. Backend: tests
- [x] 5.1 `stats/handler_test.go`: `GetMemberStats` honors `from`/`to`
- [x] 5.2 `statsprefs/repository_test.go`: upsert last-selection idempotent;
      preset CRUD; deleting a preset that is the active selection's
      `preset_id` sets it to NULL (`ON DELETE SET NULL`) without deleting
      the row
- [x] 5.3 `statsprefs/service_test.go`: `maxPresetsPerTeamUser` enforced
      (plus `statsprefs/handler_test.go` for the HTTP-layer mapping)

## 6. Frontend
- [x] 6.1 `query/keys.ts`: `queryKeys.statsPreferences(teamId)`,
      `queryKeys.statsPresets(teamId)`
- [x] 6.2 New `pages/hooks/useStatsPreferencesQuery.ts` +
      `useStatsPreferencesActions.ts` mirroring
      `usePushPreferencesQuery.ts`/`usePushPreferencesActions.ts`
      (last-selection GET/PUT, presets list/create/update/delete)
- [x] 6.3 `pages/Stats.tsx`: on mount, hydrate `statsRange` from the fetched
      last-selection instead of defaulting to `null`; extend the preset chip
      row with fetched named presets; "+ Neu" opens a small name+date-range
      form (existing native `<input type="date">`/`TextInput` pattern, no
      new date-picker dependency); edit/delete affordances per custom chip;
      every selection change also fires the last-selection PUT
- [x] 6.4 `context/AppContext.tsx`: no structural change to `statsRange`
      itself, only how it's initialized (see 6.3) — `urlState.ts` stays
      unchanged (still not URL-synced); ended up not needing any
      `AppContext.tsx` edit at all, since hydration lives entirely in
      `Stats.tsx`'s own effect
- [x] 6.5 `api/map.ts`, `services/serviceLayerReal.ts`,
      `mocks/{db.ts,handlers.ts}`: wire the 6 new endpoints + `GetMemberStats`
      from/to through; also threads the active `statsRange` through
      `api.stats.attendanceFor` (member-profile personal stats) so that view
      honors the saved range too, per design.md's Goals
- [x] 6.6 `i18n/{de.ts,en.ts}`: labels for "+ Neu", preset name field,
      edit/delete actions

## 7. Verification
- [x] 7.1 `cd backend && make test && make lint`
- [x] 7.2 `cd frontend && npm run typecheck && npm run lint && npm test`
- [x] 7.3 `make generate && make generate-ts` produce no drift
- [ ] 7.4 Manual: select a 6-month range, reload the page, confirm it's
      restored; create a "Saison 2026/27" preset, reload, select it from the
      chip row, confirm the single-member/personal view also reflects it;
      delete the preset while it's active, confirm the page doesn't error
      (not run — no browser/backend available in this environment; covered
      instead by `serviceContract.test.ts`'s MSW-backed round-trip tests)
