import { useQuery, type UseQueryResult } from '@tanstack/react-query';
import type { api as defaultApi } from '@/services';
import { queryKeys } from '@/query/keys';
import type { Poll } from '../types';

/** The team's poll list, team-scoped so a team switch swaps the cache entry instead of racing. */
export function usePollsQuery(api: typeof defaultApi, teamId: string | null): UseQueryResult<Poll[]> {
  return useQuery({
    queryKey: queryKeys.polls(teamId ?? ''),
    queryFn: () => api.polls.list(teamId!),
    enabled: !!teamId,
  });
}
