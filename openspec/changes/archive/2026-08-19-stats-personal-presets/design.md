## Context

`internal/push` is the only existing precedent for a per-(team,user)
preference: `push_preferences`, `PRIMARY KEY (team_id, user_id)`, typed
columns, `GetPreferences`/`UpsertPreferences` doing a plain upsert
(`backend/internal/push/repository.go:134-185`), exposed as `GET`/`PUT
/teams/{teamId}/push-preferences` under `x-rbac-module: public` ("a member
manages only their own preferences within a team they already belong to").
The "last selection" half of this change is a direct copy of that shape.
The "named custom presets" half has no precedent in this codebase — it's
the first one-to-many per-user collection (id, name, plus the range) with
its own CRUD surface, closer in shape to how `contributions`/`penalties`
get their own create/list/delete endpoints than to any preferences pattern.

On the frontend, `state.statsRange: DateRange | null` in `AppContext.tsx`
is deliberately excluded from `urlState.ts`'s URL sync today (unlike
`eventScope`/`eventsView`/`finTab`) — that exclusion stays; persistence
moves to the backend instead of the URL.

## Goals / Non-Goals

**Goals:**
- Reloading the stats page restores the exact range the user had selected
  last time (fixed preset or custom from/to), with no action needed.
- A user can name and save an arbitrary from/to range, see it as a chip
  next to the fixed presets on every future visit, and edit/delete it.
- The single-member statistics endpoint respects `from`/`to` like the
  other three, so a saved range actually changes what "your personal
  statistics" shows.

**Non-Goals:**
- No team-shared presets. "Saison 2026/27" saved by one person is not
  visible to teammates. A shared/team-wide preset library is a plausible
  follow-up (it would need a `scope` column and `settings`- or
  `events`-write gating on the shared write path) but is out of scope here
  — every preset in this change is single-user-private, matching how the
  request was framed ("persönliche Statistik").
- No change to the fixed presets (all/3m/6m/12m) — they remain
  client-computed constants, unaffected by this change.
- No retroactive migration of anyone's currently-selected in-memory range
  into a saved preference — the first persisted value is whatever the user
  next selects after this ships.

## Decisions

**Two tables, not one.** `stats_last_selection` (single row per
(team,user), holds the *current* selection — either raw `from_date`/
`to_date`, or a `preset_id` pointer when the current selection is a saved
preset) and `stats_view_presets` (the named, reusable collection). Folding
both into one table would conflate "what am I looking at right now" with
"what have I saved for later," which have different cardinalities (1 vs.
N) and different lifecycles (selection changes on every click; presets
persist until explicitly deleted).

**`stats_last_selection.preset_id` is a nullable FK, `ON DELETE SET
NULL`.** Deleting a preset that happens to be the currently-active
selection degrades gracefully to its last-known raw dates rather than
erroring or silently reverting to the 3-month default — the user's screen
doesn't visibly jump just because they cleaned up an old preset.

**Presets belong to `(team_id, user_id)`, matching `push_preferences`'s
scoping**, not to `user_id` alone — a preset made while looking at one
team's statistics doesn't clutter another team's preset list, consistent
with statistics themselves being team-scoped.

**`GetMemberStats` gets `from`/`to` query params added to its OpenAPI
operation, following the exact shape already used by
`GetStatsOverview`/`GetAttendanceMatrix`/`GetAbsenceStats`.** The service
method (`stats.Service.GetMemberStats(ctx, teamID, userID, from, to
*openapi_types.Date)`) already accepts `from`/`to` — only the OpenAPI
operation and the handler's `nil, nil` call site need to change
(`internal/stats/handler.go`'s `GetMemberStats`). No repository change is
needed since `SingleMemberStats` already takes `from, to string`.

**A resource cap on presets per (team,user)**, mirroring
`maxContributionsPerTeam`'s role of bounding an otherwise-unbounded
per-scope collection — pick a generous number (e.g. 20) so it's a safety
rail, not a real constraint on the feature's intended use.

## Risks

- **New one-to-many collection with no precedent**: more design surface
  than the "last selection" half. Keep the CRUD endpoints minimal (no
  reordering, no sharing) to avoid scope creep beyond what was requested.
- **`GetMemberStats` OpenAPI change touches generated code**
  (`internal/gen/api.gen.go`) — requires `make generate` in the same
  change, and its handler signature change should be reviewed for any
  other caller of `Service.GetMemberStats` that assumed `nil, nil`.
- **Known limitation, not fixed here (flagged in review)**: rescheduling a
  preset's dates via `PATCH .../stats-presets/{id}` does not propagate to
  a `stats_last_selection` row that has it as the active `presetId` —
  `Service.UpdatePreset` only touches `stats_view_presets`, so that
  selection's raw `from_date`/`to_date` would go stale relative to the
  preset's new dates. Unreachable today: the shipped UI's "rename" form
  (`Stats.tsx`) only ever PATCHes `name`, never `from`/`to`, so no user
  action can currently trigger this. If a preset date-editing UI is added
  later, `UpdatePreset` needs to also update any `stats_last_selection`
  row currently pointing at that preset.
