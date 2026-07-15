import { useQuery, type UseQueryResult } from '@tanstack/react-query';
import type { api as defaultApi } from '@/services';
import { queryKeys } from '@/query/keys';
import type { Member } from '../types';

/** The team's member list, team-scoped so a team switch swaps the cache entry instead of racing. */
export function useMembersQuery(api: typeof defaultApi, teamId: string | null): UseQueryResult<Member[]> {
  return useQuery({
    queryKey: queryKeys.members(teamId ?? ''),
    queryFn: () => api.members.list(teamId!),
    enabled: !!teamId,
  });
}
