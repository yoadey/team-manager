## 1. Database
- [ ] 1.1 `00029_member_stats_exclusion.sql`: `ALTER TABLE memberships ADD
      COLUMN exclude_from_stats boolean NOT NULL DEFAULT false;`
- [ ] 1.2 `make migrate` locally if Docker is available; otherwise rely on CI's
      `backend-migration-rollback`/`backend-migration-safety` gates

## 2. OpenAPI
- [ ] 2.1 `Member`, `UpdateMemberRequest`: add `excludeFromStats: boolean`
- [ ] 2.2 `cd backend && make generate` (commit `internal/gen/api.gen.go`)
- [ ] 2.3 repo-root `make generate-ts` (commit `frontend/src/api/types.gen.ts`)

## 3. Backend: members module
- [ ] 3.1 `model.go`: `ExcludeFromStats bool` on `MemberRow`; add to
      `MemberPatch`
- [ ] 3.2 `repository.go`: `UpdateMember` patches the new column

## 4. Backend: stats module
- [ ] 4.1 `stats/repository.go`: add `AND m.exclude_from_stats = false` to the
      `memberships m` roster join in `MemberStats`, `SingleMemberStats`, and
      the attendance-matrix queries (`matrixColumns`/`matrixCells`)
- [ ] 4.2 Leave `EventStats`'s membership join unfiltered (see design.md) —
      add a short comment at the call site pointing to the design decision so
      a future editor doesn't "fix" it by accident
- [ ] 4.3 Confirm `SingleMemberStats` on an excluded member returns the
      existing empty-result shape rather than erroring

## 5. Backend: tests
- [ ] 5.1 `members/repository_test.go`: `UpdateMember` persists
      `excludeFromStats`
- [ ] 5.2 `stats/repository_test.go`: an excluded member is absent from
      `MemberStats`/matrix rows/`SingleMemberStats`, but their historical
      yes/no response still counts in `EventStats`'s per-event turnout

## 6. Frontend
- [ ] 6.1 `features/members/components/memberFormSchema.ts`: add
      `excludeFromStats: z.boolean().default(false)`
- [ ] 6.2 `features/members/components/MemberSheets.tsx` (`MemberFormSheet`):
      add the toggle
- [ ] 6.3 `api/map.ts`, `services/serviceLayerReal.ts`,
      `mocks/{db.ts,handlers.ts}`: wire the field through
- [ ] 6.4 `i18n/{de.ts,en.ts}`: label + helper text

## 7. Verification
- [ ] 7.1 `cd backend && make test && make lint`
- [ ] 7.2 `cd frontend && npm run typecheck && npm run lint && npm test`
- [ ] 7.3 `make generate && make generate-ts` produce no drift
- [ ] 7.4 Manual: exclude a member, confirm they disappear from the quota
      list and matrix, but a past event they responded to still shows the
      correct total attendee count
