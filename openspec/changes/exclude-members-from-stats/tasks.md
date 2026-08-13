## 1. Database
- [x] 1.1 `00029_member_stats_exclusion.sql`: `ALTER TABLE memberships ADD
      COLUMN exclude_from_stats boolean NOT NULL DEFAULT false;`
- [ ] 1.2 `make migrate` locally if Docker is available; otherwise rely on CI's
      `backend-migration-rollback`/`backend-migration-safety` gates (no Docker
      in this dev sandbox -- deferred to CI)

## 2. OpenAPI
- [x] 2.1 `Member`, `UpdateMemberRequest`: add `excludeFromStats: boolean`
- [x] 2.2 `cd backend && make generate` (commit `internal/gen/api.gen.go`)
- [x] 2.3 repo-root `make generate-ts` (commit `frontend/src/api/types.gen.ts`)

## 3. Backend: members module
- [x] 3.1 `model.go`: `ExcludeFromStats bool` on `MemberRow`; add to
      `MemberPatch`
- [x] 3.2 `repository.go`: `UpdateMember` patches the new column

## 4. Backend: stats module
- [x] 4.1 `stats/repository.go`: add `AND m.exclude_from_stats = false` to the
      `memberships m` roster join in `MemberStats`, `SingleMemberStats`,
      `AbsenceStats`, and the attendance-matrix queries
      (`matrixColumns`/`matrixCells`)
- [x] 4.2 Leave `EventStats`'s membership join unfiltered (see design.md) —
      added a comment at the call site pointing to the design decision so
      a future editor doesn't "fix" it by accident
- [x] 4.3 Confirmed `SingleMemberStats` on an excluded member returns
      `pgx.ErrNoRows` (same shape as a non-member), mapped by the handler to
      404 "member not found" -- reusing established semantics rather than a
      new response variant

## 5. Backend: tests
- [x] 5.1 `members/repository_test.go`: `UpdateMember` persists
      `excludeFromStats`
- [x] 5.2 `stats/repository_test.go`: an excluded member is absent from
      `MemberStats`/matrix rows/`SingleMemberStats`/`AbsenceStats`, but their
      historical yes/no response still counts in `EventStats`'s per-event
      turnout

## 6. Frontend
- [x] 6.1 `features/members/components/memberFormSchema.ts`: add
      `excludeFromStats: z.boolean().optional()`
- [x] 6.2 `features/members/components/MemberSheets.tsx` (`MemberFormSheet`):
      add the toggle, gated on `members:write`
- [x] 6.3 `api/map.ts`, `services/serviceLayerReal.ts`,
      `mocks/{db.ts,handlers.ts}`: wire the field through
- [x] 6.4 `i18n/{de.ts,en.ts}`: label + helper text

## 7. Verification
- [x] 7.1 `cd backend && make test && make lint` (integration tests skip: no
      Docker in this sandbox; unit tests + lint green)
- [x] 7.2 `cd frontend && npm run typecheck && npm run lint && npm test`
- [x] 7.3 `make generate && make generate-ts` produce no drift
- [ ] 7.4 Manual: exclude a member, confirm they disappear from the quota
      list, matrix, and absence table, but a past event they responded to
      still shows the correct total attendee count
