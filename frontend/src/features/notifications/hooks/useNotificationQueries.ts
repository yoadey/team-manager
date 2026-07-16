import { useQuery, type UseQueryResult } from '@tanstack/react-query';
import type { api as defaultApi } from '@/services';
import { queryKeys } from '@/query/keys';
import type { NotificationsResult } from '../types';

/** The team's notification feed + unread count, team-scoped so a team switch swaps the cache entry instead of racing. */
export function useNotificationsQuery(
  api: typeof defaultApi,
  teamId: string | null,
): UseQueryResult<NotificationsResult> {
  return useQuery({
    queryKey: queryKeys.notifications(teamId ?? ''),
    queryFn: () => api.notifications.list(teamId!),
    enabled: !!teamId,
  });
}
