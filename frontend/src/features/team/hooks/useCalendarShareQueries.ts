import { useQuery, type UseQueryResult } from '@tanstack/react-query';
import type { api as defaultApi } from '@/services';
import type { CalendarShare, SharedCalendarSource, SharedCalendarEvent } from '@/types';
import { queryKeys } from '@/query/keys';

/** Teams this team has granted calendar visibility to (owner-team perspective). */
export function useCalendarSharesQuery(api: typeof defaultApi, teamId: string | null): UseQueryResult<CalendarShare[]> {
  return useQuery({
    queryKey: queryKeys.calendarShares(teamId ?? ''),
    queryFn: () => api.calendarShares.list(teamId!),
    enabled: !!teamId,
  });
}

/** Teams that have granted this team calendar visibility (viewer-team perspective). */
export function useSharedCalendarSourcesQuery(
  api: typeof defaultApi,
  teamId: string | null,
): UseQueryResult<SharedCalendarSource[]> {
  return useQuery({
    queryKey: queryKeys.sharedCalendarSources(teamId ?? ''),
    queryFn: () => api.sharedCalendars.listSources(teamId!),
    enabled: !!teamId,
  });
}

/** ownerTeamId's redacted schedule, as read through the calendar share. */
export function useSharedCalendarEventsQuery(
  api: typeof defaultApi,
  teamId: string | null,
  ownerTeamId: string | null,
): UseQueryResult<SharedCalendarEvent[]> {
  return useQuery({
    queryKey: queryKeys.sharedCalendarEvents(teamId ?? '', ownerTeamId ?? ''),
    queryFn: () => api.sharedCalendars.listEvents(teamId!, ownerTeamId!),
    enabled: !!teamId && !!ownerTeamId,
  });
}
