import { useQuery, type UseQueryResult } from '@tanstack/react-query';
import type { api as defaultApi } from '@/services';
import { queryKeys } from '@/query/keys';
import type { PushCategoryPreferences } from '../types';

/** The caller's per-category Web Push preferences for a team, team-scoped so a team switch swaps the cache entry. */
export function usePushPreferencesQuery(
  api: typeof defaultApi,
  teamId: string | null,
  enabled = true,
): UseQueryResult<PushCategoryPreferences> {
  return useQuery({
    queryKey: queryKeys.pushPreferences(teamId ?? ''),
    queryFn: () => api.push.getPreferences(teamId!),
    enabled: !!teamId && enabled,
  });
}
