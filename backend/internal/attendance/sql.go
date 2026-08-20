// Package attendance holds SQL expressions shared across modules that must
// agree on how a member's *effective* attendance for an event is derived, so
// the event summary (internal/events) and the statistics module
// (internal/stats) cannot drift apart. It is intentionally tiny and
// dependency-free: it exports only SQL snippets, not query logic.
//
// The snippets assume the consuming query exposes these table aliases:
//   - m : memberships   (m.user_id, m.team_id)
//   - e : events        (e.date, e.end_date, e.response_mode)
//   - a : attendance    (a.status)
package attendance

// AbsenceCoversExpr is a correlated EXISTS check (not a JOIN) for whether m's
// planned absence overlaps e's full span -- e.date through
// COALESCE(e.end_date, e.date) inclusive, so a single-day event (end_date
// NULL) still only matches its one date, while a multi-day event is covered
// by any absence overlapping any part of the span, not only one that covers
// the start date. EXISTS rather than a LEFT JOIN deliberately avoids fanning
// out a member's row if more than one absence entry happened to cover the
// same date -- the absences package enforces non-overlap at the application
// layer (advisory-locked check before insert/update, not a DB constraint),
// so this is a defensive guard against double-counting from corrupted or
// pre-constraint historical data, not an expected case.
const AbsenceCoversExpr = `
	EXISTS (
		SELECT 1 FROM absences ab
		WHERE ab.user_id = m.user_id AND ab.team_id = m.team_id
		  AND ab.from_date <= COALESCE(e.end_date, e.date) AND ab.to_date >= e.date
	)
`

// EffectiveStatusExpr resolves each roster row's effective attendance status in
// SQL, mirroring the precedence used everywhere attendance is summarized: an
// explicit attendance record wins; otherwise a covering planned absence
// defaults to "no"; otherwise an opt_out event defaults to "yes"; otherwise
// "pending". Shared by internal/events (event summary) and internal/stats
// (attendance quotes) so the two can never diverge.
//
// Deliberately unaware of absences.not_relevant_for_stats: a member covered
// by a "not relevant" absence still shows as "no" on the event's own
// attendance summary (internal/events) -- operationally they are, in fact,
// not attending that specific event; only internal/stats' season-long
// aggregation should drop the date entirely. See NotRelevantAbsenceCoversExpr.
const EffectiveStatusExpr = `
	CASE
		WHEN a.status IS NOT NULL THEN a.status
		WHEN ` + AbsenceCoversExpr + ` THEN 'no'
		WHEN e.response_mode = 'opt_out' THEN 'yes'
		ELSE 'pending'
	END
`

// NotRelevantAbsenceCoversExpr is a correlated EXISTS check for whether m's
// planned absence covers e's date AND has been flagged
// not_relevant_for_stats. Consumed only by internal/stats, layered on top of
// EffectiveStatusExpr to recognize a stats-only "excluded" outcome (skip the
// date entirely -- neither attending nor absent) without changing
// EffectiveStatusExpr's own "no" default that internal/events' attendance
// summary still relies on. Only meaningful when there is no explicit
// attendance record for the (member, event) pair -- an explicit response
// always wins, exactly as EffectiveStatusExpr's own precedence already
// establishes.
const NotRelevantAbsenceCoversExpr = `
	EXISTS (
		SELECT 1 FROM absences ab
		WHERE ab.user_id = m.user_id AND ab.team_id = m.team_id
		  AND ab.from_date <= COALESCE(e.end_date, e.date) AND ab.to_date >= e.date
		  AND ab.not_relevant_for_stats = true
	)
`
