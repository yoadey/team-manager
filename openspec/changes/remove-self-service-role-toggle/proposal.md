## Why

The Profile Settings sheet (`ProfileSheet` in `frontend/src/features/team/components/NavSheets.tsx`, opened via the avatar button) still shows a "Meine Rollen in {team}" block that lets a user toggle which of the team's roles they themselves hold. The i18n label itself still carries a literal `(Demo)` suffix (`team.myRolesInTeam` in `de.ts`/`en.ts`), dating back to when the app was mock-only, before the Go backend and the real Members management screen existed.

This capability is redundant today: it calls the exact same `PUT /teams/{teamId}/members/{membershipId}/roles` endpoint (`api.members.setRoles`) that admin member management already exposes in `MemberSheets.tsx`, where a member with `settings:write` can edit any member's roles — including their own, since they appear in the member list. Keeping a second entry point to the same mutation duplicates UI, state plumbing (`toggleMyRole` in `useRoleActions.ts` and `AppContext.tsx`, including race-chaining logic), and tests, for no behavioral gain, and the stray `(Demo)` label signals leftover demo scaffolding rather than an intended product feature.

## What Changes

- Remove the interactive "my roles" toggle block from `ProfileSheet` (`NavSheets.tsx`).
- Remove the now-unused `toggleMyRole` action from `useRoleActions.ts` and its wiring/type in `AppContext.tsx`.
- Remove the i18n keys used only by this block (`team.myRolesInTeam`, `team.multiRoleHint`, `team.toastRolesSaved`) from `de.ts`/`en.ts`. `team.roleAtLeastOne` stays — it's shared with `MemberSheets.tsx`.
- Remove/adjust the tests that exercised this block and action (`NavSheets.test.tsx`, `useRoleActions.test.ts`, `AppContext.test.tsx` references).
- Keep the read-only `myRoles()` helper and its other call sites (`TeamPage.tsx`'s read-only role display, `useEventActions.ts`) untouched — only the self-toggle write path goes away.
- No backend change: `setMemberRoles` stays, since admin member management keeps using it.

## Capabilities

### New Capabilities
- `profile-settings`: what a user can change about their own account/profile from the Profile Settings sheet, and that team-role assignment (including one's own) is done exclusively through Members management, not from Profile Settings.

### Modified Capabilities
<!-- none -->

## Impact

- Frontend only: `frontend/src/features/team/components/NavSheets.tsx`, `frontend/src/features/team/hooks/useRoleActions.ts`, `frontend/src/context/AppContext.tsx`, `frontend/src/i18n/de.ts`, `frontend/src/i18n/en.ts`, and the corresponding test files (`NavSheets.test.tsx`, `useRoleActions.test.ts`, `AppContext.test.tsx`).
- No API/schema change; `backend/openapi/openapi.yaml`'s `setMemberRoles` operation and its admin usage are untouched.
