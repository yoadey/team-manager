import { useQuery, type UseQueryResult } from '@tanstack/react-query';
import type { api as defaultApi } from '@/services';
import { queryKeys } from '@/query/keys';
import type { DateRange, StatsOverview } from '@/types';

/**
 * The team's attendance stats overview for the selected date range,
 * team-and-range-scoped so a team switch or range change swaps the cache
 * entry instead of racing (mirrors the pre-migration loadStats()/
 * loadStatsSeq activeTeamId guard).
 */
export function useStatsQuery(
  api: typeof defaultApi,
  teamId: string | null,
  range: DateRange | null,
): UseQueryResult<StatsOverview> {
  return useQuery({
    queryKey: queryKeys.stats(teamId ?? '', range),
    queryFn: () => api.stats.teamOverview(teamId!, range),
    enabled: !!teamId,
  });
}
