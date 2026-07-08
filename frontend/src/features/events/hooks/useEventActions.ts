import { useCallback, useRef } from 'react';
import type { api as defaultApi } from '@/services/serviceLayer';
import type { AttendanceRow, AttendanceCommentFormValues, EventCommentFormValues, TeamEvent } from '../types';
import type { AttendanceStatus, Role, TeamForUser } from '@/types';
import type { AppState } from '@/context/AppContext';
import { formValues } from '@/utils/forms';
import { canSeeReason } from '@/utils/permissions';
import { reportActionError } from '@/utils/errors';
import { t } from '@/i18n';

type SetState = (patch: Partial<AppState> | ((s: AppState) => Partial<AppState>)) => void;

type EventFeatureDeps = {
  api: typeof defaultApi;
  S: () => AppState;
  setState: SetState;
  activeTeam: () => TeamForUser | null;
  myRoles: () => Role[];
  refreshEvents: () => Promise<void>;
  setFormVal: (patch: Record<string, unknown>) => void;
  askConfirm: (cfg: {
    title: string;
    message: string;
    confirmLabel?: string;
    danger?: boolean;
    onConfirm: () => void | Promise<void>;
  }) => void;
  toastMsg: (m: string) => void;
  logout: () => void;
};

export function useEventDetailActions({
  api,
  S,
  setState,
  activeTeam,
  myRoles,
  refreshEvents,
  setFormVal,
  askConfirm,
  toastMsg,
  logout,
}: EventFeatureDeps) {
  const reloadDetail = useCallback(
    async (eventId: string) => {
      try {
        const teamId = S().activeTeamId!;
        const [event, rows, comments] = await Promise.all([
          api.events.get(eventId, teamId),
          api.attendance.listForEvent(eventId, teamId),
          api.events.listComments(eventId, teamId),
        ]);
        setState((s) =>
          s.sheet && s.sheet.type === 'eventDetail' && s.sheet.eventId === eventId
            ? { sheet: { ...s.sheet, event, rows, comments } }
            : {},
        );
      } catch (err) {
        reportActionError({ setState, toastMsg, onAuthError: logout }, err, 'error.load');
      }
    },
    [api, S, setState, toastMsg, logout],
  );

  const openEventDetail = useCallback(
    async (eventId: string) => {
      setState({ sheet: { type: 'eventDetail', eventId, event: null, rows: [] } });
      await reloadDetail(eventId);
    },
    [setState, reloadDetail],
  );

  const inFlight = useRef(new Set<string>());

  const setMyStatus = useCallback(
    async (eventId: string, status: AttendanceStatus) => {
      const key = `${eventId}:${S().user!.id}`;
      if (inFlight.current.has(key)) return;
      inFlight.current.add(key);
      const ev = S().sheet && S().sheet!.event;
      const keep = ev && ev.id === eventId ? ev.myReason || '' : '';
      try {
        await api.attendance.set(eventId, S().user!.id, { status, reason: keep }, S().activeTeamId!);
        await refreshEvents();
        if (S().sheet && S().sheet!.type === 'eventDetail' && S().sheet!.eventId === eventId)
          await reloadDetail(eventId);
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
    [api, S, refreshEvents, reloadDetail, setState, toastMsg, logout],
  );

  const setStatusFor = useCallback(
    (e: TeamEvent, row: AttendanceRow, status: AttendanceStatus) => {
      const key = `${e.id}:${row.userId}`;
      if (inFlight.current.has(key)) return;
      inFlight.current.add(key);
      void (async () => {
        try {
          await api.attendance.set(e.id, row.userId, { status, reason: row.reason || '' }, S().activeTeamId!);
          await refreshEvents();
          await reloadDetail(e.id);
        } catch (err) {
          reportActionError({ setState, toastMsg, onAuthError: logout }, err);
        } finally {
          inFlight.current.delete(key);
        }
      })();
    },
    [api, S, refreshEvents, reloadDetail, setState, toastMsg, logout],
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
    const teamId = S().activeTeamId!;
    setState({ busy: 'save' });
    try {
      await api.attendance.set(
        s.eventId!,
        s.userId!,
        {
          status: s.status!,
          reason: formValues<AttendanceCommentFormValues>(S()).commentText || '',
        },
        teamId,
      );
      await refreshEvents();
      const eid = s.eventId!;
      setState({ busy: null });
      // Don't close/reopen a sheet the user has since opened for a
      // different team after switching away mid-request.
      if (S().activeTeamId === teamId) {
        setState({ sheet: null });
        openEventDetail(eid);
      }
      toastMsg(t('events.toastCommentSaved'));
    } catch (err) {
      reportActionError({ setState, toastMsg, onAuthError: logout }, err, 'error.save');
    }
  }, [api, S, setState, refreshEvents, openEventDetail, toastMsg, logout]);

  const commentInFlight = useRef(new Set<string>());

  const postEventComment = useCallback(
    async (eventId: string) => {
      const txt = (formValues<EventCommentFormValues>(S()).newEventComment || '').trim();
      if (!txt) return;
      if (commentInFlight.current.has(eventId)) return;
      commentInFlight.current.add(eventId);
      try {
        await api.events.addComment(eventId, txt, S().activeTeamId!);
        setFormVal({ newEventComment: '' });
        await reloadDetail(eventId);
      } catch (err) {
        reportActionError({ setState, toastMsg, onAuthError: logout }, err, 'error.save');
      } finally {
        commentInFlight.current.delete(eventId);
      }
    },
    [api, S, setFormVal, reloadDetail, setState, toastMsg, logout],
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
            await api.events.removeComment(cid, eventId, S().activeTeamId!);
            await reloadDetail(eventId);
            toastMsg(t('events.toastCommentDeleted'));
          } catch (err) {
            reportActionError({ setState, toastMsg, onAuthError: logout }, err, 'error.delete');
          }
        },
      }),
    [api, S, askConfirm, reloadDetail, setState, toastMsg, logout],
  );

  const toggleNomination = useCallback(
    async (eventId: string, userId: string, currentlyNominated: boolean) => {
      try {
        await api.attendance.setNomination(eventId, userId, !currentlyNominated, S().activeTeamId!);
        await refreshEvents();
        await reloadDetail(eventId);
        toastMsg(currentlyNominated ? t('attendance.not_nominated') : t('attendance.nominated'));
      } catch (err) {
        reportActionError({ setState, toastMsg, onAuthError: logout }, err);
      }
    },
    [api, S, refreshEvents, reloadDetail, setState, toastMsg, logout],
  );

  return {
    reloadDetail,
    openEventDetail,
    setMyStatus,
    setStatusFor,
    canSeeComment,
    openComment,
    submitComment,
    postEventComment,
    removeEventComment,
    toggleNomination,
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
  openEventDetail: (eventId: string) => Promise<void>;
};

export function useEventActionFeatures({
  api,
  S,
  setState,
  askConfirm,
  refreshEvents,
  openEventDetail,
  toastMsg,
  logout,
}: EventActionDeps) {
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
            try {
              await api.events.remove(event.id, scope, event.teamId);
              await refreshEvents();
              // Don't close a sheet the user has since opened for a
              // different team after switching away mid-request.
              if (S().activeTeamId === event.teamId) setState({ sheet: null });
              toastMsg(scope === 'series' ? t('events.toastSeriesDeleted') : t('events.toastEventDeleted'));
            } catch (err) {
              reportActionError({ setState, toastMsg, onAuthError: logout }, err, 'error.delete');
            }
          },
        });
        return;
      }
      const status = action === 'cancel' ? 'cancelled' : 'active';
      try {
        await api.events.setStatus(event.id, status, scope, event.teamId);
        await refreshEvents();
        // Don't close/reopen a sheet the user has since opened for a
        // different team after switching away mid-request.
        if (S().activeTeamId === event.teamId) {
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
    [api, S, askConfirm, refreshEvents, setState, openEventDetail, toastMsg, logout],
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
