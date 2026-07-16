import { useMutation, useQueryClient } from '@tanstack/react-query';
import type { api as defaultApi } from '@/services';
import { queryKeys } from '@/query/keys';
import { useInvalidateTeamQuery } from '@/query/useInvalidateTeamQuery';

function useInvalidatePolls(teamId: string | null) {
  return useInvalidateTeamQuery(teamId, queryKeys.polls);
}

export interface SavePollInput {
  question: string;
  options: string[];
  multiple: boolean;
  anonymous: boolean;
}

export function useSavePollMutation(api: typeof defaultApi, teamId: string | null) {
  const invalidate = useInvalidatePolls(teamId);
  return useMutation({
    mutationFn: (payload: SavePollInput) => api.polls.create(teamId!, payload),
    onSuccess: () => invalidate(),
  });
}

export function useVotePollMutation(api: typeof defaultApi, teamId: string | null) {
  const invalidate = useInvalidatePolls(teamId);
  return useMutation({
    mutationFn: ({ pollId, optionIds }: { pollId: string; optionIds: string[] }) =>
      api.polls.vote(pollId, optionIds, teamId!),
    onSuccess: () => invalidate(),
  });
}

// Takes the team id per call rather than the hook-bound active team id,
// mirroring useDeleteEventMutation/useRemoveMemberMutation -- the confirm
// sheet that triggers this can still be open (and get confirmed) after the
// user has switched to a different active team.
export function useDeletePollMutation(api: typeof defaultApi) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, teamId }: { id: string; teamId: string }) => api.polls.remove(id, teamId),
    onSuccess: (_data, { teamId }) => qc.invalidateQueries({ queryKey: queryKeys.polls(teamId) }),
  });
}
