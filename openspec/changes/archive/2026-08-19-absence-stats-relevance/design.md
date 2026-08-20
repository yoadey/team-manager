## Context

Absences are the one module with no RBAC gating at all
(`x-rbac-module: public`) — `internal/absences/service.go` documents that
any team member can read any other member's absence reason, and
`CreateAbsence`/`UpdateAbsence`/`DeleteAbsence` all hard-enforce
`req.Body.UserId == user.Id` in the handler, so there has never been a path
for one member to write to another's absence. This change introduces
exactly one: flipping a single boolean, on someone else's absence, gated
behind `events:write`.

`internal/notifications/service.go` establishes the precedent for an
in-service (not route-table) permission check: it takes a
`GetPermissions(ctx, teamID, userID) (teams.PermissionsJSON, error)`
dependency and an exported `HasReadAccess(p teams.PermissionsJSON, module
string) bool` that mirrors (deliberately duplicates, per its own comment)
`internal/middleware/authz.go`'s unexported `hasWritePermission`/
`hasAnyPermission` fail-closed-on-unknown-module logic. The absences
service should follow the same shape: depend on `teams`'s permission
lookup, and add a small local helper mirroring `hasWritePermission` for the
`events` module specifically (absences only ever needs this one module
check, so a generic multi-module helper isn't warranted).

## Goals / Non-Goals

**Goals:**
- Any member can always flag their own absence as not stats-relevant —
  zero new friction for the common case (a person managing their own
  entry).
- Flagging someone else's absence requires `events:write`, enforced in the
  service layer since the route stays `x-rbac-module: public` (unchanged
  for the rest of the absences module).
- A flagged absence's covered dates are excluded entirely from that
  member's statistics — not counted as "no", not counted as anything.
- The change is minimal-blast-radius: a dedicated endpoint that can only
  toggle this one field, never dates or reason, on anyone's absence.

**Non-Goals:**
- No general "edit anyone's absence" capability — `UpdateAbsence` stays
  self-only. This is intentionally not a stepping stone toward broader
  absence-editing permissions; it is scoped to the one field.
- No notification/audit-log entry for who flagged whose absence beyond the
  `not_relevant_set_by` column (queryable, not surfaced as a UI feature in
  this change).

## Decisions

**Route stays `x-rbac-module: public`; the other-member check is inline in
the service.** The generated RBAC table operates at method+path
granularity and cannot express "self: no permission needed, other: needs
write" for the same route — that distinction depends on the request body
(whose absence is targeted) and the resource's actual owner, which the
generated middleware never inspects. `AbsencesService.SetStatsRelevance`
therefore does the ownership comparison itself (`absence.UserID ==
caller.ID`) and, only when they differ, calls the local write-permission
helper before proceeding — exactly the shape `notifications.Service.List`
already uses for its own per-item, non-route-table permission filtering.

**`EffectiveStatusExpr` itself is left unchanged — a new
`NotRelevantAbsenceCoversExpr` is layered on top, only inside
`stats/repository.go`.** `EffectiveStatusExpr` is shared with
`internal/events`' own event-attendance-summary queries
(`GetAttendanceSummary`/`GetAttendanceSummaries`), which bucket every roster
row into exactly one of `yes/no/maybe/pending/not_nominated` and sum those
buckets to `Total` — introducing a 6th value there would silently make
`Total` stop reconciling with the sum of its buckets, and semantically an
event's own attendance summary should keep showing such a member as "no"
(operationally, they are not attending that specific event; only their
season-long statistics should drop the date). So instead, each of the four
`stats/repository.go` queries wraps `EffectiveStatusExpr` in its own CASE:
`WHEN a.status IS NULL AND NotRelevantAbsenceCoversExpr THEN 'excluded'
ELSE EffectiveStatusExpr`. `'excluded'` never leaves SQL as a wire value in
these four queries — it's consumed only by `COUNT(*) FILTER (WHERE eff IN
(...))` / `WHERE eff = 'no'`, so it's safely a new internal label.

**The attendance-matrix cell is the one place `eff` IS wire-exposed
(`MatrixCellRow.Eff` is cast directly to `gen.AttendanceStatus`, whose enum
is `yes/no/maybe/pending/not_nominated` with no "excluded" member).**
Extending that enum would cascade into the OpenAPI contract, frontend
types, i18n labels, and matrix cell rendering for one narrow case. Instead,
`matrixCells`' own CASE maps a not-relevant-absence-covered date to
`'pending'` (an existing, already-unaffiliated-with-any-response value)
rather than introducing `'excluded'` on the wire. This is a semantic
approximation ("no counted response" vs. "deliberately excluded") judged
acceptable for a single grid cell, and it already gets the correct
practical effect for free: the per-row `Yes`/`Counted` aggregation in
`stats/service.go`'s `GetAttendanceMatrix` only credits `'yes'`/`'no'`/`'maybe'`
cells, so a `'pending'` cell is already excluded from both, identical to
what the other three stats queries achieve via `'excluded'`.

## Risks

- **New cross-member write path**: this is the first place one member can
  cause a write against another member's data outside team-admin actions
  (role assignment, removal). Keeping the endpoint single-purpose (one
  boolean, nothing else) bounds the risk.
- **Two different stats-only encodings of the same underlying case**
  (`'excluded'` in three queries, `'pending'` in the matrix) is a
  deliberate tradeoff to avoid touching the wire-level `AttendanceStatus`
  enum — documented here so a future reader doesn't "fix" the matrix query
  to also emit `'excluded'` without realizing that would break JSON
  contract validation against the OpenAPI enum.
