import { useQuery, type UseQueryResult } from '@tanstack/react-query';
import type { api as defaultApi } from '@/services';
import { queryKeys } from '@/query/keys';
import type { StatsPreferences, StatsPreset } from '@/types';

/** The caller's last-selected statistics date range for a team, team-scoped
 * so a team switch swaps the cache entry. */
export function useStatsPreferencesQuery(
  api: typeof defaultApi,
  teamId: string | null,
): UseQueryResult<StatsPreferences> {
  return useQuery({
    queryKey: queryKeys.statsPreferences(teamId ?? ''),
    queryFn: () => api.statsPrefs.getPreferences(teamId!),
    enabled: !!teamId,
  });
}

/** The caller's saved named statistics date-range presets for a team. */
export function useStatsPresetsQuery(api: typeof defaultApi, teamId: string | null): UseQueryResult<StatsPreset[]> {
  return useQuery({
    queryKey: queryKeys.statsPresets(teamId ?? ''),
    queryFn: () => api.statsPrefs.listPresets(teamId!),
    enabled: !!teamId,
  });
}
