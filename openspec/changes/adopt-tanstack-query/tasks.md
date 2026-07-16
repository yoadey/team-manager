## 1. Setup
- [x] 1.1 Add `@tanstack/react-query@^5` (and devtools as a dev-only dynamic import)
- [x] 1.2 Create `src/query/client.ts` with a `QueryClient`; retry policy excludes `AuthError`/`ForbiddenError`/`ValidationError`
- [x] 1.3 Wrap the app in `<QueryClientProvider>` in `main.tsx`, inside existing ErrorBoundaries
- [x] 1.4 Create `src/query/keys.ts` — team-scoped key factory (`['teams', teamId, <resource>]`)

## 2. Queries & mutations
- [x] 2.1 Add `use<Resource>` query hooks for **events** (`useEventsQuery`, `useEventDetailQuery`). Remaining resources
      (members, finances, polls, news, absences, notifications, stats) are follow-up work — see section 4.
- [x] 2.2 Add `use<Action>` mutation hooks for events (attendance, nomination, comments, event CRUD, cancel/delete/
      reactivate) with `onSuccess` → `invalidateQueries` on the team-scoped key
- [x] 2.3 Replace the global `busy` flag usage with `mutation.isPending` for the two events actions that used it
      (`saveEvent` → `state.savingEvent`, `submitComment` → `state.savingComment`). `busy` itself stays for the
      not-yet-migrated verticals (absences, members, finances, news, polls, team) — see 3.3.

## 3. Context slimming
- [x] 3.1 Remove events' list state, loader, and sequence ref (`events`, `refreshEvents`, `refreshEventsSeq`) and
      drop `events` from the `afterLoginLoad` `Promise.allSettled` bundle. Other verticals' loaders/sequence refs stay
      until they're migrated (4.2).
- [x] 3.2 Auth session, `activeTeamId`, route/URL sync, toast, and sheet state remain in `AppContext` (unchanged;
      events was the only slice removed so far)
- [ ] 3.3 Delete `clearBusyIfOwned` and the `busy` field once fully migrated (blocked on 4.2 — still used by
      absences/members/finances/news/polls/team)
- [x] 3.4 On deep-link/`popstate` restore for an event detail sheet, `openEventDetail` now only sets `eventId`;
      `EventDetailSheet` loads its data via `useEventDetailQuery` on mount instead of an imperative reload

## 4. Migration order
- [x] 4.1 Migrate the `events` vertical fully and get it green first — done, including its own query/mutation hooks,
      per-operation pending state, and the page/detail/calendar/export components reading via the hooks directly
      (not through `AppContext`)
- [ ] 4.2 Migrate remaining features one at a time: members, finances, polls, news, absences, notifications, stats.
      Each follows the events vertical's pattern (query hook consumed directly by feature components + mutation
      hooks with `invalidateQueries`); `AppContext`'s corresponding loader/sequence-ref/`Promise.allSettled` slot is
      then removed the same way section 3 removed events'.
  - [x] 4.2.1 `members` — `useMembersQuery`/`useSaveMemberMutation`/`useRemoveMemberMutation`
        (`useMemberMutations.ts`); `MembersPage`, `AppShell`, and `PenaltyAssignSheet` read the query directly;
        `saveMember`'s `busy === 'save'` replaced by `mutation.isPending` (`state.savingMember`, mirroring
        `savingEvent`); `members` dropped from `AppState`/`afterLoginLoad`/`ensureRouteData`; still-unmigrated
        callers (`useTeamActions.uploadMyPhoto`, `useFinanceActions.openPenaltyAssign`) bridged to
        `useInvalidateMembers`/the sheet's own query instead of the old `refreshMembers` loader.
  - [x] 4.2.2 `finances` — `useFinanceOverviewQuery` (`useFinanceQueries.ts`); `useSaveTxMutation`/
        `useDeleteTxMutation`/`useSavePenaltyMutation`/`useDeletePenaltyMutation`/`useSavePenaltyAssignMutation`/
        `useDeleteAssignmentMutation`/`useSaveContribMutation`/`useTogglePenaltyMutation`/
        `useToggleContributionMutation` (`useFinanceMutations.ts`); `FinancesPage`, `PenaltyCatalogSheet`,
        `TxFormSheet`, and `PenaltyAssignSheet` read the query directly; the four save flows' `busy === 'save'`
        replaced by per-mutation `isPending` (`state.savingTx`/`savingPenalty`/`savingPenaltyAssign`/
        `savingContrib`, mirroring `savingMember`); the three confirm-gated deletes (tx/penalty/assignment) take
        `teamId` per call rather than a hook-bound reactive value, mirroring `useDeleteEventMutation`/
        `useRemoveMemberMutation`; `finances` dropped from `AppState`/`afterLoginLoad`/`ensureRouteData`.
  - [x] 4.2.3 `polls` — `usePollsQuery` (`usePollQueries.ts`); `useSavePollMutation`/`useVotePollMutation`/
        `useDeletePollMutation` (`usePollMutations.ts`); `PollsPage` reads the query directly; `savePoll`'s
        `busy === 'save'` replaced by `mutation.isPending` (`state.savingPoll`); `removePoll` (confirm-gated) takes
        `teamId` per call, mirroring `useDeleteEventMutation`/`useRemoveMemberMutation`; the old loader's paired
        `loadNotifications()` refresh is preserved both per-action (`votePoll`/`savePoll`/`removePoll` each still
        call it after success, mirroring the events vertical) and per-navigation (`ensureRouteData`'s existing
        route-dispatch mechanism now has a `polls` branch calling `loadNotifications()`, reusing the same
        internal-only loader `loadStats`/`loadNews` already use there instead of exposing it publicly); `polls`
        dropped from `AppState`/`afterLoginLoad`/`ensureRouteData`'s data-fetch branches.
  - [x] 4.2.4 `news` — `useNewsQuery` (`useNewsQueries.ts`); `useSaveNewsMutation`/`useDeleteNewsMutation`
        (`useNewsMutations.ts`); `NewsPage` and `Home` (dashboard preview) read the query directly; `saveNews`'s
        `busy === 'save'` replaced by `mutation.isPending` (`state.savingNews`); `removeNews` (confirm-gated) takes
        `teamId` per call, mirroring `useDeleteEventMutation`/`useRemoveMemberMutation`; the old loader's paired
        `loadNotifications()` refresh is preserved both per-action (`saveNews`/`removeNews` each still call it after
        success) and per-navigation (`ensureRouteData`'s `news` branch now calls `loadNotifications()` instead of the
        removed `loadNews()`, sharing the same branch as `polls`); `news` dropped from
        `AppState`/`afterLoginLoad`/`ensureRouteData`'s data-fetch branches.
  - [ ] 4.2.5 `absences`
  - [ ] 4.2.6 `notifications`
  - [ ] 4.2.7 `stats`

## 5. Verification
- [x] 5.1 `npm run typecheck` + `npm run lint` green
- [x] 5.2 `npm run test` green; tests use a per-test `QueryClient` with `retry: false`
      (`src/test/queryTestUtils.tsx`'s `createTestQueryClient`/`createQueryWrapper`)
- [x] 5.3 `npm run build` + `check:bundle` under budget (228.1 KB gzipped total, largest chunk 94.6 KB — budget is
      250 KB/chunk, 600 KB total)
- [x] 5.4 Manual/automated smoke: rapid team switch shows no stale event lists (`useEventQueries.test.ts`'s
      "discards a stale response for a previous team after switching teams"); a mutation invalidates and refreshes
      its list (every events mutation test); a failing module leaves others intact (`AppContext.test.tsx`'s
      afterLoginLoad ForbiddenError/403 tests)
