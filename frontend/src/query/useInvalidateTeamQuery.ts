import { useCallback } from 'react';
import { useQueryClient } from '@tanstack/react-query';

/**
 * Returns a function that invalidates the given team-scoped query key,
 * resolving once the invalidated query has refetched. A no-op (resolved
 * promise) when there's no active team.
 */
export function useInvalidateTeamQuery(teamId: string | null, queryKey: (teamId: string) => readonly unknown[]) {
  const qc = useQueryClient();
  return useCallback(() => {
    if (!teamId) return Promise.resolve();
    return qc.invalidateQueries({ queryKey: queryKey(teamId) });
  }, [qc, teamId, queryKey]);
}
