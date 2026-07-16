import { useCallback } from 'react';
import type { api as defaultApi } from '@/services';
import type { AppState } from '@/context/AppContext';
import type { AbsenceFormValues } from '../types';
import { validateDateRange } from '@/utils/validation';
import { todayStr } from '@/styles/tokens';
import { reportActionError } from '@/utils/errors';
import { t } from '@/i18n';
import { useDeleteAbsenceMutation, useSaveAbsenceMutation } from './useAbsenceMutations';

type SetState = (patch: Partial<AppState> | ((s: AppState) => Partial<AppState>)) => void;

type AbsenceDeps = {
  api: typeof defaultApi;
  S: () => AppState;
  setState: SetState;
  /** Reactive (render-time) active team id -- the query/mutation hooks key off this directly
   * rather than through `S()`, since a `useQuery`/`useMutation` call must re-run on every
   * render to pick up a team switch instead of only when some later callback fires. */
  teamId: string | null;
  /** An absence can flip a "pending RSVP" notification (its auto-attendance
   * side effect on an overlapping event), so each mutation also refreshes
   * the notification badge, mirroring the pre-migration loadAbsences(). */
  loadNotifications: () => Promise<void>;
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
  teamId,
  loadNotifications,
  askConfirm,
  toastMsg,
  logout,
}: AbsenceDeps) {
  const { mutateAsync: saveAbsenceAsync, isPending: savingAbsence } = useSaveAbsenceMutation(api, teamId);
  const { mutateAsync: deleteAbsenceAsync } = useDeleteAbsenceMutation(api);

  const openAbsenceForm = useCallback(
    (absence?: { id: string; from: string; to: string; reason: string } | null) => {
      const f: AbsenceFormValues = absence
        ? { id: absence.id, from: absence.from, to: absence.to, reason: absence.reason }
        : { from: todayStr(), to: todayStr(), reason: '' };
      setState({ sheet: { type: 'absenceForm', mode: absence ? 'edit' : 'create' }, form: f });
    },
    [setState],
  );

  const saveAbsence = useCallback(async () => {
    const f = S().form as AbsenceFormValues;
    const range = validateDateRange(f.from, f.to);
    if (!range.ok) {
      toastMsg(range.message!, undefined, 'error');
      return;
    }
    const mode = S().sheet!.mode;
    const sh = S().sheet;
    const savedTeamId = teamId;
    try {
      await saveAbsenceAsync({
        id: mode === 'edit' ? f.id : undefined,
        payload: { from: range.value!.from, to: range.value!.to, reason: f.reason },
        userId: S().user!.id,
      });
      loadNotifications();
      // Don't close a sheet the user has since opened for a different team
      // after switching away mid-request, or one they've since opened while
      // this save was in flight.
      if (S().activeTeamId === savedTeamId && S().sheet === sh) setState({ sheet: null });
      toastMsg(mode === 'edit' ? t('events.toastAbsenceUpdated') : t('events.toastAbsenceCreated'));
    } catch (err) {
      reportActionError({ setState, toastMsg, onAuthError: logout }, err, 'error.save');
    }
  }, [S, setState, saveAbsenceAsync, loadNotifications, teamId, toastMsg, logout]);

  const removeAbsence = useCallback(
    (id: string) => {
      const deletedTeamId = teamId!;
      askConfirm({
        title: t('events.absenceDeleteTitle'),
        message: t('events.absenceDeleteMsg'),
        confirmLabel: t('common.delete'),
        danger: true,
        onConfirm: async () => {
          try {
            await deleteAbsenceAsync({ id, teamId: deletedTeamId });
            loadNotifications();
            toastMsg(t('events.toastAbsenceDeleted'));
          } catch (err) {
            reportActionError({ setState, toastMsg, onAuthError: logout }, err, 'error.delete');
          }
        },
      });
    },
    [askConfirm, deleteAbsenceAsync, loadNotifications, setState, teamId, toastMsg, logout],
  );

  return { openAbsenceForm, saveAbsence, removeAbsence, savingAbsence };
}
