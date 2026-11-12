import { useQuery, type UseQueryResult } from '@tanstack/react-query';
import type { api as defaultApi } from '@/services';
import type { StatsGroupThreshold } from '@/types';
import { queryKeys } from '@/query/keys';

/**
 * Every role's attendance threshold for the team (default applied), for the
 * team settings sheet. Settings-gated on the server (settings:read), unlike
 * the statistics page itself, which reads each group's resolved threshold
 * straight off the overview response.
 */
export function useStatsGroupThresholdsQuery(
  api: typeof defaultApi,
  teamId: string | null,
): UseQueryResult<StatsGroupThreshold[]> {
  return useQuery({
    queryKey: queryKeys.statsGroupThresholds(teamId ?? ''),
    queryFn: () => api.stats.groupThresholds(teamId!),
    enabled: !!teamId,
  });
}
