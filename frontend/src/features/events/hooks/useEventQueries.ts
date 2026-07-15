import { useQuery, type UseQueryResult } from '@tanstack/react-query';
import type { api as defaultApi } from '@/services';
import { queryKeys } from '@/query/keys';
import type { AttendanceRow, EventComment, TeamEvent } from '../types';

/** The team's event list, team-scoped so a team switch swaps the cache entry instead of racing. */
export function useEventsQuery(api: typeof defaultApi, teamId: string | null): UseQueryResult<TeamEvent[]> {
  return useQuery({
    queryKey: queryKeys.events(teamId ?? ''),
    queryFn: () => api.events.list(teamId!, 'all'),
    enabled: !!teamId,
  });
}

export interface EventDetailData {
  event: TeamEvent | null;
  rows: AttendanceRow[];
  comments: EventComment[];
}

/** Event detail sheet data (event + attendance rows + comment thread) as one query. */
export function useEventDetailQuery(
  api: typeof defaultApi,
  teamId: string | null,
  eventId: string | null,
): UseQueryResult<EventDetailData> {
  return useQuery({
    queryKey: queryKeys.eventDetail(teamId ?? '', eventId ?? ''),
    queryFn: async (): Promise<EventDetailData> => {
      const [event, rows, comments] = await Promise.all([
        api.events.get(eventId!, teamId!),
        api.attendance.listForEvent(eventId!, teamId!),
        api.events.listComments(eventId!, teamId!),
      ]);
      return { event, rows, comments };
    },
    enabled: !!teamId && !!eventId,
  });
}
