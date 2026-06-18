/* eslint-disable @typescript-eslint/no-explicit-any */
import { useCallback } from 'react';
import type { api as defaultApi } from '@/services/serviceLayer';
import type { AttendanceRow, TeamEvent } from '../types';
import type { AttendanceStatus, ModuleKey, PermLevel, Role, TeamForUser, User } from '@/types';
import type { AppState } from '@/context/AppContext';

type SetState = (patch: Partial<AppState> | ((s: AppState) => Partial<AppState>)) => void;

type EventFeatureDeps = {
  api: typeof defaultApi;
  S: () => AppState;
  setState: SetState;
  activeTeam: () => TeamForUser | null;
  myRoles: () => Role[];
  refreshEvents: () => Promise<void>;
  setFormVal: (patch: Record<string, any>) => void;
  toastMsg: (m: string) => void;
};

export function useEventDetailActions({ api, S, setState, activeTeam, myRoles, refreshEvents, setFormVal, toastMsg }: EventFeatureDeps) {
  const reloadDetail = useCallback(async (eventId: string) => {
    const [event, rows, comments] = await Promise.all([api.events.get(eventId), api.attendance.listForEvent(eventId), api.events.listComments(eventId)]);
    setState((s) => (s.sheet && s.sheet.type === 'eventDetail') ? { sheet: { ...s.sheet, event, rows, comments } } : {});
  }, [api, setState]);

  const openEventDetail = useCallback(async (eventId: string) => { setState({ sheet: { type: 'eventDetail', eventId, event: null, rows: [] } }); await reloadDetail(eventId); }, [setState, reloadDetail]);

  const setMyStatus = useCallback(async (eventId: string, status: AttendanceStatus) => {
    const ev = S().sheet && S().sheet!.event;
    const keep = (ev && ev.id === eventId) ? (ev.myReason || '') : '';
    await api.attendance.set(eventId, S().user!.id, { status, reason: keep });
    await refreshEvents();
    if (S().sheet && S().sheet!.type === 'eventDetail' && S().sheet!.eventId === eventId) await reloadDetail(eventId);
    toastMsg(status === 'yes' ? 'Zugesagt' : (status === 'maybe' ? 'Als unsicher markiert' : 'Abgesagt'));
  }, [api, S, refreshEvents, reloadDetail, toastMsg]);

  const setStatusFor = useCallback((e: TeamEvent, row: AttendanceRow, status: AttendanceStatus) => {
    (async () => { await api.attendance.set(e.id, row.userId, { status, reason: row.reason || '' }); await refreshEvents(); await reloadDetail(e.id); })();
  }, [api, refreshEvents, reloadDetail]);

  const canSeeComment = useCallback((row: AttendanceRow) => {
    if (row.userId === S().user!.id) return true;
    if (!row.reason) return false;
    if (row.status !== 'no') return true;
    const vis = (activeTeam() && activeTeam()!.reasonVisibilityRoles) || [];
    const myIds = myRoles().map((r) => r.id);
    return myIds.some((id) => vis.includes(id));
  }, [S, activeTeam, myRoles]);

  const openComment = useCallback((e: TeamEvent, row: { userId: string; name: string; status: AttendanceStatus; reason?: string }) => {
    setState((st) => ({ sheet: { type: 'comment', eventId: e.id, userId: row.userId, name: row.name, status: row.status, back: st.sheet }, form: { commentText: row.reason || '' } }));
  }, [setState]);

  const submitComment = useCallback(async () => {
    const s = S().sheet!;
    setState({ busy: 'save' });
    await api.attendance.set(s.eventId, s.userId, { status: s.status, reason: S().form.commentText || '' });
    await refreshEvents();
    const eid = s.eventId;
    setState({ busy: null, sheet: null });
    openEventDetail(eid);
    toastMsg('Kommentar gespeichert');
  }, [api, S, setState, refreshEvents, openEventDetail, toastMsg]);

  const postEventComment = useCallback(async (eventId: string) => {
    const txt = (S().form.newEventComment || '').trim();
    if (!txt) return;
    await api.events.addComment(eventId, txt);
    setFormVal({ newEventComment: '' });
    await reloadDetail(eventId);
  }, [api, S, setFormVal, reloadDetail]);

  const removeEventComment = useCallback(async (eventId: string, cid: string) => { await api.events.removeComment(cid); await reloadDetail(eventId); }, [api, reloadDetail]);

  const toggleNomination = useCallback(async (eventId: string, userId: string, currentlyNominated: boolean) => {
    await api.attendance.setNomination(eventId, userId, !currentlyNominated);
    await refreshEvents(); await reloadDetail(eventId);
    toastMsg(currentlyNominated ? 'Nicht nominiert' : 'Nominiert');
  }, [api, refreshEvents, reloadDetail, toastMsg]);

  return { reloadDetail, openEventDetail, setMyStatus, setStatusFor, canSeeComment, openComment, submitComment, postEventComment, removeEventComment, toggleNomination };
}

export type EventActionDeps = EventFeatureDeps & {
  askConfirm: (cfg: { title: string; message: string; confirmLabel?: string; danger?: boolean; onConfirm: () => void | Promise<void> }) => void;
  openEventDetail: (eventId: string) => Promise<void>;
};

export function useEventActionFeatures({ api, setState, askConfirm, refreshEvents, openEventDetail, toastMsg }: EventActionDeps) {
  const runEventAction = useCallback(async (action: 'cancel' | 'delete' | 'reactivate', event: TeamEvent, scope: 'single' | 'series') => {
    if (action === 'delete') {
      askConfirm({
        title: scope === 'series' ? 'Ganze Serie löschen?' : 'Termin löschen?',
        message: scope === 'series' ? 'Alle Termine dieser Serie und alle Rückmeldungen werden dauerhaft entfernt.' : '„' + event.title + '" und alle Rückmeldungen werden dauerhaft entfernt.',
        confirmLabel: 'Löschen', danger: true,
        onConfirm: async () => { await api.events.remove(event.id, scope); await refreshEvents(); setState({ sheet: null }); toastMsg(scope === 'series' ? 'Serie gelöscht' : 'Termin gelöscht'); },
      });
      return;
    }
    const status = action === 'cancel' ? 'cancelled' : 'active';
    await api.events.setStatus(event.id, status, scope);
    await refreshEvents();
    setState({ sheet: null });
    openEventDetail(event.id);
    toastMsg(action === 'cancel' ? (scope === 'series' ? 'Serie abgesagt' : 'Termin abgesagt') : (scope === 'series' ? 'Serie aktiviert' : 'Termin aktiviert'));
  }, [api, askConfirm, refreshEvents, setState, openEventDetail, toastMsg]);

  const askEventAction = useCallback((action: 'cancel' | 'delete' | 'reactivate', event: TeamEvent) => {
    if (event.seriesId) setState((st) => ({ sheet: { type: 'seriesAction', action, event, back: st.sheet } }));
    else runEventAction(action, event, 'single');
  }, [setState, runEventAction]);

  return { askEventAction, runEventAction };
}
