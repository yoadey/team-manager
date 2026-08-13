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

**`EffectiveStatusExpr` gains a terminal `'excluded'` branch, evaluated
before the opt-out fallback.** Today: `explicit response → that response`,
else `absence covers date → 'no'`, else `opt-out event → 'yes'`, else
`'pending'`. New: `explicit response → that response`, else
`not-relevant absence covers date → 'excluded'`, else `(any other)
absence covers date → 'no'`, else `opt-out event → 'yes'`, else
`'pending'`. `'excluded'` is deliberately a new distinct value, not a reuse
of `'pending'` — `'pending'` means "we don't know", `'excluded'` means "we
know, and it's being deliberately left out." Every `COUNT(*) FILTER (WHERE
eff IN ('yes','no','maybe'))` site already excludes anything outside that
allowlist, so `'excluded'` is dropped automatically with no change needed
beyond the CASE expression itself.

**Audit the blast radius of `EffectiveStatusExpr` beyond stats.** This
expression is shared with non-statistics consumers (e.g. an event's own
attendance-summary display). Implementation must locate every caller and
confirm a `'excluded'` cell renders sensibly there too (most likely: same
as `'pending'`/no visible response, since from a non-statistics viewer's
perspective the member simply has no counted response for that date) —
this is a required implementation check, not assumed correct by
construction.

## Risks

- **New cross-member write path**: this is the first place one member can
  cause a write against another member's data outside team-admin actions
  (role assignment, removal). Keeping the endpoint single-purpose (one
  boolean, nothing else) bounds the risk.
- **Shared SQL expression**: `EffectiveStatusExpr` change must be audited
  against every current caller, not just the five `stats/repository.go`
  queries, to avoid an unintended display regression.
