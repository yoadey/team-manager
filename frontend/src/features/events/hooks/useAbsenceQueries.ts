import { useQuery, type UseQueryResult } from '@tanstack/react-query';
import type { api as defaultApi } from '@/services';
import { queryKeys } from '@/query/keys';
import type { Absence } from '../types';

/**
 * The team's planned absences, team-scoped so a team switch swaps the cache
 * entry instead of racing. `enabled` additionally gates the fetch on whatever
 * on-demand condition the caller has (EventCalendar only wants this while its
 * "show absences" overlay is on; EventAbsences wants it whenever it's
 * mounted, i.e. always, since RouteScreen only mounts it for the Absences
 * tab) -- mirroring the pre-migration loadAbsences() being triggered on-demand
 * rather than eagerly on every team switch.
 */
export function useAbsencesQuery(
  api: typeof defaultApi,
  teamId: string | null,
  enabled = true,
): UseQueryResult<Absence[]> {
  return useQuery({
    queryKey: queryKeys.absences(teamId ?? ''),
    queryFn: () => api.absences.listForTeam(teamId!),
    enabled: !!teamId && enabled,
  });
}
