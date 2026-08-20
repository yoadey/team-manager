## Why

A planned absence (`internal/absences`) currently always makes its covered
event dates count as "no" in attendance statistics
(`attendance.AbsenceCoversExpr`/`EffectiveStatusExpr`) — including for
supervisors/coaches (e.g. Lara, Ben, Kenny, Micha) whose time away is often
for club-internal reasons (a course, a Vorstand duty, covering another
team) that shouldn't read as a personal attendance gap in their own
statistics. Today there is no way to mark an absence as not
statistics-relevant, and — more fundamentally — absences are pure
self-service (`CreateAbsence` hard-enforces `req.Body.UserId == user.Id`;
there is no path today for one member to touch another's absence at all).

## What Changes

- **New `notRelevantForStats` flag on an absence.** When set, the absence
  still exists and is still shown (it's not hidden from anyone — absences
  have no RBAC gating beyond membership), but the event dates it covers are
  excluded entirely from that member's attendance statistics: neither
  counted as "no" (today's default) nor counted at all, as opposed to a
  normal absence which counts as "no".
- **Settable by the absence's own owner, or by a privileged caller on
  someone else's absence.** A member can flag their own absence at any
  time (parity with today's full self-service). Flagging *another*
  member's absence additionally requires `events:write` on the team —
  this is the first capability that lets one member touch another's
  absence, so it is deliberately narrow: a dedicated endpoint that can only
  flip this one boolean, not edit dates or reason.
- **New narrow endpoint**, not an extension of `UpdateAbsence`:
  `PATCH /teams/{teamId}/absences/{absenceId}/stats-relevance`.

## Capabilities

### New Capabilities
- `absence-stats-relevance`: a per-absence flag, settable by the absence's
  owner unconditionally or by an `events:write` holder on any team
  member's absence, that excludes the absence's covered event dates from
  that member's attendance statistics entirely (rather than counting them
  as "no").

## Impact

- Database: new migration `backend/internal/db/migrations/00028_absence_stats_relevance.sql`
  (adds `absences.not_relevant_for_stats boolean NOT NULL DEFAULT false` and
  `absences.not_relevant_set_by uuid REFERENCES users(id)`).
- API contract: `backend/openapi/openapi.yaml` — `Absence` gains
  `notRelevantForStats`; new path
  `/teams/{teamId}/absences/{absenceId}/stats-relevance` (`PATCH`,
  `x-rbac-module: public` at the route-classification level, with the
  other-member case additionally checked in application code — see
  design.md). Regenerated `internal/gen/api.gen.go`,
  `frontend/src/api/types.gen.ts`.
- Backend: `internal/absences/{model.go,repository.go,service.go,handler.go}`,
  `internal/attendance/sql.go`, `internal/stats/repository.go`,
  `internal/server/server.go`.
- Frontend: `features/events/components/EventAbsences.tsx`, a new mutation
  hook alongside `features/events/hooks/useAbsence{Queries,Mutations,Actions}.ts`,
  `api/map.ts`, `services/serviceLayerReal.ts`, `mocks/{db.ts,handlers.ts}`,
  `i18n/{de.ts,en.ts}`.
