## Why

`frontend/src/mocks/handlers.ts` (the MSW mock backend used for demo mode
and for `frontend/src/services/serviceContract.test.ts`, which CLAUDE.md
describes as pinning behavioral regression coverage for `realApi` against
the MSW demo backend) registers roughly 94 handlers for team-scoped
operations. Only a minority call `requireAuth()` at all, and essentially
none check the caller's per-module `read`/`write` permission level. The
real backend (`backend/internal/middleware/authz.go`'s `RequirePermission`)
gates every team-scoped route: GET requires at least `read` on the route's
module, mutating methods require `write`, a module set to `none` blocks
GET too (not just writes), and `x-rbac-self-service: true` routes never
require `write` on their own module but still require `read`.

Because the mock doesn't replicate any of this, `serviceContract.test.ts`
has zero test cases asserting a Forbidden response for an under-permissioned
caller. A regression that removes a permission check from the real
backend, or from the frontend UI's own `can()` gating, would pass every
mock-backed test while silently failing (or silently succeeding when it
shouldn't) against the real API. This is a correctness gap in the
project's only automated cross-check between the demo backend and the
documented RBAC contract.

## What Changes

- Add a `requirePermission(userId, teamId, module, level)` helper to
  `frontend/src/mocks/handlers.ts`, backed by a new `permissionFor(userId,
  teamId)` helper in `frontend/src/mocks/db.ts` that folds the caller's
  roles for a team into effective permissions (max across roles, `write`
  implies `read`, mirroring `db.ts`'s existing `mergePerms`).
- Apply `requirePermission` to every team-scoped handler in `handlers.ts`
  whose route is module-gated per `backend/openapi/openapi.yaml`'s
  `x-rbac-module`/`x-rbac-self-service` extensions (the same table
  `cmd/genrbac` compiles into `internal/middleware/rbac_table.gen.go`),
  using the correct module and the correct required level (`read` for
  GET/HEAD-equivalent and for self-service mutations, `write` for every
  other mutation) for each route. `x-rbac-module: public` routes (team
  info/photo/logo, absences, notifications, push-preferences,
  shared-calendars) are untouched — they correctly require nothing beyond
  membership.
- Add an explicit ownership check to `DELETE
  /teams/{teamId}/events/{eventId}/comments/{commentId}`, which previously
  had no auth check at all and could delete any comment; it now 404s
  unless the caller owns the comment, matching
  `backend/internal/events/repository.go`'s `DeleteComment`.
- Add an acting-on-another-member permission check to `POST
  /teams/{teamId}/events/{eventId}/attendance`: setting one's own
  attendance stays self-service (`events: read`), setting another
  member's attendance now requires `events: write`, matching
  `backend/internal/events/service.go`'s `SetAttendance`.
- Add new `serviceContract.test.ts` cases covering: a caller whose role
  has the route's module set to `none` gets 403 on GET; a caller with
  `read`-only gets 403 on a mutating request; a self-service route (member
  title, event comment/attendance, poll vote) still works for a
  `read`-only caller acting on their own record.

## Capabilities

### Modified Capabilities
- `demo-mode`: the MSW mock backend now enforces the same per-module
  RBAC checks (including self-service exceptions) as the real backend,
  instead of silently allowing every request through once authenticated.

## Impact

- `frontend/src/mocks/db.ts`: new `permissionFor` helper.
- `frontend/src/mocks/handlers.ts`: new `requirePermission` helper; ~64
  handlers gain a permission check; the comment-delete and
  set-attendance-for-another-member handlers gain the ownership/escalation
  checks described above.
- `frontend/src/services/serviceContract.test.ts`: new 403 coverage; any
  existing test whose seeded caller lacked permission for a call it made
  (and was only passing because the mock didn't enforce anything) has its
  setup fixed to grant the permission the scenario actually needs.
- `frontend/src/context/AppContext.test.tsx`: four tests called mock API
  routes as setup without ever establishing a session (previously harmless,
  since those routes had no `requireAuth()` at all); fixed to establish and
  then tear down a session around the setup calls, so the app under test
  still mounts against a clean, unauthenticated mock as each test expects.
- No API contract change — `backend/openapi/openapi.yaml` is unchanged, so
  no `make generate` / `make generate-ts` is required.
