// Package attendance holds SQL expressions shared across modules that must
// agree on how a member's *effective* attendance for an event is derived, so
// the event summary (internal/events) and the statistics module
// (internal/stats) cannot drift apart. It is intentionally tiny and
// dependency-free: it exports only SQL snippets, not query logic.
//
// The snippets assume the consuming query exposes these table aliases:
//   - m : memberships   (m.user_id, m.team_id)
//   - e : events        (e.date, e.response_mode)
//   - a : attendance    (a.status)
package attendance

// AbsenceCoversExpr is a correlated EXISTS check (not a JOIN) for whether m's
// planned absence covers e's date. EXISTS rather than a LEFT JOIN deliberately
// avoids fanning out a member's row if more than one absence entry happened to
// cover the same date -- the absences package enforces non-overlap at the
// application layer (advisory-locked check before insert/update, not a DB
// constraint), so this is a defensive guard against double-counting from
// corrupted or pre-constraint historical data, not an expected case.
const AbsenceCoversExpr = `
	EXISTS (
		SELECT 1 FROM absences ab
		WHERE ab.user_id = m.user_id AND ab.team_id = m.team_id
		  AND ab.from_date <= e.date AND ab.to_date >= e.date
	)
`

// EffectiveStatusExpr resolves each roster row's effective attendance status in
// SQL, mirroring the precedence used everywhere attendance is summarized: an
// explicit attendance record wins; otherwise a covering planned absence
// defaults to "no"; otherwise an opt_out event defaults to "yes"; otherwise
// "pending". Shared by internal/events (event summary) and internal/stats
// (attendance quotes) so the two can never diverge.
const EffectiveStatusExpr = `
	CASE
		WHEN a.status IS NOT NULL THEN a.status
		WHEN ` + AbsenceCoversExpr + ` THEN 'no'
		WHEN e.response_mode = 'opt_out' THEN 'yes'
		ELSE 'pending'
	END
`
