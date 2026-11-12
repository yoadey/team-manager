## Context

Statistics today expose three shapes, all derived from the shared
effective-attendance expression (`internal/attendance`): per-member quotes,
per-event turnout (`EventStat`: `yes`, `nominated` = counted, `pct`,
`enough = pct >= 0.5`), and the member × event matrix. The frontend renders
the per-event list at the bottom of the quota tab with a team-wide
"Vollständig / Zu wenige" chip.

A team's members hold one or more *roles* (`membership_roles`), e.g. the
seeded system roles "Admin / Trainer" and "Mitglied" plus custom ones
("Spieler", "Teamkapitän"). Roles are the app's only notion of a named
member group that is consistently maintained — `memberships."group"` is an
optional free-text label most teams leave empty.

The calendar is a 7 × 6 CSS grid inside a `Card`, inside a page capped at
820 px, inside the shell's scrolling `<main>`.

## Goals / Non-Goals

**Goals:**
- A coach sees per event, per group, whether the group was complete, and if
  not how well it was attended — with a per-group, coach-tunable "too low"
  cut-off expressed either relatively (percent) or absolutely (head count).
- The per-event list is reachable in one tap as its own tab.
- The matrix's count is readable next to the name.
- The calendar uses the screen it is given.

**Non-Goals:**
- Grouping by the free-text `memberships."group"` label, or by a member's
  single "primary" role.
- Changing how effective attendance is computed, or what `EventStat.yes` /
  `nominated` / `pct` / `enough` mean.
- A team-wide default threshold with per-role overrides (explicitly declined
  in favour of plain per-group values).
- A week/day calendar view, or virtualising the month grid.

## Decisions

### A "group" is a role; a member counts in every role they hold
The user's own examples ("Mitglied, Trainer, Spieler") are role names. A
member holding two roles (the common "Trainer" + "Mitglied" setup) counts in
both groups — the indicator answers "were all trainers there?" and "were all
members there?" independently. *Alternative considered:* primary role only
(disjoint groups) — rejected: which role is "primary" is an ordering
artefact (`ORDER BY r.id`), not something a coach chooses, so a trainer
could silently vanish from the "Trainer" group.

### Group figures use the same roster and status as the event row
`EventGroupStats` reuses `EventStats`' exact shape — `events JOIN
memberships` (deliberately *not* filtered on `m.exclude_from_stats`, per
`member-stats-exclusion`: this is per-event turnout, not a personal quota),
the shared `EffectiveStatusExpr`, the stats-only "excluded" outcome for a
not-relevant absence — and only adds `JOIN membership_roles JOIN roles`. Both
queries run in the overview's existing read-only repeatable-read transaction
(`WithReadTx`), so group figures cannot drift from the event row.

### A missing response counts as not present — in the line-up only
Product decision (the owner's, after the independent review flagged the
alternative): a group is "fully present" only when *everyone expected*
attended. A member with no response at all (effective status `pending`: no
explicit answer, no covering absence, not an opt-out event) was not
confirmed present and counts against completeness and in the percentage's
denominator. Three "yes" and twelve silent members are 3 of 15 (20 %), not a
check mark — and an absolute threshold of "at least 8" is red.

Mechanically, both queries report `pending` *next to* `counted` instead of
folding it in: `EventStat.yes` / `nominated` / `pct` / `enough` and the
per-member quotas keep meaning exactly what they meant (responses only, so
nothing else in the statistics changes), while the line-up — the event's
"x/y anwesend" line and every group indicator — uses `counted + pending` as
"expected". Not expected, and so in neither number, are members who are not
nominated (per member, or because the event nominates only roles they don't
hold), members who without an explicit response are covered by an absence
flagged not relevant for statistics, and — for `pending` — members flagged
`exclude_from_stats` (their explicit responses still count in the turnout,
per `member-stats-exclusion`, but their silence does not count against the
event). A group nobody was expected from shows "no data" (–).

*Role nominations need special handling in the stats queries:* the events
module stores them only as `events.nominated_role_ids`; it writes
`not_nominated` attendance rows solely for per-member nominations, and
`EffectiveStatusExpr` never reads the list. Without an extra condition
(`notNominatedByRoleExpr`), every member of a role the event did not
nominate would come out as `pending` and every non-nominated group would
show "0 %" in red. The condition applies only without an explicit response
and is checked before the opt-out default, so a non-nominated member is
neither assumed to attend nor counted as missing. (The event summary in
`internal/events` has the same blind spot for role nominations; it is
pre-existing and out of scope here — it shows such members as "offen".)

*Alternative considered (and first implemented):* responses only, i.e.
`complete = yes == counted`, for consistency with the quotas — rejected by
the owner: it shows a check mark for a group most of which never answered.

### Thresholds live in their own table keyed by role
`role_stats_thresholds(role_id PK → roles ON DELETE CASCADE, mode, value,
updated_at)`. A missing row means the default (percent, 50). *Alternatives
considered:*
- *Columns on `roles` + fields on the `Role` schema* — rejected: `Role` is
  assembled by hand-written SQL in `teams`, `members`, `events`, `roles`
  (sqlc) and `absences`; every one would need the new columns or `Role`
  payloads would disagree depending on the endpoint.
- *JSONB map on `teams`* — rejected: needs a scrub on role deletion (the
  `reason_visibility_role_ids` problem again); a FK cascade is free.

No `team_id` column: the role already pins the team, and every query
reaches the table through `roles r … WHERE r.team_id = $1`, which is what
keeps the team-scoping invariant (a threshold can only be read or written
through a role of the URL's team). That also means no new index — the PK
covers the only join.

### Semantics of "red"
For a group with somebody expected (`counted + pending > 0`) that is not
complete (`yes < counted + pending`):
- `percent`: red when `yes / (counted + pending) * 100 < value`.
- `count`: red when `yes < value`.

Complete always wins (a two-person trainer group with a threshold of 3 that
is fully present shows the check mark). `value = 0` therefore means "never
red". The comparison uses the exact ratio, not the rounded label; a label
that would round up to 100 % for an incomplete group is capped at 99 % so
"100 %" never appears without the check mark.

Validation (service layer, mirrored by DB CHECKs as a backstop): `mode ∈
{percent, count}`; `value` integer, `0..100` for percent, `0..10000` for
count.

### The resolved threshold travels with each group stat
`EventGroupStat` carries `thresholdMode`/`thresholdValue` (default already
applied). The statistics page needs `stats:read` only; the thresholds
endpoint is settings-gated. Shipping the resolved values in the overview
avoids a second request and avoids making a stats reader depend on a
settings permission.

### Dedicated endpoints, gated by `settings`
`GET`/`PUT /teams/{teamId}/stats/group-thresholds`, `x-rbac-module:
settings` (GET → `settings:read`, PUT → `settings:write`), because the value
is edited in the team settings sheet by whoever administers the team — the
same audience as `reasonVisibilityRoleIds`. `GET` returns one item per team
role (defaults filled in) so the form needs no client-side default logic.
`PUT` takes a list, upserts each item inside one transaction, rejects the
whole request with 422 if any `roleId` is not a role of the team, and leaves
roles not mentioned untouched. Implemented in `internal/stats` (the module
that consumes the values), not `internal/roles`.

### Frontend
- `Stats.tsx`: `StatsTab = 'quota' | 'matrix' | 'events'`; the events list
  moves into an `EventsView` fed by the already-loaded overview query (no
  new request). Group indicators are small pills: role-colour dot + name +
  (check | percent | –); `aria-label` spells out "name: x of y attended".
- Matrix: the count column becomes the second column (see below for
  pinning).
- Team settings sheet: one row per role with a `%`/`Anzahl` toggle and a
  number input, part of the existing react-hook-form + zod form and saved by
  the existing "save" button (PUT only when the section is dirty).
- Calendar: the events page's calendar branch becomes a flex column with
  `minHeight: 100%` of `<main>`'s content box, so the card reaches the
  bottom of the page. The six week rows are *not* flexible tracks: a hook
  (`useStretchedRowHeight`, `ResizeObserver` on `<main>`) computes the
  height each row gets when the grid reaches the bottom of the scroll
  container, from the container's height and the grid's top edge only, and
  the rows are `minmax(<that height>, auto)`. The per-cell entry limits are
  derived from the same number and never drop below today's fixed limits
  (also the fallback where nothing can be measured, e.g. jsdom).
  *Why not `1fr` rows:* the grid's height is indefinite (the page is only
  min-height-constrained so it can still grow and scroll), and `fr` tracks
  sized against an indefinite height all take the size of the tallest
  track -- one busy day inflated all six weeks and pushed the calendar off
  screen (found in independent review, covered by an e2e test).
  *Why not measure the rows themselves:* their height grows with the entries
  shown, so limits derived from it would feed back into themselves.
- Matrix: on desktop the count column is pinned (`position: sticky`) right
  after the pinned, fixed-width name column; on a phone only the name is
  pinned, so the event columns keep enough room.

## Risks / Trade-offs

- [A member in several roles is counted once per role, so group counts do
  not sum to the event's total] → documented in the end-user docs; the
  event row keeps showing the de-duplicated team total.
- [An absolute threshold larger than the group can never be met except by
  being complete] → intended; complete always shows the check mark.
- [Extra aggregate query per overview request] → same join shape and same
  5 s timeout as `EventStats`, bounded by the existing 730-day range clamp
  and `maxRolesPerTeam`.
- [Calendar height measurement can lag one frame behind a resize] → limits
  only ever grow from the safe minimum, and cells keep `overflow: hidden`.
- [A day whose *minimum* entries (the previous fixed limits) need more than
  the stretched row has -- e.g. three events in an 82 px row on a 1440×900
  screen -- makes its week taller, so the page scrolls by that excess] →
  accepted: it is exactly the pre-change behaviour for that week, and the
  alternative (showing fewer entries than before) is worse.
- [Saving team settings and thresholds is two requests, not one
  transaction] → a threshold failure is reported as a partial save, the
  team list is refreshed and the threshold list reloaded.
- [The line-up's "x/y anwesend" (everyone expected) can differ from what
  the quota tab implies (responses only)] → intended, see the decision
  above; the two answer different questions.
- [The demo (MSW) backend's *per-member quotas* still count explicit
  responses only, while the real backend counts effective attendance] →
  pre-existing drift, left alone. The per-event line-up of the demo was
  moved to effective attendance in this change (it has to be: with missing
  responses counting as not present, raw counting would have shown every
  unanswered opt-out training as 0 %), so it now agrees with the demo's own
  matrix and with the real backend.

## Migration Plan

Additive only: one new table (no backfill — absence of a row is the
default), new optional-to-ignore response field, two new routes. Rollback is
`goose down` (drops the table) plus redeploying the previous images; an old
frontend ignores `groups`, an old backend never sees the new routes.
