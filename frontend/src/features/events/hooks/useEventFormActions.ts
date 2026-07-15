import { useCallback } from 'react';
import type { api as defaultApi } from '@/services';
import type { EventFormValues, TeamEvent } from '../types';
import type { AppState } from '@/context/AppContext';
import { formValues } from '@/utils/forms';
import { hhmm, todayStr } from '@/styles/tokens';
import { validateEventForm } from '@/utils/validation';
import { reportActionError } from '@/utils/errors';
import { t } from '@/i18n';
import { useSaveEventMutation } from './useEventMutations';

type SetState = (patch: Partial<AppState> | ((s: AppState) => Partial<AppState>)) => void;

type EventFormDeps = {
  api: typeof defaultApi;
  S: () => AppState;
  setState: SetState;
  teamId: string | null;
  loadNotifications: () => Promise<void>;
  openEventDetail: (eventId: string) => void;
  toastMsg: (m: string, action?: { label: string; fn: () => void }, kind?: 'success' | 'error') => void;
  logout: () => void;
};

export function useEventFormActions({
  api,
  S,
  setState,
  teamId,
  loadNotifications,
  openEventDetail,
  toastMsg,
  logout,
}: EventFormDeps) {
  const { mutateAsync: saveEventAsync, isPending: savingEvent } = useSaveEventMutation(api, teamId);

  const openEventForm = useCallback(
    (event: TeamEvent | null) => {
      const f: EventFormValues = event
        ? {
            id: event.id,
            seriesId: event.seriesId || null,
            type: event.type,
            title: event.title,
            date: event.date,
            meetT: hhmm(event.meetTime),
            startT: hhmm(event.startTime),
            endT: hhmm(event.endTime),
            location: event.location || '',
            note: event.note || '',
            meetTimeMandatory: !!event.meetTimeMandatory,
            responseMode: event.responseMode || 'opt_in',
            nominatedRoleIds: event.nominatedRoleIds || S().roles.map((r) => r.id),
            recurring: false,
            repeatWeeks: 8,
          }
        : {
            type: 'training',
            title: '',
            date: todayStr(),
            meetT: '19:15',
            startT: '19:30',
            endT: '21:30',
            location: '',
            note: '',
            meetTimeMandatory: true,
            responseMode: 'opt_out',
            nominatedRoleIds: S().roles.map((r) => r.id),
            recurring: false,
            repeatWeeks: 8,
          };
      setState((st) => ({
        sheet: {
          type: 'eventForm',
          mode: event ? 'edit' : 'create',
          back: st.sheet && st.sheet.type === 'eventDetail' ? st.sheet : null,
        },
        form: f,
        formErrors: {},
      }));
    },
    [setState, S],
  );

  const saveEvent = useCallback(
    async (scope: 'single' | 'series' = 'single') => {
      const f = S().form as EventFormValues;
      const sh = S().sheet!;
      const mode = sh.mode;
      const validation = validateEventForm(f, mode);
      if (!validation.ok) {
        toastMsg(validation.message!, undefined, 'error');
        return;
      }
      const back = sh.back;
      const payload = {
        type: f.type,
        title: f.title.trim(),
        date: f.date,
        location: f.location,
        note: f.note,
        meetTimeMandatory: f.meetTimeMandatory,
        responseMode: f.responseMode,
        meetT: f.meetT,
        startT: f.startT,
        endT: f.endT,
        nominatedRoleIds: f.nominatedRoleIds,
      };
      try {
        if (mode === 'edit') await saveEventAsync({ mode: 'edit', eventId: f.id!, scope, payload });
        else
          await saveEventAsync({
            mode: 'create',
            payload: { ...payload, recurring: f.recurring, repeatWeeks: validation.value!.repeatWeeks },
          });
        loadNotifications();
        // Don't close/reopen a sheet the user has since opened for a
        // different team after switching away mid-request -- openEventDetail
        // would look up f.id in the new team's event list and find nothing.
        // Also don't touch it if the user has since closed this form and
        // opened a DIFFERENT one (same team) while this save was in flight --
        // otherwise a slow save for event A would silently close and replace
        // whatever the user is now looking at (e.g. an edit form for event B)
        // with A's detail view, discarding B's unsaved edits without warning.
        if (S().activeTeamId === teamId && S().sheet === sh) {
          setState({ sheet: null });
          if (mode === 'edit' && back && back.type === 'eventDetail') openEventDetail(f.id!);
        }
        toastMsg(
          mode === 'edit'
            ? scope === 'series'
              ? t('events.toastSeriesUpdated')
              : t('events.toastEventUpdated')
            : t('events.toastEventCreated'),
        );
      } catch (err) {
        reportActionError({ setState, toastMsg, onAuthError: logout }, err, 'error.save');
      }
    },
    [S, setState, teamId, saveEventAsync, openEventDetail, loadNotifications, toastMsg, logout],
  );

  const toggleFormNomRole = useCallback(
    (roleId: string) =>
      setState((s) => {
        const cur = formValues<EventFormValues>(s).nominatedRoleIds ?? [];
        const next = cur.includes(roleId) ? cur.filter((x) => x !== roleId) : cur.concat(roleId);
        return { form: { ...s.form, nominatedRoleIds: next } };
      }),
    [setState],
  );

  return { openEventForm, saveEvent, toggleFormNomRole, savingEvent };
}
