## Context

`events/repository.go` already derives an effective status (`effectiveStatusExpr`-style CASE: explicit status → overlapping absence → opt_out default → pending) for the event summary. `stats/repository.go` (lines ~76, 114, 151) independently counts explicit rows only. `MemberStats` joins `FROM memberships` (current members); event-level counting `LEFT JOIN attendance` without a membership filter includes ex-members' retained rows.

## Goals / Non-Goals

**Goals:**
- Stats quotes match the effective attendance the event summary shows.
- A single source of truth for the effective-status expression, shared by events and stats.
- Consistent treatment of ex-members between the aggregations shown together.

**Non-Goals:**
- Changing retention of ex-members' attendance rows (kept intentionally).
- Changing the stats date-range clamping (`maxStatsRangeDays`).

## Decisions

- Extract the effective-status CASE into a shared SQL snippet (e.g. an exported const in an `internal/attendance` helper, or reused from `events`) and use it in the stats `FILTER` clauses.
- Decide ex-member handling explicitly: exclude ex-members from **both** event-level and member-level counts (so the numbers reconcile), or document why event-level intentionally includes them. Prefer excluding for consistency within a single stats view.
- Keep opt-out semantics identical to the event summary (opt_out with no explicit response → counts as "yes").

## Risks / Trade-offs

- Effective status requires the absence/opt_out joins in the stats queries, adding cost; bounded by the date-range clamp.
- Extracting the shared expression touches `events` too; keep the event-summary behavior byte-identical (its tests must stay green).
