import { useMutation, useQueryClient } from '@tanstack/react-query';
import type { api as defaultApi } from '@/services';
import { queryKeys } from '@/query/keys';
import { useInvalidateTeamQuery } from '@/query/useInvalidateTeamQuery';
import { useInvalidateEvents } from './useEventMutations';

export interface SaveAbsenceInput {
  // Explicit `| undefined` -- callers (useAbsenceActions.ts) pass
  // `mode === 'edit' ? f.id : undefined` to select update-vs-create, so
  // `undefined` is a meaningful value here, not just "field omitted".
  id?: string | undefined;
  payload: { from: string; to: string; reason: string };
  /** Required when creating (no `id`); ignored for an update. */
  userId?: string;
}

/**
 * An absence overlapping an upcoming event auto-marks attendance for that
 * event, so every save/delete also invalidates the events cache alongside
 * this vertical's own -- folding in what the pre-migration refreshEvents()
 * bridge (useFeatureActions.ts) used to do, now that absences owns its own
 * query cache instead of a manual loader.
 */
export function useSaveAbsenceMutation(api: typeof defaultApi, teamId: string | null) {
  const invalidateAbsences = useInvalidateTeamQuery(teamId, queryKeys.absences);
  const invalidateEvents = useInvalidateEvents(teamId);
  return useMutation({
    mutationFn: ({ id, payload, userId }: SaveAbsenceInput) =>
      id
        ? api.absences.update(id, payload, teamId!)
        : api.absences.create({ teamId: teamId!, userId: userId!, ...payload }),
    onSuccess: () => Promise.all([invalidateAbsences(), invalidateEvents()]),
  });
}

// Takes the team id per call rather than the hook-bound active team id,
// mirroring useDeleteEventMutation/useRemoveMemberMutation -- the confirm
// sheet that triggers this can still be open (and get confirmed) after the
// user has switched to a different active team.
export function useDeleteAbsenceMutation(api: typeof defaultApi) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, teamId }: { id: string; teamId: string }) => api.absences.remove(id, teamId),
    onSuccess: (_data, { teamId }) =>
      Promise.all([
        qc.invalidateQueries({ queryKey: queryKeys.absences(teamId) }),
        qc.invalidateQueries({ queryKey: queryKeys.events(teamId) }),
      ]),
  });
}
