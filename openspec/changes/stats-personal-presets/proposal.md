## Why

The statistics page's date-range selection (`state.statsRange` in
`AppContext.tsx`) is pure in-memory React state: it is not written to
`localStorage`, not URL-synced (unlike `eventScope`/`eventsView`/`finTab`),
and resets to `null` (→ the last-3-months default) on every reload. A user
who regularly checks, say, the last 6 months has to reselect it every
visit. There is also no way to save a named, reusable custom range (e.g.
"Saison 2026/27") — only the fixed presets (all/3m/6m/12m) and one ad-hoc
custom from/to pair that isn't remembered.

## What Changes

- **Last selection persisted server-side, per user per team.** Whatever
  range (a fixed preset or a custom from/to) the user last selected on the
  stats page is saved and restored automatically on their next visit —
  mirroring how `internal/push`'s `push_preferences` already persists a
  single-row-per-(team,user) preference.
- **Named custom presets, created/renamed/deleted by the user.** A user can
  save a from/to range under a name (e.g. "Saison 2026/27") and it appears
  as a selectable chip alongside the fixed presets, on every future visit,
  until they delete it. Presets are private to the user who created them
  (not shared team-wide) — a Non-Goal noted in design.md.
- **Fixes a gap the presets feature depends on**: `GET
  /teams/{teamId}/stats/members/{userId}` (the single-member/"personal"
  statistics endpoint) currently has no `from`/`to` query parameters at
  all — the handler always calls the service with `nil, nil`, silently
  ignoring any range. Without fixing this, a saved custom range would
  visibly not apply to the one view most literally described as
  "personal."

## Capabilities

### New Capabilities
- `stats-view-preferences`: per-user, per-team persistence of the last
  selected statistics date range, plus a private collection of named custom
  date-range presets the user can create, rename, and delete.

### Modified Capabilities
- `attendance-statistics`: the single-member statistics endpoint gains
  `from`/`to` query parameters, matching the other three statistics
  endpoints that already support them.

## Impact

- Database: new migration `backend/internal/db/migrations/00030_stats_view_preferences.sql`
  (adds `stats_last_selection` — single row per `(team_id, user_id)`, and
  `stats_view_presets` — one-to-many per `(team_id, user_id)`).
- API contract: `backend/openapi/openapi.yaml` — new `GET`/`PUT
  /teams/{teamId}/stats-preferences` (last selection), new `GET`/`POST`/
  `PATCH`/`DELETE /teams/{teamId}/stats-presets[/{presetId}]`; `GET
  /teams/{teamId}/stats/members/{userId}` gains `from`/`to` query params.
  Regenerated `internal/gen/api.gen.go`, `frontend/src/api/types.gen.ts`.
- Backend: new package `internal/statsprefs/{model.go,repository.go,service.go,handler.go}`
  (mirrors `internal/push`), `internal/stats/handler.go`,
  `internal/server/server.go`.
- Frontend: `pages/Stats.tsx`, `pages/hooks/useStatsQueries.ts`, new
  `useStatsPreferencesQuery.ts`/`useStatsPreferencesActions.ts`
  (mirroring `usePushPreferencesQuery.ts`/`usePushPreferencesActions.ts`),
  `context/AppContext.tsx`, `query/keys.ts`, `api/map.ts`,
  `services/serviceLayerReal.ts`, `mocks/{db.ts,handlers.ts}`,
  `i18n/{de.ts,en.ts}`.
