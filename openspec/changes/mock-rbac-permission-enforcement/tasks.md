## 1. Mock RBAC primitives
- [x] 1.1 `frontend/src/mocks/db.ts`: add `permissionFor(userId, teamId):
      Permissions`, folding the caller's roles for the team via the
      existing `mergePerms`/`rolesOf` (all-`none` for a non-member)
- [x] 1.2 `frontend/src/mocks/handlers.ts`: add `requirePermission(userId,
      teamId, module, level)` returning `true | HttpResponse<Problem>`
      (403 `application/problem+json`, matching `requireAuth`'s existing
      `problem()` convention)

## 2. Apply enforcement — events module
- [x] 2.1 `listEvents`, `getEvent` → `events:read`
- [x] 2.2 `createEvent`, `updateEvent`, `deleteEvent`, `setEventStatus` →
      `events:write`
- [x] 2.3 `listEventComments`, `listAttendance` → `events:read`
- [x] 2.4 `addEventComment` → self-service, `events:read`
- [x] 2.5 `deleteEventComment` → self-service, `events:read` + ownership
      check (404 if not the comment's author, matching
      `events.Repository.DeleteComment`)
- [x] 2.6 `setAttendance` → self-service, `events:read` when
      `body.userId === auth`, else `events:write` (matching
      `events.Service.SetAttendance`'s acting-on-another-member rule)
- [x] 2.7 `setNomination` → `events:write` (not self-service)
- [x] 2.8 `issueCalendarFeedToken`, `revokeCalendarFeedToken`,
      `getCalendarFeedSettings`, `updateCalendarFeedSettings` →
      self-service, `events:read`

## 3. Apply enforcement — members module
- [x] 3.1 `listMembers` → `members:read`
- [x] 3.2 `updateMember`, `removeMember` → `members:write`
- [x] 3.3 `setMemberTitle` → self-service, `members:read` (ownership check
      already present, kept as-is)

## 4. Apply enforcement — settings module
- [x] 4.1 `updateTeam`, `uploadTeamPhoto`, `deleteTeamPhoto`,
      `uploadTeamLogo`, `deleteTeamLogo`, `createInvite` → `settings:write`
- [x] 4.2 `setMemberRoles` → `settings:write`
- [x] 4.3 `listRoles` → `settings:read`; `createRole`, `updateRole`,
      `deleteRole` → `settings:write`
- [x] 4.4 `listCalendarShares` → `settings:read`; `createCalendarShare`,
      `deleteCalendarShare` → `settings:write`

## 5. Apply enforcement — news, polls modules
- [x] 5.1 `listNews` → `news:read`; `createNews`, `updateNews`,
      `deleteNews` → `news:write`
- [x] 5.2 `listPolls` → `polls:read`; `createPoll`, `deletePoll` →
      `polls:write`
- [x] 5.3 `votePoll` → self-service, `polls:read`

## 6. Apply enforcement — finances module
- [x] 6.1 `getFinanceOverview`, `listTransactions` → `finances:read`
- [x] 6.2 `createTransaction`, `updateTransaction`, `deleteTransaction`,
      `createPenalty`, `updatePenalty`, `deletePenalty`,
      `createPenaltyAssignment`, `deletePenaltyAssignment`,
      `createContributions`, `updateContribution`, `deleteContribution` →
      `finances:write`

## 7. Apply enforcement — stats module
- [x] 7.1 `getStatsOverview`, `getMemberStats`, `getAttendanceMatrix`,
      `getStatsPreferences`, `listStatsPresets` → `stats:read`
- [x] 7.2 `setStatsPreferences` → self-service, `stats:read`
- [x] 7.3 `createStatsPreset`, `updateStatsPreset`, `deleteStatsPreset` →
      `stats:write`

## 8. Tests
- [x] 8.1 `serviceContract.test.ts`: module-`none` caller gets 403 on a
      representative GET per module
- [x] 8.2 `serviceContract.test.ts`: `read`-only caller gets 403 on a
      representative mutating request per module
- [x] 8.3 `serviceContract.test.ts`: self-service routes (member title,
      event comment add/delete, attendance for self, poll vote) still
      succeed for a `read`-only caller acting on their own record
- [x] 8.4 `serviceContract.test.ts`: acting on another member's attendance
      still requires `events:write`
- [x] 8.5 Audit existing tests across the frontend suite for callers that
      lacked the permission their scenario now requires; fix seed/setup
      (grant the right role/permission), not the new enforcement
- [x] 8.6 `src/context/AppContext.test.tsx`: four tests called mock API
      routes directly, as setup, without ever establishing a session --
      previously harmless (those routes had no `requireAuth()` at all), now
      401s that `afterLoginLoad` treats as a genuine failure and logs the
      caller out. Fixed by establishing a real session for setup calls (via
      `api.auth.login`) and logging back out (`api.auth.logout`) before the
      app under test is actually mounted, so its own session-restore
      bootstrap still starts clean at 'login' as each test expects instead
      of discovering the setup-only session and racing ahead with its own
      real login flow. One test (`does not show a "joined" toast...`)
      instead seeds the invite directly on `db.invites` (via the file's
      top-level `sharedMockDb` import, not a `vi.resetModules()`-scoped
      dynamic import of `@/mocks/db`, which resolves to a different module
      instance than the one already wired into the running MSW server)
      since the caller there is a non-admin who wouldn't hold `settings:write`
      to create it through the API.

## 9. Verification
- [x] 9.1 `cd frontend && npm test -- --run`
- [x] 9.2 `cd frontend && npm run typecheck`
- [x] 9.3 `cd frontend && npm run lint`
