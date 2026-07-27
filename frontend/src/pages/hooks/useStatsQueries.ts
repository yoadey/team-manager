import { useQuery, type UseQueryResult } from '@tanstack/react-query';
import type { api as defaultApi } from '@/services';
import { queryKeys } from '@/query/keys';
import type { AttendanceAbsenceTable, AttendanceMatrix, DateRange, StatsOverview } from '@/types';

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

/**
 * The per-member-per-event attendance matrix for the selected range. Kept as a
 * separate query (own key) from useStatsQuery so the Matrix tab lazy-loads:
 * `enabled` gates the fetch on the tab actually being open, and a team/range
 * change swaps the cache entry just like the overview.
 */
export function useAttendanceMatrixQuery(
  api: typeof defaultApi,
  teamId: string | null,
  range: DateRange | null,
  enabled = true,
): UseQueryResult<AttendanceMatrix> {
  return useQuery({
    queryKey: queryKeys.statsMatrix(teamId ?? '', range),
    queryFn: () => api.stats.attendanceMatrix(teamId!, range),
    enabled: !!teamId && enabled,
  });
}

/**
 * The team's absence table (member/event/date rows) for the selected date
 * range -- the same team-and-range-scoped cache-key shape as useStatsQuery,
 * so the absence tab and the quota view swap in lockstep on a range change.
 */
export function useAbsenceTableQuery(
  api: typeof defaultApi,
  teamId: string | null,
  range: DateRange | null,
): UseQueryResult<AttendanceAbsenceTable> {
  return useQuery({
    queryKey: queryKeys.statsAbsences(teamId ?? '', range),
    queryFn: () => api.stats.absenceTable(teamId!, range),
    enabled: !!teamId,
  });
}
