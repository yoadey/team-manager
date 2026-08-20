import { useCallback } from 'react';
import type { api as defaultApi } from '@/services';
import type { TeamEvent } from '../types';
import type { EventFormValues } from '../components/eventFormSchema';
import type { AppState } from '@/context/AppContext';
import { hhmm, todayStr } from '@/styles/tokens';
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

/** Combines the hours+minutes cancellation-lead-time inputs into the single total-minutes value the API expects; 0/unset in both fields means no cutoff. */
function combineCancelLeadMinutes(f: EventFormValues): number | undefined {
  const total = (f.cancelLeadHours || 0) * 60 + (f.cancelLeadMinutes || 0);
  return total > 0 ? total : undefined;
}

/** True when both arrays contain the same ids, order and duplicates aside. */
function sameIdSet(a: string[], b: string[]): boolean {
  if (a.length !== b.length) return false;
  const sa = new Set(a);
  return b.every((id) => sa.has(id));
}

/** Builds the base event write payload shared by create and edit -- everything except the create-only recurrence fields (see buildRecurrencePayload) and crossTeamIds (see saveEvent's change-detection). */
function buildBasePayload(f: EventFormValues): {
  type: EventFormValues['type'];
  title: string;
  date: string;
  multiDayEndDate: string;
  location: string;
  note: string;
  meetTimeMandatory: boolean;
  responseMode: 'opt_in' | 'opt_out';
  meetT: string;
  startT: string;
  endT: string;
  nominatedRoleIds: string[];
  cancelLeadMinutes: number | undefined;
  excludeFromStats: boolean;
  crossTeamIds?: string[];
} {
  return {
    type: f.type,
    title: f.title.trim(),
    date: f.date,
    multiDayEndDate: f.recurring ? '' : f.multiDayEndDate || '',
    location: f.location || '',
    note: f.note || '',
    meetTimeMandatory: !!f.meetTimeMandatory,
    responseMode: f.responseMode || 'opt_in',
    meetT: f.meetT || '',
    startT: f.startT || '',
    endT: f.endT || '',
    nominatedRoleIds: f.nominatedRoleIds || [],
    cancelLeadMinutes: combineCancelLeadMinutes(f),
    excludeFromStats: !!f.excludeFromStats,
  };
}

/**
 * Builds the create-only recurrence fields, branching on repeatMode: the two
 * are mutually exclusive server-side (endDate takes precedence when both are
 * set), so only the field the toggle is currently on gets forwarded.
 */
function buildRecurrencePayload(f: EventFormValues) {
  const recurring = f.recurring ?? false;
  const usingEndDate = recurring && f.repeatMode === 'until';
  return {
    recurring,
    ...(usingEndDate ? { endDate: f.repeatEndDate } : { repeatWeeks: f.repeatWeeks || 8 }),
  };
}

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
    (event: TeamEvent | null, initialDate?: string) => {
      const f: EventFormValues = event
        ? {
            seriesId: event.seriesId || null,
            type: event.type,
            title: event.title,
            date: event.date,
            multiDayEndDate: event.multiDayEndDate || '',
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
            repeatMode: 'weeks',
            repeatEndDate: '',
            cancelLeadHours: event.cancelLeadMinutes != null ? Math.floor(event.cancelLeadMinutes / 60) : 0,
            cancelLeadMinutes: event.cancelLeadMinutes != null ? event.cancelLeadMinutes % 60 : 0,
            excludeFromStats: !!event.excludeFromStats,
            crossTeamIds: event.crossTeamIds || [],
          }
        : {
            type: 'training',
            title: '',
            date: initialDate || todayStr(),
            multiDayEndDate: '',
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
            repeatMode: 'weeks',
            repeatEndDate: '',
            cancelLeadHours: 0,
            cancelLeadMinutes: 0,
            excludeFromStats: false,
            crossTeamIds: [],
          };
      setState((st) => ({
        sheet: {
          type: 'eventForm',
          mode: event ? 'edit' : 'create',
          eventId: event?.id,
          back: st.sheet && st.sheet.type === 'eventDetail' ? st.sheet : null,
          formInitial: f,
        },
      }));
    },
    [setState, S],
  );

  // Opens the create form pre-filled from an existing event, but as a new,
  // standalone, non-recurring event: seriesId/date reset so saving never
  // touches the source event or its series (see design.md's "Copy is
  // client-side only" decision). A multi-day source event's span length is
  // preserved, anchored to the new default date instead of the source's own
  // dates.
  const duplicateEvent = useCallback(
    (event: TeamEvent, initialDate?: string) => {
      const newDate = initialDate || todayStr();
      let newEndDate = '';
      if (event.multiDayEndDate) {
        const spanDays = Math.round(
          (new Date(event.multiDayEndDate + 'T00:00:00Z').getTime() - new Date(event.date + 'T00:00:00Z').getTime()) /
            86_400_000,
        );
        const shifted = new Date(newDate + 'T00:00:00Z');
        shifted.setUTCDate(shifted.getUTCDate() + spanDays);
        newEndDate = shifted.toISOString().slice(0, 10);
      }
      const f: EventFormValues = {
        seriesId: null,
        type: event.type,
        title: event.title,
        date: newDate,
        multiDayEndDate: newEndDate,
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
        repeatMode: 'weeks',
        repeatEndDate: '',
        cancelLeadHours: event.cancelLeadMinutes != null ? Math.floor(event.cancelLeadMinutes / 60) : 0,
        cancelLeadMinutes: event.cancelLeadMinutes != null ? event.cancelLeadMinutes % 60 : 0,
        excludeFromStats: !!event.excludeFromStats,
        crossTeamIds: event.crossTeamIds || [],
      };
      setState((st) => ({
        sheet: {
          type: 'eventForm',
          mode: 'create',
          eventId: undefined,
          back: st.sheet && st.sheet.type === 'eventDetail' ? st.sheet : null,
          formInitial: f,
        },
      }));
    },
    [setState, S],
  );

  const saveEvent = useCallback(
    async (f: EventFormValues, scope: 'single' | 'series' = 'single') => {
      const sh = S().sheet!;
      const mode = sh.mode;
      const back = sh.back;
      const payload = buildBasePayload(f);
      // On create, always forward the picker's current selection (empty means
      // single-team, same as absent). On edit, only forward it when the user
      // actually changed the selection from what the form opened with --
      // sending it unconditionally would make every edit (even an unrelated
      // field) re-validate events:write across the *current* full target set
      // (see UpdateEventRequest's "absent leaves the target set unchanged" --
      // that's exactly the escape hatch this preserves for edits that don't
      // touch sharing).
      const initialCrossTeamIds = (sh.formInitial as EventFormValues | undefined)?.crossTeamIds ?? [];
      const crossTeamIds = f.crossTeamIds ?? [];
      const crossTeamIdsChanged = mode === 'create' || !sameIdSet(initialCrossTeamIds, crossTeamIds);
      if (crossTeamIdsChanged) payload.crossTeamIds = crossTeamIds;
      try {
        if (mode === 'edit') await saveEventAsync({ mode: 'edit', eventId: sh.eventId!, scope, payload });
        else await saveEventAsync({ mode: 'create', payload: { ...payload, ...buildRecurrencePayload(f) } });
        loadNotifications();
        // Don't close/reopen a sheet the user has since opened for a
        // different team after switching away mid-request -- openEventDetail
        // would look up sh.eventId in the new team's event list and find
        // nothing. Also don't touch it if the user has since closed this form
        // and opened a DIFFERENT one (same team) while this save was in
        // flight -- otherwise a slow save for event A would silently close
        // and replace whatever the user is now looking at (e.g. an edit form
        // for event B) with A's detail view, discarding B's unsaved edits
        // without warning.
        if (S().activeTeamId === teamId && S().sheet === sh) {
          setState({ sheet: null });
          if (mode === 'edit' && back && back.type === 'eventDetail') openEventDetail(sh.eventId!);
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
        throw err;
      }
    },
    [S, setState, teamId, saveEventAsync, openEventDetail, loadNotifications, toastMsg, logout],
  );

  return { openEventForm, duplicateEvent, saveEvent, savingEvent };
}
