import { useCallback } from 'react';
import type { api as defaultApi } from '@/services/serviceLayer';
import type { AttendanceRow, TeamEvent } from '../types';
import type { AttendanceStatus, Role, TeamForUser } from '@/types';
import type { AppState } from '@/context/AppContext';
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
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  setFormVal: (patch: Record<string, any>) => void;
  toastMsg: (m: string) => void;
};

export function useEventDetailActions({
  api,
  S,
  setState,
  activeTeam,
  myRoles,
  refreshEvents,
  setFormVal,
  toastMsg,
}: EventFeatureDeps) {
  const reloadDetail = useCallback(
    async (eventId: string) => {
      try {
        const [event, rows, comments] = await Promise.all([
          api.events.get(eventId),
          api.attendance.listForEvent(eventId),
          api.events.listComments(eventId),
        ]);
        setState((s) =>
          s.sheet && s.sheet.type === 'eventDetail' ? { sheet: { ...s.sheet, event, rows, comments } } : {},
        );
      } catch (err) {
        reportActionError({ setState, toastMsg }, err, 'error.load');
      }
    },
    [api, setState, toastMsg],
  );

  const openEventDetail = useCallback(
    async (eventId: string) => {
      setState({ sheet: { type: 'eventDetail', eventId, event: null, rows: [] } });
      await reloadDetail(eventId);
    },
    [setState, reloadDetail],
  );

  const setMyStatus = useCallback(
    async (eventId: string, status: AttendanceStatus) => {
      const ev = S().sheet && S().sheet!.event;
      const keep = ev && ev.id === eventId ? ev.myReason || '' : '';
      try {
        await api.attendance.set(eventId, S().user!.id, { status, reason: keep });
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
        reportActionError({ setState, toastMsg }, err);
      }
    },
    [api, S, refreshEvents, reloadDetail, setState, toastMsg],
  );

  const setStatusFor = useCallback(
    (e: TeamEvent, row: AttendanceRow, status: AttendanceStatus) => {
      void (async () => {
        try {
          await api.attendance.set(e.id, row.userId, { status, reason: row.reason || '' });
          await refreshEvents();
          await reloadDetail(e.id);
        } catch (err) {
          reportActionError({ setState, toastMsg }, err);
        }
      })();
    },
    [api, refreshEvents, reloadDetail, setState, toastMsg],
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
        form: { commentText: row.reason || '' },
      }));
    },
    [setState],
  );

  const submitComment = useCallback(async () => {
    const s = S().sheet!;
    setState({ busy: 'save' });
    try {
      await api.attendance.set(s.eventId!, s.userId!, {
        status: s.status!,
        reason: (S().form.commentText as string) || '',
      });
      await refreshEvents();
      const eid = s.eventId!;
      setState({ busy: null, sheet: null });
      openEventDetail(eid);
      toastMsg(t('events.toastCommentSaved'));
    } catch (err) {
      reportActionError({ setState, toastMsg }, err, 'error.save');
    }
  }, [api, S, setState, refreshEvents, openEventDetail, toastMsg]);

  const postEventComment = useCallback(
    async (eventId: string) => {
      const txt = (S().form.newEventComment || '').trim();
      if (!txt) return;
      try {
        await api.events.addComment(eventId, txt);
        setFormVal({ newEventComment: '' });
        await reloadDetail(eventId);
      } catch (err) {
        reportActionError({ setState, toastMsg }, err, 'error.save');
      }
    },
    [api, S, setFormVal, reloadDetail, setState, toastMsg],
  );

  const removeEventComment = useCallback(
    async (eventId: string, cid: string) => {
      try {
        await api.events.removeComment(cid);
        await reloadDetail(eventId);
      } catch (err) {
        reportActionError({ setState, toastMsg }, err, 'error.delete');
      }
    },
    [api, reloadDetail, setState, toastMsg],
  );

  const toggleNomination = useCallback(
    async (eventId: string, userId: string, currentlyNominated: boolean) => {
      try {
        await api.attendance.setNomination(eventId, userId, !currentlyNominated);
        await refreshEvents();
        await reloadDetail(eventId);
        toastMsg(currentlyNominated ? t('attendance.not_nominated') : t('attendance.nominated'));
      } catch (err) {
        reportActionError({ setState, toastMsg }, err);
      }
    },
    [api, refreshEvents, reloadDetail, setState, toastMsg],
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
  setState,
  askConfirm,
  refreshEvents,
  openEventDetail,
  toastMsg,
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
              await api.events.remove(event.id, scope);
              await refreshEvents();
              setState({ sheet: null });
              toastMsg(scope === 'series' ? t('events.toastSeriesDeleted') : t('events.toastEventDeleted'));
            } catch (err) {
              reportActionError({ setState, toastMsg }, err, 'error.delete');
            }
          },
        });
        return;
      }
      const status = action === 'cancel' ? 'cancelled' : 'active';
      try {
        await api.events.setStatus(event.id, status, scope);
        await refreshEvents();
        setState({ sheet: null });
        openEventDetail(event.id);
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
        reportActionError({ setState, toastMsg }, err);
      }
    },
    [api, askConfirm, refreshEvents, setState, openEventDetail, toastMsg],
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
