import { useCallback, useRef } from 'react';
import type { api as defaultApi } from '@/services';
import type { AttendanceRow, AttendanceCommentFormValues, EventCommentFormValues, TeamEvent } from '../types';
import type { AttendanceStatus, Role, TeamForUser } from '@/types';
import type { AppState } from '@/context/AppContext';
import { formValues } from '@/utils/forms';
import { canSeeReason } from '@/utils/permissions';
import { reportActionError } from '@/utils/errors';
import { t } from '@/i18n';
import {
  usePostEventCommentMutation,
  useRemoveEventCommentMutation,
  useSetAttendanceMutation,
  useSetNominationMutation,
  useSubmitCommentMutation,
  useEventStatusMutation,
  useDeleteEventMutation,
} from './useEventMutations';

type SetState = (patch: Partial<AppState> | ((s: AppState) => Partial<AppState>)) => void;

type EventFeatureDeps = {
  api: typeof defaultApi;
  S: () => AppState;
  setState: SetState;
  activeTeam: () => TeamForUser | null;
  myRoles: () => Role[];
  /** Reactive (render-time) active team id -- query/mutation hooks key off this directly
   * rather than through `S()`, since a `useQuery`/`useMutation` call must re-run on every
   * render to pick up a team switch instead of only when some later callback fires. */
  teamId: string | null;
  /** A successful attendance/nomination/status change can flip a notification's
   * read-worthy state (e.g. a "pending RSVP" reminder), so each such mutation also
   * refreshes the notification badge, mirroring the pre-migration refreshEvents(). */
  loadNotifications: () => Promise<void>;
  setFormVal: (patch: Record<string, unknown>) => void;
  askConfirm: (cfg: {
    title: string;
    message: string;
    confirmLabel?: string;
    danger?: boolean;
    onConfirm: () => void | Promise<void>;
  }) => void;
  toastMsg: (m: string, action?: { label: string; fn: () => void }, kind?: 'success' | 'error') => void;
  logout: () => void;
};

export function useEventDetailActions({
  api,
  S,
  setState,
  activeTeam,
  myRoles,
  teamId,
  loadNotifications,
  setFormVal,
  askConfirm,
  toastMsg,
  logout,
}: EventFeatureDeps) {
  // `mutateAsync` is a stable reference for the observer's lifetime (unlike
  // the mutation result object itself, which is a fresh snapshot every
  // render) -- destructuring it lets the useCallbacks below depend on a
  // plain stable identifier instead of the whole snapshot, keeping their own
  // identity stable across renders too (see AppContext.tsx's "actions object
  // identity" invariant).
  const { mutateAsync: setAttendanceAsync } = useSetAttendanceMutation(api, teamId);
  const { mutateAsync: submitCommentAsync, isPending: savingComment } = useSubmitCommentMutation(api, teamId);
  const { mutateAsync: setNominationAsync } = useSetNominationMutation(api, teamId);
  const { mutateAsync: postCommentAsync } = usePostEventCommentMutation(api, teamId);
  const { mutateAsync: removeCommentAsync } = useRemoveEventCommentMutation(api, teamId);

  const openEventDetail = useCallback(
    (eventId: string) => {
      setState({ sheet: { type: 'eventDetail', eventId } });
    },
    [setState],
  );

  const inFlight = useRef(new Set<string>());

  const setMyStatus = useCallback(
    async (eventId: string, status: AttendanceStatus, currentReason?: string) => {
      const key = `${eventId}:${S().user!.id}`;
      if (inFlight.current.has(key)) return;
      inFlight.current.add(key);
      try {
        await setAttendanceAsync({ eventId, userId: S().user!.id, status, reason: currentReason || '' });
        loadNotifications();
        toastMsg(
          status === 'yes'
            ? t('attendance.yes')
            : status === 'maybe'
              ? t('events.toastStatusMaybe')
              : t('attendance.no'),
        );
      } catch (err) {
        reportActionError({ setState, toastMsg, onAuthError: logout }, err);
      } finally {
        inFlight.current.delete(key);
      }
    },
    [S, setAttendanceAsync, setState, loadNotifications, toastMsg, logout],
  );

  const setStatusFor = useCallback(
    (e: TeamEvent, row: AttendanceRow, status: AttendanceStatus) => {
      const key = `${e.id}:${row.userId}`;
      if (inFlight.current.has(key)) return;
      inFlight.current.add(key);
      void (async () => {
        try {
          await setAttendanceAsync({ eventId: e.id, userId: row.userId, status, reason: row.reason || '' });
          loadNotifications();
        } catch (err) {
          reportActionError({ setState, toastMsg, onAuthError: logout }, err);
        } finally {
          inFlight.current.delete(key);
        }
      })();
    },
    [setAttendanceAsync, setState, loadNotifications, toastMsg, logout],
  );

  const canSeeComment = useCallback(
    (row: AttendanceRow) =>
      canSeeReason({
        isSelf: row.userId === S().user!.id,
        reason: row.reason,
        status: row.status,
        reasonVisibilityRoles: (activeTeam() && activeTeam()!.reasonVisibilityRoles) || [],
        myRoleIds: myRoles().map((r) => r.id),
      }),
    [S, activeTeam, myRoles],
  );

  const openComment = useCallback(
    (e: TeamEvent, row: { userId: string; name: string; status: AttendanceStatus; reason?: string }) => {
      setState((st) => ({
        sheet: {
          type: 'comment',
          eventId: e.id,
          userId: row.userId,
          name: row.name,
          status: row.status,
          back: st.sheet,
        },
        form: { commentText: row.reason || '' } satisfies AttendanceCommentFormValues,
      }));
    },
    [setState],
  );

  const submitComment = useCallback(async () => {
    const s = S().sheet!;
    try {
      await submitCommentAsync({
        eventId: s.eventId!,
        userId: s.userId!,
        status: s.status!,
        reason: formValues<AttendanceCommentFormValues>(S()).commentText || '',
      });
      loadNotifications();
      const eid = s.eventId!;
      // Don't close/reopen a sheet the user has since opened for a different
      // team after switching away mid-request, or one they've since opened
      // while this save was in flight.
      if (S().activeTeamId === teamId && S().sheet === s) {
        setState({ sheet: null });
        openEventDetail(eid);
      }
      toastMsg(t('events.toastCommentSaved'));
    } catch (err) {
      reportActionError({ setState, toastMsg, onAuthError: logout }, err, 'error.save');
    }
  }, [S, setState, submitCommentAsync, teamId, openEventDetail, loadNotifications, toastMsg, logout]);

  const commentInFlight = useRef(new Set<string>());

  const postEventComment = useCallback(
    async (eventId: string) => {
      const txt = (formValues<EventCommentFormValues>(S()).newEventComment || '').trim();
      if (!txt) return;
      if (commentInFlight.current.has(eventId)) return;
      commentInFlight.current.add(eventId);
      try {
        await postCommentAsync({ eventId, text: txt });
        setFormVal({ newEventComment: '' });
      } catch (err) {
        reportActionError({ setState, toastMsg, onAuthError: logout }, err, 'error.save');
      } finally {
        commentInFlight.current.delete(eventId);
      }
    },
    [S, postCommentAsync, setFormVal, setState, toastMsg, logout],
  );

  const removeEventComment = useCallback(
    (eventId: string, cid: string) =>
      askConfirm({
        title: t('events.deleteCommentTitle'),
        message: t('events.deleteCommentMsg'),
        confirmLabel: t('common.delete'),
        danger: true,
        onConfirm: async () => {
          try {
            await removeCommentAsync({ eventId, commentId: cid });
            toastMsg(t('events.toastCommentDeleted'));
          } catch (err) {
            reportActionError({ setState, toastMsg, onAuthError: logout }, err, 'error.delete');
          }
        },
      }),
    [askConfirm, removeCommentAsync, setState, toastMsg, logout],
  );

  const toggleNomination = useCallback(
    async (eventId: string, userId: string, currentlyNominated: boolean) => {
      const key = `${eventId}:${userId}`;
      if (inFlight.current.has(key)) return;
      inFlight.current.add(key);
      try {
        await setNominationAsync({ eventId, userId, nominated: !currentlyNominated });
        loadNotifications();
        toastMsg(currentlyNominated ? t('attendance.not_nominated') : t('attendance.nominated'));
      } catch (err) {
        reportActionError({ setState, toastMsg, onAuthError: logout }, err);
      } finally {
        inFlight.current.delete(key);
      }
    },
    [setNominationAsync, setState, loadNotifications, toastMsg, logout],
  );

  return {
    openEventDetail,
    setMyStatus,
    setStatusFor,
    canSeeComment,
    openComment,
    submitComment,
    postEventComment,
    removeEventComment,
    toggleNomination,
    savingComment,
  };
}

export type EventActionDeps = EventFeatureDeps & {
  askConfirm: (cfg: {
    title: string;
    message: string;
    confirmLabel?: string;
    danger?: boolean;
    onConfirm: () => void | Promise<void>;
  }) => void;
  openEventDetail: (eventId: string) => void;
};

export function useEventActionFeatures({
  api,
  S,
  setState,
  askConfirm,
  teamId,
  loadNotifications,
  openEventDetail,
  toastMsg,
  logout,
}: EventActionDeps) {
  const { mutateAsync: setEventStatusAsync } = useEventStatusMutation(api, teamId);
  const { mutateAsync: deleteEventAsync } = useDeleteEventMutation(api, teamId);

  const runEventAction = useCallback(
    async (action: 'cancel' | 'delete' | 'reactivate', event: TeamEvent, scope: 'single' | 'series') => {
      if (action === 'delete') {
        askConfirm({
          title: scope === 'series' ? t('events.deleteSeriesTitle') : t('events.deleteEventTitle'),
          message:
            scope === 'series' ? t('events.deleteSeriesMsg') : t('events.deleteEventMsg', { title: event.title }),
          confirmLabel: t('common.delete'),
          danger: true,
          onConfirm: async () => {
            const sh = S().sheet;
            try {
              await deleteEventAsync({ eventId: event.id, scope });
              loadNotifications();
              // Don't close a sheet the user has since opened for a different
              // team after switching away mid-request, or one they've since
              // opened for a different event (same team) while this delete
              // was in flight.
              if (S().activeTeamId === event.teamId && S().sheet === sh) setState({ sheet: null });
              toastMsg(scope === 'series' ? t('events.toastSeriesDeleted') : t('events.toastEventDeleted'));
            } catch (err) {
              reportActionError({ setState, toastMsg, onAuthError: logout }, err, 'error.delete');
            }
          },
        });
        return;
      }
      const sh = S().sheet;
      const status = action === 'cancel' ? 'cancelled' : 'active';
      try {
        await setEventStatusAsync({ eventId: event.id, status, scope });
        loadNotifications();
        // Don't close/reopen a sheet the user has since opened for a different
        // team after switching away mid-request, or one they've since opened
        // for a different event (same team) while this cancel/reactivate was
        // in flight.
        if (S().activeTeamId === event.teamId && S().sheet === sh) {
          setState({ sheet: null });
          openEventDetail(event.id);
        }
        toastMsg(
          action === 'cancel'
            ? scope === 'series'
              ? t('events.toastSeriesCancelled')
              : t('events.toastEventCancelled')
            : scope === 'series'
              ? t('events.toastSeriesActivated')
              : t('events.toastEventActivated'),
        );
      } catch (err) {
        reportActionError({ setState, toastMsg, onAuthError: logout }, err);
      }
    },
    [S, askConfirm, deleteEventAsync, setEventStatusAsync, setState, openEventDetail, loadNotifications, toastMsg, logout],
  );

  const askEventAction = useCallback(
    (action: 'cancel' | 'delete' | 'reactivate', event: TeamEvent) => {
      if (event.seriesId) setState((st) => ({ sheet: { type: 'seriesAction', action, event, back: st.sheet } }));
      else runEventAction(action, event, 'single');
    },
    [setState, runEventAction],
  );

  return { askEventAction, runEventAction };
}
