import { useCallback } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import type { api as defaultApi } from '@/services';
import { queryKeys } from '@/query/keys';
import type { AttendanceStatus } from '@/types';

type EventPayload = {
  type: string;
  title: string;
  date: string;
  location?: string;
  note?: string;
  meetTimeMandatory?: boolean;
  responseMode?: string;
  meetT?: string;
  startT?: string;
  endT?: string;
  nominatedRoleIds?: string[];
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
  | { mode: 'create'; payload: EventPayload & { recurring?: boolean; repeatWeeks?: number } }
  | { mode: 'edit'; eventId: string; scope: 'single' | 'series'; payload: EventPayload };

export function useSaveEventMutation(api: typeof defaultApi, teamId: string | null) {
  const invalidate = useInvalidateEvents(teamId);
  return useMutation({
    mutationFn: (args: SaveEventArgs) =>
      args.mode === 'edit'
        ? api.events.update(args.eventId, args.payload, args.scope, teamId!)
        : api.events.create(teamId!, args.payload),
    onSuccess: (_data, args) => invalidate(args.mode === 'edit' ? args.eventId : undefined),
  });
}

/**
 * Unlike the other mutations in this file, cancel/reactivate/delete take the
 * event's OWN team id per call rather than the hook-bound active team id: the
 * confirm sheet that triggers these can still be open after the user has
 * switched to a different active team, and the event must still be mutated
 * (and its cache invalidated) under the team it actually belongs to.
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
    }) => api.events.setStatus(eventId, status, scope, teamId),
    onSuccess: (_data, { eventId, teamId }) =>
      Promise.all([
        qc.invalidateQueries({ queryKey: queryKeys.events(teamId) }),
        qc.invalidateQueries({ queryKey: queryKeys.eventDetail(teamId, eventId) }),
      ]),
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
