import { useQuery, type UseQueryResult } from '@tanstack/react-query';
import type { api as defaultApi } from '@/services';
import { queryKeys } from '@/query/keys';
import type { FinanceOverview } from '../types';

/** The team's finance overview (balance, transactions, penalties, assignments, contributions), team-scoped so a team switch swaps the cache entry instead of racing. */
export function useFinanceOverviewQuery(
  api: typeof defaultApi,
  teamId: string | null,
): UseQueryResult<FinanceOverview> {
  return useQuery({
    queryKey: queryKeys.finances(teamId ?? ''),
    queryFn: () => api.finances.overview(teamId!),
    enabled: !!teamId,
  });
}
