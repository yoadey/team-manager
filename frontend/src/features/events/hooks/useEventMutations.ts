import { useCallback } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import type { api as defaultApi } from '@/services';
import { queryKeys } from '@/query/keys';
import type { AttendanceStatus } from '@/types';

type EventPayload = {
  type: string;
  title: string;
  date: string;
  /** Optional last day of a multi-day span; YYYY-MM-DD, or '' to mean "no span" (create: omitted; update: translated to clearMultiDayEndDate). */
  multiDayEndDate?: string | undefined;
  location?: string;
  note?: string;
  meetTimeMandatory?: boolean;
  responseMode?: string;
  meetT?: string;
  startT?: string;
  endT?: string;
  nominatedRoleIds?: string[];
  /** Minutes before the event's start, or undefined for "no cutoff". */
  cancelLeadMinutes?: number | undefined;
  excludeFromStats?: boolean | undefined;
  /** Additional target team ids (besides the owning team). Create: empty/absent = single-team. Update: present replaces the full target set, absent leaves it unchanged. */
  crossTeamIds?: string[] | undefined;
};

/**
 * Invalidates the team's event list and (when given) one event's detail
 * cache, returning a promise that resolves once the invalidated queries have
 * actually refetched -- mutation `onSuccess` handlers below return this
 * promise so `mutateAsync()` (and thus any success toast/sheet-close that
 * follows it) only resolves once the cache is genuinely fresh, not merely
 * once invalidation was requested.
 *
 * The returned function is stable (useCallback) since it's used as a
 * dependency outside this file too (useAbsenceMutations.ts's
 * useSaveAbsenceMutation, since an absence overlapping an upcoming event
 * auto-marks attendance) -- an unmemoized closure here would recreate that
 * callback (and everything depending on it) on every render, breaking the
 * app-wide "actions object identity is stable" invariant.
 */
export function useInvalidateEvents(teamId: string | null) {
  const qc = useQueryClient();
  return useCallback(
    (eventId?: string) => {
      if (!teamId) return Promise.resolve();
      return Promise.all([
        qc.invalidateQueries({ queryKey: queryKeys.events(teamId) }),
        eventId ? qc.invalidateQueries({ queryKey: queryKeys.eventDetail(teamId, eventId) }) : Promise.resolve(),
      ]);
    },
    [qc, teamId],
  );
}

export function useSetAttendanceMutation(api: typeof defaultApi, teamId: string | null) {
  const invalidate = useInvalidateEvents(teamId);
  return useMutation({
    mutationFn: ({
      eventId,
      userId,
      status,
      reason,
    }: {
      eventId: string;
      userId: string;
      status: AttendanceStatus;
      reason?: string;
    }) => api.attendance.set(eventId, userId, { status, reason: reason || '' }, teamId!),
    onSuccess: (_data, { eventId }) => invalidate(eventId),
  });
}

/** Separate from useSetAttendanceMutation so the comment sheet's own pending state
 * doesn't light up while an unrelated RSVP/attendance-grid click is in flight. */
export function useSubmitCommentMutation(api: typeof defaultApi, teamId: string | null) {
  const invalidate = useInvalidateEvents(teamId);
  return useMutation({
    mutationFn: ({
      eventId,
      userId,
      status,
      reason,
    }: {
      eventId: string;
      userId: string;
      status: AttendanceStatus;
      reason: string;
    }) => api.attendance.set(eventId, userId, { status, reason }, teamId!),
    onSuccess: (_data, { eventId }) => invalidate(eventId),
  });
}

export function useSetNominationMutation(api: typeof defaultApi, teamId: string | null) {
  const invalidate = useInvalidateEvents(teamId);
  return useMutation({
    mutationFn: ({ eventId, userId, nominated }: { eventId: string; userId: string; nominated: boolean }) =>
      api.attendance.setNomination(eventId, userId, nominated, teamId!),
    onSuccess: (_data, { eventId }) => invalidate(eventId),
  });
}

export function usePostEventCommentMutation(api: typeof defaultApi, teamId: string | null) {
  const invalidate = useInvalidateEvents(teamId);
  return useMutation({
    mutationFn: ({ eventId, text }: { eventId: string; text: string }) => api.events.addComment(eventId, text, teamId!),
    onSuccess: (_data, { eventId }) => invalidate(eventId),
  });
}

export function useRemoveEventCommentMutation(api: typeof defaultApi, teamId: string | null) {
  const invalidate = useInvalidateEvents(teamId);
  return useMutation({
    mutationFn: ({ eventId, commentId }: { eventId: string; commentId: string }) =>
      api.events.removeComment(commentId, eventId, teamId!),
    onSuccess: (_data, { eventId }) => invalidate(eventId),
  });
}

export type SaveEventArgs =
  | {
      mode: 'create';
      payload: EventPayload & {
        recurring?: boolean | undefined;
        repeatWeeks?: number | undefined;
        endDate?: string | undefined;
      };
    }
  | {
      mode: 'edit';
      eventId: string;
      scope: 'single' | 'series';
      payload: EventPayload;
      /** The additional target teams (besides the owning team) the event
       * had BEFORE this edit -- only meaningful when payload.crossTeamIds
       * is present (a sharing change). Needed so a team REMOVED from
       * sharing also gets its cached event list invalidated, not just a
       * newly added one; the response body only carries the new set. */
      previousCrossTeamIds?: string[] | undefined;
    };

export function useSaveEventMutation(api: typeof defaultApi, teamId: string | null) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (args: SaveEventArgs) =>
      args.mode === 'edit'
        ? api.events.update(args.eventId, args.payload, args.scope, teamId!)
        : api.events.create(teamId!, args.payload),
    // Every team the event is now targeted at, plus (on an edit) every team
    // it was previously targeted at -- a create/edit that adds or removes a
    // cross-team target must invalidate every affected team's cache, not
    // just the team the request was made through (mirrors
    // useEventStatusMutation's identical allTeamIds fan-out for cancel/
    // reactivate).
    onSuccess: (data, args) => {
      const eventId = args.mode === 'edit' ? args.eventId : data.id;
      const currentTeamIds = [data.teamId, ...(data.crossTeamIds ?? [])];
      const previousTeamIds = args.mode === 'edit' ? (args.previousCrossTeamIds ?? []) : [];
      const allTeamIds = Array.from(new Set([...currentTeamIds, ...previousTeamIds]));
      return Promise.all(
        allTeamIds.flatMap((tid) => [
          qc.invalidateQueries({ queryKey: queryKeys.events(tid) }),
          qc.invalidateQueries({ queryKey: queryKeys.eventDetail(tid, eventId) }),
        ]),
      );
    },
  });
}

/**
 * Unlike the other mutations in this file, cancel/reactivate/delete take the
 * event's OWN team id per call rather than the hook-bound active team id: the
 * confirm sheet that triggers these can still be open after the user has
 * switched to a different active team, and the event must still be mutated
 * under the team the request is made through.
 *
 * Cancel/reactivate is deliberately callable through any of a cross-team
 * event's targeted teams, not just its owner (see events.Repository's
 * eventScopedByAnyTargetTeam relaxation for SetStatus) -- so `allTeamIds`
 * (the event's owning team plus every crossTeamIds target) invalidates every
 * targeted team's cached event list/detail, not just the one the request was
 * made through, otherwise the other targeted teams keep showing the event's
 * pre-mutation status until something else happens to refetch them.
 */
export function useEventStatusMutation(api: typeof defaultApi) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({
      eventId,
      status,
      scope,
      teamId,
    }: {
      eventId: string;
      status: 'active' | 'cancelled';
      scope: 'single' | 'series';
      teamId: string;
      allTeamIds: string[];
    }) => api.events.setStatus(eventId, status, scope, teamId),
    onSuccess: (_data, { eventId, allTeamIds }) =>
      Promise.all(
        allTeamIds.flatMap((tid) => [
          qc.invalidateQueries({ queryKey: queryKeys.events(tid) }),
          qc.invalidateQueries({ queryKey: queryKeys.eventDetail(tid, eventId) }),
        ]),
      ),
  });
}

export function useDeleteEventMutation(api: typeof defaultApi) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ eventId, scope, teamId }: { eventId: string; scope: 'single' | 'series'; teamId: string }) =>
      api.events.remove(eventId, scope, teamId),
    onSuccess: (_data, { eventId, teamId }) =>
      Promise.all([
        qc.invalidateQueries({ queryKey: queryKeys.events(teamId) }),
        qc.invalidateQueries({ queryKey: queryKeys.eventDetail(teamId, eventId) }),
      ]),
  });
}
