import { useMutation, useQueryClient } from '@tanstack/react-query';
import type { api as defaultApi } from '@/services';
import { queryKeys } from '@/query/keys';
import { useInvalidateTeamQuery } from '@/query/useInvalidateTeamQuery';

function useInvalidateNews(teamId: string | null) {
  return useInvalidateTeamQuery(teamId, queryKeys.news);
}

export interface SaveNewsInput {
  // Explicit `| undefined` -- the create/update decision is made by the
  // caller passing `undefined` for a new item, not by omitting the key.
  id?: string | undefined;
  payload: { title: string; body: string; pinned: boolean };
}

export function useSaveNewsMutation(api: typeof defaultApi, teamId: string | null) {
  const invalidate = useInvalidateNews(teamId);
  return useMutation({
    mutationFn: ({ id, payload }: SaveNewsInput) =>
      id ? api.news.update(id, payload, teamId!) : api.news.create(teamId!, payload),
    onSuccess: () => invalidate(),
  });
}

// Takes the team id per call rather than the hook-bound active team id,
// mirroring useDeleteEventMutation/useRemoveMemberMutation -- the confirm
// sheet that triggers this can still be open (and get confirmed) after the
// user has switched to a different active team.
export function useDeleteNewsMutation(api: typeof defaultApi) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, teamId }: { id: string; teamId: string }) => api.news.remove(id, teamId),
    onSuccess: (_data, { teamId }) => qc.invalidateQueries({ queryKey: queryKeys.news(teamId) }),
  });
}
