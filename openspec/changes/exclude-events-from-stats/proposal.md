## Why

Some events on a team's calendar — most concretely a "GL-Training" (a
supplementary/optional training some clubs run alongside the regular
schedule) — should exist on the calendar and support normal RSVP/attendance
like any other event, but must not count toward attendance statistics at
all: not in a member's personal quote, not in the event-list/matrix views,
not in the team overview. Today `backend/internal/stats/repository.go`'s
five queries (`MemberStats`, `EventStats`, `AbsenceStats`,
`SingleMemberStats`, the attendance matrix) all filter events solely on
`e.team_id = $1 AND e.date BETWEEN $2 AND $3 AND e.status = 'active'` — there
is no way, today, to keep an event on the calendar while keeping it out of
statistics.

## What Changes

- **New per-event `excludeFromStats` flag.** Settable in the event
  create/edit form, alongside the existing type picker. Defaults to `false`
  (unchanged behavior for every existing event).
- **Series-aware.** A recurring series' template carries the same flag;
  creating a series seeds it onto every generated occurrence, exactly as
  `cancelLeadMinutes` already is today. Editing an existing series with
  scope "series" updates every occurrence of that series (no date
  filtering — mirroring exactly how `cancelLeadMinutes` and every other
  series-wide-editable field already behave), matching the existing
  single-vs-series edit scope choice (`SeriesEditSubmit`). Editing with
  scope "single" changes only that occurrence, so an individual exception —
  one GL-Training that *should* count, or one regular training that
  exceptionally shouldn't — is always possible without touching the
  series.
- **Stats queries exclude flagged events entirely.** All five
  `stats/repository.go` queries gain `AND e.exclude_from_stats = false`. An
  excluded event contributes to no quote, no matrix cell, no event-level
  stats row — it remains a completely normal event everywhere else (event
  list, detail, RSVP, comments, notifications).

## Capabilities

### New Capabilities
- `event-stats-exclusion`: a per-event (and per-series-template, with
  per-occurrence override) flag that removes an event from all attendance
  statistics computations (overview, event stats, absence stats,
  single-member stats, attendance matrix) while leaving it otherwise
  unchanged.

## Impact

- Database: new migration `backend/internal/db/migrations/00027_event_stats_exclusion.sql`
  (adds `events.exclude_from_stats` and `event_series.exclude_from_stats`,
  both `boolean NOT NULL DEFAULT false`).
- API contract: `backend/openapi/openapi.yaml` — `TeamEvent`,
  `CreateEventRequest`, `UpdateEventRequest`, and the series-template fields
  gain `excludeFromStats`. Regenerated `internal/gen/api.gen.go`,
  `frontend/src/api/types.gen.ts`.
- Backend: `internal/events/{model.go,repository.go}`,
  `internal/stats/repository.go`.
- Frontend: `features/events/components/{EventFormSheet.tsx,eventFormSchema.ts}`,
  `api/map.ts`, `services/serviceLayerReal.ts`, `mocks/{db.ts,handlers.ts}`,
  `i18n/{de.ts,en.ts}`.
