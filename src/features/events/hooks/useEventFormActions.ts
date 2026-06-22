import { useCallback } from 'react';
import type { api as defaultApi } from '@/services/serviceLayer';
import type { EventFormValues, TeamEvent } from '../types';
import type { AppState } from '@/context/AppContext';
import { formValues } from '@/utils/forms';
import { hhmm, todayStr } from '@/styles/tokens';
import { validateEventForm } from '@/utils/validation';
import { reportActionError } from '@/utils/errors';

type SetState = (patch: Partial<AppState> | ((s: AppState) => Partial<AppState>)) => void;

type EventFormDeps = {
  api: typeof defaultApi;
  S: () => AppState;
  setState: SetState;
  refreshEvents: () => Promise<void>;
  openEventDetail: (eventId: string) => Promise<void>;
  toastMsg: (m: string) => void;
};

export function useEventFormActions({ api, S, setState, refreshEvents, openEventDetail, toastMsg }: EventFormDeps) {
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
            location: 'Tanzsporthalle Eilendorf',
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
        toastMsg(validation.message!);
        return;
      }
      const back = sh.back;
      setState({ busy: 'save' });
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
        if (mode === 'edit') await api.events.update(f.id!, payload, scope);
        else
          await api.events.create(S().activeTeamId!, {
            ...payload,
            recurring: f.recurring,
            repeatWeeks: validation.value!.repeatWeeks,
            nominatedRoleIds: f.nominatedRoleIds,
          });
        await refreshEvents();
        setState({ busy: null, sheet: null });
        if (mode === 'edit' && back && back.type === 'eventDetail') openEventDetail(f.id!);
        toastMsg(
          mode === 'edit'
            ? scope === 'series'
              ? 'Ganze Serie aktualisiert'
              : 'Termin aktualisiert'
            : 'Termin angelegt',
        );
      } catch (err) {
        reportActionError({ setState, toastMsg }, err, 'error.save');
      }
    },
    [api, S, setState, refreshEvents, openEventDetail, toastMsg],
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

  return { openEventForm, saveEvent, toggleFormNomRole };
}
