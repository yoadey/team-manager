import { useCallback } from 'react';
import type { api as defaultApi } from '@/services/serviceLayer';
import type { AppState } from '@/context/AppContext';
import { validateDateRange } from '@/utils/validation';
import { todayStr } from '@/styles/tokens';
import { reportActionError } from '@/utils/errors';

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
  toastMsg: (m: string) => void;
};

export function useAbsenceActions({
  api,
  S,
  setState,
  refreshEvents,
  loadAbsences,
  askConfirm,
  toastMsg,
}: AbsenceDeps) {
  const openAbsenceForm = useCallback(
    (absence?: { id: string; from: string; to: string; reason: string } | null) => {
      const f = absence
        ? { id: absence.id, from: absence.from, to: absence.to, reason: absence.reason }
        : { from: todayStr(), to: todayStr(), reason: 'Urlaub' };
      setState({ sheet: { type: 'absenceForm', mode: absence ? 'edit' : 'create' }, form: f });
    },
    [setState],
  );

  const saveAbsence = useCallback(async () => {
    const f = S().form;
    const range = validateDateRange(f.from, f.to);
    if (!range.ok) {
      toastMsg(range.message!);
      return;
    }
    const mode = S().sheet!.mode;
    setState({ busy: 'save' });
    try {
      if (mode === 'edit')
        await api.absences.update(f.id, { from: range.value!.from, to: range.value!.to, reason: f.reason });
      else await api.absences.create({ from: range.value!.from, to: range.value!.to, reason: f.reason });
      await Promise.all([refreshEvents(), loadAbsences()]);
      setState({ busy: null, sheet: null });
      toastMsg(mode === 'edit' ? 'Abwesenheit aktualisiert' : 'Abwesenheit eingetragen');
    } catch (err) {
      reportActionError({ setState, toastMsg }, err, 'error.save');
    }
  }, [api, S, setState, refreshEvents, loadAbsences, toastMsg]);

  const removeAbsence = useCallback(
    (id: string) => {
      askConfirm({
        title: 'Abwesenheit löschen?',
        message: 'Der Zeitraum wird entfernt. Automatisch gesetzte Absagen in diesem Zeitraum werden zurückgenommen.',
        confirmLabel: 'Löschen',
        danger: true,
        onConfirm: async () => {
          try {
            await api.absences.remove(id);
            await Promise.all([refreshEvents(), loadAbsences()]);
            toastMsg('Abwesenheit entfernt');
          } catch (err) {
            reportActionError({ setState, toastMsg }, err, 'error.delete');
          }
        },
      });
    },
    [api, askConfirm, refreshEvents, loadAbsences, setState, toastMsg],
  );

  return { openAbsenceForm, saveAbsence, removeAbsence };
}
