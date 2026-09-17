## Context

`frontend/src/mocks/handlers.ts` is a hand-written parallel implementation
of every team-scoped operation in `backend/openapi/openapi.yaml`, wired up
through MSW so the frontend can run against a backend-less demo/test
double instead of the real Go server. It already replicates several
pieces of backend business logic faithfully (event effective-status
defaulting, drift-bug-fixed poll/contribution edge cases, enumeration-safe
auth responses) but never replicated RBAC: almost every handler runs the
requested mutation or read unconditionally once (if at all) `requireAuth()`
confirms the caller is logged in. `db.ts` already models everything RBAC
enforcement needs — `RoleDto.permissions: Permissions` per role,
`Membership.roleIds`, and `mergePerms(roles)` folding multiple roles' levels
per module (max across roles) — none of it is consulted at the HTTP-handler
layer.

The authoritative source for which module gates which route, and whether a
route is self-service, is `backend/openapi/openapi.yaml`'s
`x-rbac-module`/`x-rbac-self-service` extensions — the same source
`cmd/genrbac` compiles into `internal/middleware/rbac_table.gen.go` for the
real backend. That generated table (read directly, not re-derived) is the
ground truth used to classify every handler below.

## Goals / Non-Goals

**Goals:**
- Every module-gated handler in `handlers.ts` enforces the same
  read/write/self-service rule the real backend's `RequirePermission`
  enforces, using the same module and self-service classification as
  `rbac_table.gen.go`.
- `serviceContract.test.ts` gains coverage that would catch a regression
  removing a permission check from either the real backend or the
  frontend's own gating, without that regression being masked by the mock
  quietly allowing everything.
- Existing tests that only passed because the mock enforced nothing keep
  passing, but by being given a caller with the permission the scenario
  actually requires — not by leaving the new enforcement out of their path.

**Non-Goals:**
- No change to `backend/openapi/openapi.yaml` or any generated code —
  this is a mock-only change matching an already-documented contract.
- No change to `x-rbac-module: public` routes' behavior (team info/photo/
  logo, absences, notifications, push-preferences, shared-calendars) —
  they correctly require membership only, not a module permission, per
  CLAUDE.md and `authz.go`.
- Not replicating `RequireMembership` (the real backend's separate
  "caller belongs to this team at all" 404 check) in the mock. The new
  `permissionFor` returns all-`none` permissions for a caller with no
  membership row on the team, which `requirePermission` turns into a 403
  — different status code than the real backend's 404 for a genuine
  non-member, but the finding this change addresses is specifically about
  per-module read/write enforcement, and every existing demo/test caller
  is already a member of the teams it acts on. Replicating the 404 path
  faithfully would require a second "is this caller a member at all"
  gate threaded through the same ~64 handlers for no behavior any current
  test exercises; left as a follow-up if a future test needs it.

## Decisions

**A single `requirePermission(userId, teamId, module, level)` helper,
called with an already-resolved `level`.** The real backend's
`hasRequiredPermission` computes the required level from
`(selfService, method)`: self-service or a read method needs `read`,
everything else needs `write`. Rather than have the mock helper re-derive
that from a `selfService` flag plus the request method, each call site
simply passes the correct literal (`'read'` or `'write'`) for its specific
route — the classification is static per (method, route) pair and reads
more directly at the call site than threading a boolean through. This
mirrors how `requireAuth()` is already a small, call-site-driven helper in
this file rather than a generic middleware.

**Ownership checks stay separate from `requirePermission`.** Self-service
routes still need a second check for "is this the caller's own record"
where the underlying operation is scoped to the caller (member title,
comment deletion, poll vote, stats last-selection) — `requirePermission`
only answers "does the caller have at least `level` on `module`", exactly
like the real backend's `RequirePermission` middleware, which is likewise
silent on per-record ownership; that's the handler/service layer's job on
both sides. `setMemberTitle` already had this check; `deleteEventComment`
gains one (previously missing any check at all); `setAttendance` gains an
acting-on-another-member escalation check (`events:write` required unless
`body.userId === auth`), matching
`backend/internal/events/service.go`'s `SetAttendance`.

**`permissionFor` returns all-`none` for a non-member rather than
throwing.** Keeps `requirePermission` a total function over any
`(userId, teamId)` pair the same way the real backend's
`GetPermissions` does (it just returns the DB's folded zero-value for a
user with no membership rows), and naturally yields the correct 403
outcome without a separate not-a-member branch.

## Risks / Trade-offs

- **Status code divergence for genuine non-members (403 vs. the real
  backend's 404).** Accepted per the Non-Goals section above — no current
  test relies on this path, and adding it would be a second cross-cutting
  change to the same set of handlers for a scenario nothing exercises yet.
- **Large, mechanical diff (~64 handlers touched).** Kept intentionally
  boring: each call site gets the same `requireAuth()` (if not already
  present) + `requirePermission(...)` two-line prologue, with the module
  and level chosen directly from `rbac_table.gen.go`, rather than
  introducing a generic per-route-table abstraction that would itself need
  to be kept in sync with the OpenAPI spec by hand (the opposite of the
  spec-generated approach the real backend uses).
