## 1. Frontend UI
- [x] 1.1 Remove the "Meine Rollen in {team}" role-toggle block (`key="rs"` section) from `ProfileSheet` in `frontend/src/features/team/components/NavSheets.tsx`
- [x] 1.2 Remove now-unused imports/variables in `NavSheets.tsx` that only existed for that block (e.g. `roles`, `myIds`, `canEditMyRoles` if unused elsewhere in the file)

## 2. State/action plumbing
- [x] 2.1 Remove `toggleMyRole` from `frontend/src/features/team/hooks/useRoleActions.ts` (including the `toggleChain` race-serialization logic that existed only for it)
- [x] 2.2 Remove `toggleMyRole` from the `AppContext` type and its wiring in `frontend/src/context/AppContext.tsx`
- [x] 2.3 Confirm `myRoles()` and its other call sites (`TeamPage.tsx`, `useEventActions.ts`) are untouched

## 3. i18n
- [x] 3.1 Remove `team.myRolesInTeam`, `team.multiRoleHint`, `team.toastRolesSaved` from `frontend/src/i18n/de.ts` and `en.ts`
- [x] 3.2 Keep `team.roleAtLeastOne` (still used by `MemberSheets.tsx`)

## 4. Tests
- [x] 4.1 Remove/update `ProfileSheet` role-toggle tests in `frontend/src/features/team/components/NavSheets.test.tsx`
- [x] 4.2 Remove `toggleMyRole` tests in `frontend/src/features/team/hooks/useRoleActions.test.ts`
- [x] 4.3 Remove `toggleMyRole` references in `frontend/src/context/AppContext.test.tsx` (none existed; only unrelated `myRoles()` mentions)

## 5. Verification
- [x] 5.1 `npm run lint` clean (0 errors, only pre-existing warnings)
- [x] 5.2 `npm run typecheck` clean
- [x] 5.3 `npm test` green (1125 tests); `npm run test:coverage` 87.23/82.4/89.52/88.2 (well above 80/65/75/80 thresholds)
- [x] 5.4 `npm run build` + `npm run check:bundle` green (252.5 KB total gzipped, within 600 KB budget; no chunk over 250 KB)
