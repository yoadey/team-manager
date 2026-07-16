import { useCallback } from 'react';
import type { api as defaultApi } from '@/services';
import type { AppState } from '@/context/AppContext';
import { todayStr } from '@/styles/tokens';
import { reportActionError } from '@/utils/errors';
import { t } from '@/i18n';
import type { AbsenceFormValues } from '../components/absenceFormSchema';

type SetState = (patch: Partial<AppState> | ((s: AppState) => Partial<AppState>)) => void;

type AbsenceDeps = {
  api: typeof defaultApi;
  S: () => AppState;
  setState: SetState;
  refreshEvents: () => Promise<void>;
  loadAbsences: () => Promise<void>;
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

export function useAbsenceActions({
  api,
  S,
  setState,
  refreshEvents,
  loadAbsences,
  askConfirm,
  toastMsg,
  logout,
}: AbsenceDeps) {
  const openAbsenceForm = useCallback(
    (absence?: { id: string; from: string; to: string; reason: string } | null) => {
      const f: AbsenceFormValues = absence
        ? { id: absence.id, from: absence.from, to: absence.to, reason: absence.reason }
        : { from: todayStr(), to: todayStr(), reason: '' };
      setState({ sheet: { type: 'absenceForm', mode: absence ? 'edit' : 'create' }, form: f });
    },
    [setState],
  );

  const saveAbsence = useCallback(
    async (f: AbsenceFormValues) => {
      const mode = S().sheet!.mode;
      const sh = S().sheet;
      try {
        const teamId = S().activeTeamId!;
        if (mode === 'edit')
          await api.absences.update(f.id!, { from: f.from, to: f.to, reason: f.reason || '' }, teamId);
        else
          await api.absences.create({
            teamId,
            userId: S().user!.id,
            from: f.from,
            to: f.to,
            reason: f.reason || '',
          });
        await Promise.all([refreshEvents(), loadAbsences()]);
        // Don't close a sheet the user has since opened for a different team
        // after switching away mid-request, or one they've since opened while
        // this save was in flight.
        if (S().activeTeamId === teamId && S().sheet === sh) setState({ sheet: null });
        toastMsg(mode === 'edit' ? t('events.toastAbsenceUpdated') : t('events.toastAbsenceCreated'));
      } catch (err) {
        reportActionError({ setState, toastMsg, onAuthError: logout, S }, err, 'error.save');
        throw err;
      }
    },
    [api, S, setState, refreshEvents, loadAbsences, toastMsg, logout],
  );

  const removeAbsence = useCallback(
    (id: string) => {
      askConfirm({
        title: t('events.absenceDeleteTitle'),
        message: t('events.absenceDeleteMsg'),
        confirmLabel: t('common.delete'),
        danger: true,
        onConfirm: async () => {
          try {
            await api.absences.remove(id, S().activeTeamId!);
            await Promise.all([refreshEvents(), loadAbsences()]);
            toastMsg(t('events.toastAbsenceDeleted'));
          } catch (err) {
            reportActionError({ setState, toastMsg, onAuthError: logout }, err, 'error.delete');
          }
        },
      });
    },
    [api, S, askConfirm, refreshEvents, loadAbsences, setState, toastMsg, logout],
  );

  return { openAbsenceForm, saveAbsence, removeAbsence };
}
