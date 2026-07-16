import { useQuery, type UseQueryResult } from '@tanstack/react-query';
import type { api as defaultApi } from '@/services';
import { queryKeys } from '@/query/keys';
import type { NewsItem } from '../types';

/** The team's news list, team-scoped so a team switch swaps the cache entry instead of racing. */
export function useNewsQuery(api: typeof defaultApi, teamId: string | null): UseQueryResult<NewsItem[]> {
  return useQuery({
    queryKey: queryKeys.news(teamId ?? ''),
    queryFn: () => api.news.list(teamId!),
    enabled: !!teamId,
  });
}
