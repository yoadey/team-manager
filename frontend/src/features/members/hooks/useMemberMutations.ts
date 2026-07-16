import { useMutation, useQueryClient } from '@tanstack/react-query';
import type { api as defaultApi } from '@/services';
import { queryKeys } from '@/query/keys';
import { useInvalidateTeamQuery } from '@/query/useInvalidateTeamQuery';
import type { Member } from '../types';
import type { User } from '@/types';

/** Invalidates the team's member list, returning a promise that resolves once the invalidated query has refetched. */
export function useInvalidateMembers(teamId: string | null) {
  return useInvalidateTeamQuery(teamId, queryKeys.members);
}

export interface SaveMemberInput {
  membershipId: string;
  patch: { name: string; email: string; phone: string; birthday: string; address: string; group: string };
  roleIds: string[];
  rolesChanged: boolean;
  /** Self's own changed photo (already gated by the caller to self-only); omitted otherwise. */
  photo?: string | null;
  /** Whether this save is the caller's own profile -- refreshes the session user/teams too. */
  self?: boolean;
}

export interface SaveMemberResult {
  member: Member;
  user: User | null;
}

// Everything that must stay "busy" for a save -- profile patch, role
// assignment, self-photo upload, and the self-profile session refresh -- runs
// inside this one mutationFn, so `isPending` (exposed as `savingMember`)
// covers the whole operation. Splitting the self-refresh out into a separate
// awaited step after `mutateAsync` resolves would flip `isPending` back to
// false while that step is still in flight, re-enabling the Save button and
// allowing a double-submit -- exactly what the pre-migration shared `busy`
// flag stayed set through.
export function useSaveMemberMutation(
  api: typeof defaultApi,
  teamId: string | null,
  refreshTeams: () => Promise<void>,
) {
  const invalidate = useInvalidateMembers(teamId);
  return useMutation({
    mutationFn: async ({
      membershipId,
      patch,
      roleIds,
      rolesChanged,
      photo,
      self,
    }: SaveMemberInput): Promise<SaveMemberResult> => {
      let member = await api.members.update(membershipId, patch, teamId!);
      // Role assignment is a separate write path (members.setRoles -> PUT
      // .../roles, gated on settings:write) from the profile-field patch
      // (members.update -> PATCH .../{membershipId}, gated on members:write)
      // -- the backend's UpdateMember handler never applies a roleIds field
      // embedded in the PATCH body, so it must be sent via setRoles()
      // whenever it actually changed.
      if (rolesChanged) member = await api.members.setRoles(membershipId, roleIds, teamId!);
      // Photo has its own dedicated endpoint (auth.setPhoto, self-only --
      // there is no backend endpoint to set another member's photo at all),
      // not a members.update() field.
      if (photo) await api.auth.setPhoto(photo);
      if (!self) return { member, user: null };
      const user = await api.auth.currentUser();
      await refreshTeams();
      return { member, user };
    },
    onSuccess: () => invalidate(),
  });
}

// Takes the member's own team id per call rather than the hook-bound active
// team id, mirroring useDeleteEventMutation/useEventStatusMutation -- the
// confirm sheet that triggers this can still be open (and get confirmed)
// after the user has switched to a different active team, and React Query
// always runs the mutationFn from the most recently rendered hook call, not
// the one bound when the confirm dialog was opened.
export function useRemoveMemberMutation(api: typeof defaultApi) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ membershipId, teamId }: { membershipId: string; teamId: string }) =>
      api.members.remove(membershipId, teamId),
    onSuccess: (_data, { teamId }) => qc.invalidateQueries({ queryKey: queryKeys.members(teamId) }),
  });
}
