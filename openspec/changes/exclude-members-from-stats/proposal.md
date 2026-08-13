## Why

A club sometimes wants to keep a person's membership record (roster,
contact info, role) while excluding them entirely from attendance
statistics — e.g. a member on long-term leave, an honorary member who
never trains, or someone whose participation is tracked elsewhere. Today
every `stats/repository.go` query is roster-driven via an unconditional
`FROM memberships m ... WHERE m.team_id = $1` join — there is no way to
keep a person on the team without them appearing in every quote and matrix
row.

## What Changes

- **New `excludeFromStats` flag on a membership**, settable in the
  member's edit form by anyone with `members:write` (the same permission
  that already gates the rest of profile editing).
- **Excluded from personal-quota views, kept in event-level views.** A
  flagged member disappears from `MemberStats` (the overview quota list),
  `SingleMemberStats`, the attendance matrix's rows/columns, and the
  absence table (`AbsenceStats`) — their personal statistics are simply not
  computed. `EventStats` (per-event turnout numbers) is deliberately left
  unfiltered: it answers "how many people were at this training", and
  silently dropping a flagged member's historical yes/no response there
  would understate real turnout for that specific event. The exclusion is
  about *that person's personal quota*, not a rewrite of event-level
  history.

## Capabilities

### New Capabilities
- `member-stats-exclusion`: a per-membership flag, editable via the member
  profile form, that removes a member from personal-quota-oriented
  statistics views (overview, single-member, matrix, absence table) while
  leaving event-level turnout aggregates unaffected.

## Impact

- Database: new migration `backend/internal/db/migrations/00029_member_stats_exclusion.sql`
  (adds `memberships.exclude_from_stats boolean NOT NULL DEFAULT false`).
- API contract: `backend/openapi/openapi.yaml` — `Member` and
  `UpdateMemberRequest` gain `excludeFromStats`. Regenerated
  `internal/gen/api.gen.go`, `frontend/src/api/types.gen.ts`.
- Backend: `internal/members/{model.go,repository.go}`,
  `internal/stats/repository.go`.
- Frontend: `features/members/components/{MemberSheets.tsx,memberFormSchema.ts}`,
  `api/map.ts`, `services/serviceLayerReal.ts`, `mocks/{db.ts,handlers.ts}`,
  `i18n/{de.ts,en.ts}`.
