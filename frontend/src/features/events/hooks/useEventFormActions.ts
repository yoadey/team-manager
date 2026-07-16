import { useCallback } from 'react';
import type { api as defaultApi } from '@/services';
import type { TeamEvent } from '../types';
import type { EventFormValues } from '../components/eventFormSchema';
import type { AppState } from '@/context/AppContext';
import { hhmm, todayStr } from '@/styles/tokens';
import { reportActionError } from '@/utils/errors';
import { t } from '@/i18n';

type SetState = (patch: Partial<AppState> | ((s: AppState) => Partial<AppState>)) => void;

type EventFormDeps = {
  api: typeof defaultApi;
  S: () => AppState;
  setState: SetState;
  refreshEvents: () => Promise<void>;
  openEventDetail: (eventId: string) => Promise<void>;
  toastMsg: (m: string, action?: { label: string; fn: () => void }, kind?: 'success' | 'error') => void;
  logout: () => void;
};

export function useEventFormActions({
  api,
  S,
  setState,
  refreshEvents,
  openEventDetail,
  toastMsg,
  logout,
}: EventFormDeps) {
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
    async (f: EventFormValues, scope: 'single' | 'series' = 'single') => {
      const sh = S().sheet!;
      const mode = sh.mode;
      const back = sh.back;
      const teamId = S().activeTeamId!;
      const payload = {
        type: f.type,
        title: f.title.trim(),
        date: f.date,
        location: f.location || '',
        note: f.note || '',
        meetTimeMandatory: !!f.meetTimeMandatory,
        responseMode: f.responseMode || 'opt_in',
        meetT: f.meetT || '',
        startT: f.startT || '',
        endT: f.endT || '',
        nominatedRoleIds: f.nominatedRoleIds || [],
      };
      try {
        if (mode === 'edit') await api.events.update(f.id!, payload, scope, teamId);
        else
          await api.events.create(teamId, {
            ...payload,
            recurring: f.recurring,
            repeatWeeks: f.repeatWeeks || 8,
          });
        await refreshEvents();
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
        reportActionError({ setState, toastMsg, onAuthError: logout, S }, err, 'error.save');
        throw err; // propagates error to let react-hook-form handle submission state
      }
    },
    [api, S, setState, refreshEvents, openEventDetail, toastMsg, logout],
  );

  return { openEventForm, saveEvent };
}
