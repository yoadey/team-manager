## Why

Three usability gaps reported by a team using the app day to day:

1. **The calendar wastes most of a desktop screen.**
   `frontend/src/features/events/components/EventCalendar.tsx` caps the month
   grid at `maxWidth: 820px` with fixed-height day cells (`minHeight: 76px`),
   so on a large monitor the calendar sits in the top-left corner while each
   day still truncates to three entries plus a "+n" overflow marker.
2. **The per-event line-up in the statistics only says "complete / too few"
   for the team as a whole.** `frontend/src/pages/Stats.tsx`'s quota tab ends
   with a "line-up per event" list driven by `EventStat.enough` (a hard-coded
   `pct >= 0.5` in `backend/internal/stats/service.go`). A coach cannot see
   *which* group was short — e.g. all trainers present but only half the
   players — and cannot tune what counts as "too few". The list is also
   buried below the (potentially long) per-person quota list.
3. **The matrix's attendance count is the last column.** With a season's
   worth of events the count is off-screen to the right of a horizontally
   scrolling table, far from the member name it belongs to.

## What Changes

- **Calendar fills the available space.** The calendar view drops its width
  cap and stretches its week rows to the height the viewport leaves free;
  each day cell shows as many entries as its measured height has room for
  (never fewer than today's limits). Falls back to today's layout when the
  viewport is too small to stretch.
- **New "Termine" statistics tab** next to "Quote" and "Matrix", holding the
  per-event line-up that used to sit at the bottom of the quota tab.
- **Per-group breakdown per event.** Each event row shows one indicator per
  group — a group being a team *role* (Mitglied, Trainer, …): a check mark
  when every counted member of that group attended, otherwise the group's
  attendance percentage, rendered red when it is below that group's
  threshold. `GET /teams/{teamId}/stats` gains `EventStat.groups`.
- **Per-group threshold, configurable in the team settings**, either as a
  percentage (red below N %) or as an absolute head count (red below N
  attendees). New table `role_stats_thresholds`, new endpoints
  `GET`/`PUT /teams/{teamId}/stats/group-thresholds` (`x-rbac-module:
  settings`). Default for a group nobody configured: 50 %, i.e. exactly the
  cut-off the old "too few" chip used.
- **Matrix count column moves** from the far right to directly after the
  member name.
- `EventStat.enough` stays on the wire unchanged (no breaking API change);
  the frontend simply stops rendering the team-wide chip it fed.

## Capabilities

### New Capabilities
- `stats-group-thresholds`: per-role attendance threshold (percent or
  absolute count) that decides when a group's per-event attendance is
  highlighted as too low; storage, defaults, validation, authorization and
  the team-settings UI to edit it.
- `calendar-layout`: how the month calendar sizes itself to the viewport.

### Modified Capabilities
- `attendance-statistics`: adds the per-event per-group breakdown, the
  dedicated "Termine" tab, and the position of the matrix's attendance
  count column.

## Impact

- **API contract**: `backend/openapi/openapi.yaml` (`EventStat.groups`,
  `EventGroupStat`, `StatsGroupThreshold*`, two new operations) →
  regenerated `backend/internal/gen/api.gen.go`,
  `backend/internal/middleware/rbac_table.gen.go`,
  `frontend/src/api/types.gen.ts`.
- **Database**: new migration
  `backend/internal/db/migrations/00037_role_stats_thresholds.sql` →
  regenerated `backend/internal/db/gen/models.go` (sqlc).
- **Backend**: `backend/internal/stats/{model,repository,service,handler}.go`
  (+ tests), `backend/internal/server/server.go` (two delegations).
- **Frontend**: `frontend/src/pages/Stats.tsx` (+ test),
  `frontend/src/features/events/components/EventCalendar.tsx` (+ test),
  `frontend/src/features/events/EventsPage.tsx`,
  `frontend/src/features/team/components/{TeamSheets.tsx,teamSettingsSchema.ts}`
  (+ test), `frontend/src/features/team/hooks/useTeamActions.ts` (+ test),
  `frontend/src/services/{apiContract.ts,serviceLayerReal.ts}`,
  `frontend/src/api/map.ts`, `frontend/src/types/index.ts`,
  `frontend/src/mocks/{db.ts,handlers.ts}`, `frontend/src/i18n/{de,en}.ts`.
- **Docs**: `docs/end-user/statistik.md`, `docs/end-user/termine.md`,
  `CLAUDE.md` (RBAC note for the new settings-gated stats route).
- No new runtime dependency on either side.
