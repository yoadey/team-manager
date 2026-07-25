## Context

The stats overview (`GET /teams/{teamId}/stats`) returns aggregates only. The raw per-member-per-event attendance already exists in the `attendance` table and is resolved to an *effective* status by `attendance.EffectiveStatusExpr` (explicit response → covering absence = no → opt_out default = yes → else pending), shared by events and stats. The matrix is a projection of that same expression, not a new definition of "present".

## Goals

- One backend round-trip for the whole grid (no 1+N per-event fetches).
- Cell status identical to what the overview/event summary would show for the same member+event.
- Consistent snapshot: columns and cells read inside one read-only transaction, like `GetOverview`.

## Decisions

- **New endpoint, not an extension of `StatsOverview`.** The matrix is only needed when the user opens the Matrix tab; bundling it into the always-loaded overview would inflate every stats load with member×event data. A separate endpoint keeps the overview lean and lets the tab lazy-load.
- **Four cell states, folded from the effective status.** `yes | no | maybe | pending`. `pending` is the "unbekannt" (grey –) bucket — a member with no explicit response and no opt_out/absence default. The frontend renders ✓/?/✗/– accordingly. Nomination is deliberately *not* a cell state: the matrix is roster-driven (every current member appears), matching how the quotes are computed.
- **Cells as a map keyed by event id**, not an index-aligned array. The frontend filters columns by event type client-side; a keyed lookup stays correct under filtering without re-indexing every row. Absent key ⇒ member had no row for that event (treated as `pending` by the client) — but the backend emits an explicit entry for every (member, event) pair in range, so this is only a defensive default.
- **Sort rows by `yes` desc, then name** — same ordering as `MemberStats`, so the matrix and the overview's per-person bars agree on who is "most present".
- **Columns sorted by date asc** (then id for stability) — reads left-to-right chronologically, matching the user's mental model (1.1., 6.1., 7.1., …).
- **Reuse the overview's date-range handling** (`defaultDateRange`, 3-month default, `maxStatsRangeDays` clamp). A wide range bounds the grid size the same way it bounds the aggregation.
- **Single repository method `AttendanceMatrix` opening its own read tx**, returning `([]MatrixColumnRow, []MatrixCellRow)`. Keeps the `statsRepo` interface (and its unit-test mock) to one new function rather than a new reader interface + tx wrapper.

## Risks

- **Grid size.** members × events within the clamped range. For a large club over 2 years this is bounded but non-trivial; acceptable for a lazy-loaded tab, and the `maxStatsRangeDays` clamp caps the worst case. If it ever needs paging, the keyed-cell shape allows column windowing without a contract change.
- **Cell/aggregate drift.** Mitigated by computing both from `attendance.EffectiveStatusExpr`; a test asserts a member's matrix `yes` count equals their overview `yes`.
