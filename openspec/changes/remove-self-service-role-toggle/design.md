## Context

`ProfileSheet` (`NavSheets.tsx`) renders a "Meine Rollen in {team}" checklist that lets the signed-in user toggle their own team roles directly from Profile Settings, gated by `app.can('settings', 'write')`. It calls `app.toggleMyRole`, which is backed by `useRoleActions.ts`'s `toggleMyRole` — the same `api.members.setRoles` call `MemberSheets.tsx` already uses to edit any member's roles (including the signed-in user's own membership row, since they appear in the team's member list). The i18n label for this block still reads `(Demo)`, a leftover from before the real backend/member-management screen existed.

## Goals / Non-Goals

**Goals:**
- Remove the duplicate self-service role-toggle UI and its dedicated plumbing from Profile Settings.
- Leave role assignment reachable through exactly one place: Members management.
- Leave the read-only "my roles" display (`myRoles()`) and unrelated Profile Settings sections (photo, color scheme, language, logout, data export/erasure) untouched.

**Non-Goals:**
- Changing the RBAC model, the `settings:write` gating, or the `setMemberRoles` backend endpoint.
- Changing how `MemberSheets.tsx` edits roles for any member, including the current user's own row.

## Decisions

- Delete the role-list block (`NavSheets.tsx`, the `key="rs"` section of `ProfileSheet`) entirely rather than making it read-only — a read-only duplicate of `TeamPage.tsx`'s existing read-only role display would still be dead weight.
- Delete `toggleMyRole` rather than keeping it unused, since nothing else calls it once the block is gone.
- Keep `team.roleAtLeastOne` (shared with `MemberSheets.tsx`); drop `team.myRolesInTeam`, `team.multiRoleHint`, `team.toastRolesSaved` (exclusive to the removed block).

## Risks / Trade-offs

- A user without access to Members management (i.e. without `settings:write`) already couldn't change their own roles from Profile Settings either (the block was disabled for them), so no capability is lost for any user who previously had it.
