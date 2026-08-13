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
      `memberships m` roster join in `MemberStats`, `AbsenceStats`, and the
      attendance-matrix queries (`matrixColumns`/`matrixCells`) — these
      genuinely drop the excluded member's row. `SingleMemberStats` is
      different (see 4.3): it does NOT filter the WHERE clause this way,
      since dropping the row there would surface as a 404, not "no data".
- [x] 4.2 Leave `EventStats`'s membership join unfiltered (see design.md) —
      added a comment at the call site pointing to the design decision so
      a future editor doesn't "fix" it by accident
- [x] 4.3 **Corrected during review** (initial implementation had this
      backwards): `SingleMemberStats` must NOT return `pgx.ErrNoRows` for an
      excluded member — that conflated "excluded" with "not a member" and
      surfaced as an incorrect 404 through `GetMemberStats`, contradicting
      this same design.md's explicit decision below. Fixed by keeping the
      membership row in the query (WHERE only checks team_id/user_id) and
      instead forcing `eff = 'pending'` via a `CASE WHEN m.exclude_from_stats
      THEN 'pending' ...` branch evaluated before the real per-event status,
      so `yes`/`counted` both come out 0 (the same "no data" shape as a
      member with zero events in range) without leaking their real
      responses. A genuine non-member still produces zero rows → 404, since
      the WHERE clause still requires a matching membership row to exist at
      all. Regression-covered by
      `TestStatsRepository_ExcludedMember_OmittedFromPersonalQuotas_ButCountedInEventStats`
      and `serviceContract.test.ts`'s matching drift-bug-fix test.

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
