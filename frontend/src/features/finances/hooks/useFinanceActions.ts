import { useCallback, useRef } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import type { api as defaultApi } from '@/services';
import type { DateRange } from '@/types';
import type { AppState, ConfirmConfig } from '@/context/AppContext';
import type {
  Contribution,
  ContribFormValues,
  FinanceOverview,
  Penalty,
  PenaltyAssignFormValues,
  PenaltyFormValues,
  Transaction,
  TxFormValues,
} from '../types';
import { MAX_MONEY_AMOUNT_EUROS, validateMoneyAmount, validateRequiredText } from '@/utils/validation';
import { reportActionError } from '@/utils/errors';
import { t } from '@/i18n';
import { queryKeys } from '@/query/keys';
import {
  useDeleteAssignmentMutation,
  useDeletePenaltyMutation,
  useDeleteTxMutation,
  useSaveContribMutation,
  useSavePenaltyAssignMutation,
  useSavePenaltyMutation,
  useSaveTxMutation,
  useTogglePenaltyMutation,
  useToggleContributionMutation,
} from './useFinanceMutations';

type SetState = (patch: Partial<AppState> | ((s: AppState) => Partial<AppState>)) => void;

type FinanceFeatureDeps = {
  api: typeof defaultApi;
  S: () => AppState;
  setState: SetState;
  /** Reactive (render-time) active team id -- the query/mutation hooks key off this directly
   * rather than through `S()`, since a `useQuery`/`useMutation` call must re-run on every
   * render to pick up a team switch instead of only when some later callback fires. */
  teamId: string | null;
  loadStats: (range?: DateRange | null) => Promise<void>;
  askConfirm: (cfg: ConfirmConfig) => void;
  toastMsg: (m: string, action?: { label: string; fn: () => void }, kind?: 'success' | 'error') => void;
  logout: () => void;
};

export function useFinanceActions({
  api,
  S,
  setState,
  teamId,
  loadStats,
  askConfirm,
  toastMsg,
  logout,
}: FinanceFeatureDeps) {
  const queryClient = useQueryClient();
  const { mutateAsync: saveTxAsync, isPending: savingTx } = useSaveTxMutation(api, teamId);
  const { mutateAsync: deleteTxAsync } = useDeleteTxMutation(api);
  const { mutateAsync: savePenaltyAsync, isPending: savingPenalty } = useSavePenaltyMutation(api, teamId);
  const { mutateAsync: deletePenaltyAsync } = useDeletePenaltyMutation(api);
  const { mutateAsync: savePenaltyAssignAsync, isPending: savingPenaltyAssign } = useSavePenaltyAssignMutation(
    api,
    teamId,
  );
  const { mutateAsync: deleteAssignmentAsync } = useDeleteAssignmentMutation(api);
  const { mutateAsync: saveContribAsync, isPending: savingContrib } = useSaveContribMutation(api, teamId);
  const { mutateAsync: togglePenaltyAsync } = useTogglePenaltyMutation(api, teamId);
  const { mutateAsync: toggleContributionAsync } = useToggleContributionMutation(api, teamId);

  const openTxForm = useCallback(
    (tx?: Transaction) => {
      const f: TxFormValues = tx
        ? { id: tx.id, type: tx.type, title: tx.title, amount: String(tx.amount), category: tx.category }
        : { type: 'income', title: '', amount: '', category: '' };
      setState({ sheet: { type: 'txForm', mode: tx ? 'edit' : 'create' }, form: f, formErrors: {} });
    },
    [setState],
  );

  const saveTx = useCallback(async () => {
    const f = S().form as TxFormValues;
    const title = validateRequiredText(f.title, t('finances.txFieldTitleError'));
    if (!title.ok) {
      toastMsg(title.message!, undefined, 'error');
      return;
    }
    const amount = validateMoneyAmount(f.amount, { positive: true, max: MAX_MONEY_AMOUNT_EUROS });
    if (!amount.ok) {
      toastMsg(amount.message!, undefined, 'error');
      return;
    }
    const sh = S().sheet;
    const savedTeamId = teamId;
    try {
      const mode = S().sheet!.mode === 'edit' ? 'edit' : 'create';
      await saveTxAsync({
        mode,
        id: f.id,
        payload: { type: f.type, title: title.value!, amount: amount.value!, category: f.category },
      });
      // Don't close a sheet the user has since opened for a different team
      // after switching away mid-request, or one they've since opened for a
      // different entity (same team) while this save was in flight.
      if (S().activeTeamId === savedTeamId && S().sheet === sh) setState({ sheet: null });
      toastMsg(t('finances.toastTxSaved'));
    } catch (err) {
      reportActionError({ setState, toastMsg, onAuthError: logout }, err, 'error.save');
    }
  }, [S, setState, saveTxAsync, teamId, toastMsg, logout]);

  const deleteTx = useCallback(
    async (id: string) => {
      const sh = S().sheet;
      const deletedTeamId = teamId!;
      try {
        await deleteTxAsync({ id, teamId: deletedTeamId });
        if (S().activeTeamId === deletedTeamId && S().sheet === sh) setState({ sheet: null });
        toastMsg(t('finances.toastTxDeleted'));
      } catch (err) {
        reportActionError({ setState, toastMsg, onAuthError: logout }, err, 'error.delete');
      }
    },
    [S, deleteTxAsync, setState, teamId, toastMsg, logout],
  );

  const openPenaltyCatalog = useCallback(() => setState({ sheet: { type: 'penaltyCatalog' } }), [setState]);

  const openPenaltyForm = useCallback(
    (p?: Penalty) =>
      setState((st) => ({
        sheet: {
          type: 'penaltyForm',
          mode: p ? 'edit' : 'create',
          back: st.sheet && st.sheet.type === 'penaltyCatalog' ? st.sheet : null,
        },
        form: (p
          ? { id: p.id, label: p.label, amount: String(p.amount) }
          : { label: '', amount: '' }) satisfies PenaltyFormValues,
        formErrors: {},
      })),
    [setState],
  );

  const savePenalty = useCallback(async () => {
    const f = S().form as PenaltyFormValues;
    const label = validateRequiredText(f.label, t('finances.penaltyFieldLabelError'));
    if (!label.ok) {
      toastMsg(label.message!, undefined, 'error');
      return;
    }
    const amount = validateMoneyAmount(f.amount, { positive: true, max: MAX_MONEY_AMOUNT_EUROS });
    if (!amount.ok) {
      toastMsg(amount.message!, undefined, 'error');
      return;
    }
    const sh = S().sheet!;
    const back = sh.back || null;
    const create = sh.mode === 'create';
    const savedTeamId = teamId;
    try {
      await savePenaltyAsync({
        mode: create ? 'create' : 'edit',
        id: f.id,
        payload: { label: label.value!, amount: amount.value! },
      });
      // Don't navigate away from a sheet the user has since opened for a
      // different team after switching away mid-request, or one they've
      // since opened for a different entity (same team) while this save was
      // in flight.
      if (S().activeTeamId === savedTeamId && S().sheet === sh) setState({ sheet: back });
      toastMsg(create ? t('finances.toastPenaltyAdded') : t('finances.toastPenaltySaved'));
    } catch (err) {
      reportActionError({ setState, toastMsg, onAuthError: logout }, err, 'error.save');
    }
  }, [S, setState, savePenaltyAsync, teamId, toastMsg, logout]);

  const deletePenaltyDef = useCallback(
    (id: string) => {
      const deletedTeamId = teamId!;
      askConfirm({
        title: t('finances.penaltyDeleteTitle'),
        message: t('finances.penaltyDeleteMsg'),
        confirmLabel: t('finances.penaltyDeleteConfirm'),
        danger: true,
        onConfirm: async () => {
          const sh = S().sheet;
          try {
            await deletePenaltyAsync({ id, teamId: deletedTeamId });
            if (S().activeTeamId === deletedTeamId && S().sheet === sh) setState({ sheet: { type: 'penaltyCatalog' } });
            toastMsg(t('finances.toastPenaltyRemoved'));
          } catch (err) {
            reportActionError({ setState, toastMsg, onAuthError: logout }, err, 'error.delete');
          }
        },
      });
    },
    [S, askConfirm, deletePenaltyAsync, setState, teamId, toastMsg, logout],
  );

  const openPenaltyAssign = useCallback(() => {
    // The member picker and penalty catalog are populated by
    // PenaltyAssignSheet's own useMembersQuery/useFinanceOverviewQuery, which
    // fetch/retry on their own -- no manual refresh needed here.
    const f = queryClient.getQueryData<FinanceOverview>(queryKeys.finances(teamId ?? ''));
    const first = f && f.penalties[0] ? f.penalties[0].id : null;
    const form: PenaltyAssignFormValues = { userId: '', penaltyId: first };
    setState({ sheet: { type: 'penaltyAssign' }, form, formErrors: {} });
  }, [queryClient, teamId, setState]);

  const savePenaltyAssign = useCallback(async () => {
    const f = S().form as PenaltyAssignFormValues;
    if (!f.userId) {
      toastMsg(t('finances.assignPersonError'), undefined, 'error');
      return;
    }
    if (!f.penaltyId) {
      toastMsg(t('finances.assignPenaltyError'), undefined, 'error');
      return;
    }
    const sh = S().sheet;
    const savedTeamId = teamId;
    try {
      await savePenaltyAssignAsync({ userId: f.userId, penaltyId: f.penaltyId });
      if (S().activeTeamId === savedTeamId && S().sheet === sh) setState({ sheet: null });
      toastMsg(t('finances.toastPenaltyAssigned'));
    } catch (err) {
      reportActionError({ setState, toastMsg, onAuthError: logout }, err, 'error.save');
    }
  }, [S, setState, savePenaltyAssignAsync, teamId, toastMsg, logout]);

  const deleteAssignment = useCallback(
    (id: string) => {
      const deletedTeamId = teamId!;
      askConfirm({
        title: t('finances.assignmentDeleteTitle'),
        message: t('finances.assignmentDeleteMsg'),
        confirmLabel: t('finances.assignmentDeleteConfirm'),
        danger: true,
        onConfirm: async () => {
          try {
            await deleteAssignmentAsync({ id, teamId: deletedTeamId });
            toastMsg(t('finances.toastPenaltyAssignDeleted'));
          } catch (err) {
            reportActionError({ setState, toastMsg, onAuthError: logout }, err, 'error.delete');
          }
        },
      });
    },
    [askConfirm, deleteAssignmentAsync, setState, teamId, toastMsg, logout],
  );

  const openContribForm = useCallback(
    (c: Contribution) => {
      const form: ContribFormValues = { id: c.id, label: c.label, amount: String(c.amount) };
      setState({ sheet: { type: 'contribForm' }, form, formErrors: {} });
    },
    [setState],
  );

  const saveContrib = useCallback(async () => {
    const f = S().form as ContribFormValues;
    const label = validateRequiredText(f.label, t('finances.contribFieldLabelError'));
    if (!label.ok) {
      toastMsg(label.message!, undefined, 'error');
      return;
    }
    const amount = validateMoneyAmount(f.amount, { positive: true, max: MAX_MONEY_AMOUNT_EUROS });
    if (!amount.ok) {
      toastMsg(amount.message!, undefined, 'error');
      return;
    }
    const sh = S().sheet;
    const savedTeamId = teamId;
    try {
      await saveContribAsync({ id: f.id, payload: { label: label.value!, amount: amount.value! } });
      if (S().activeTeamId === savedTeamId && S().sheet === sh) setState({ sheet: null });
      toastMsg(t('finances.toastContribSaved'));
    } catch (err) {
      reportActionError({ setState, toastMsg, onAuthError: logout }, err, 'error.save');
    }
  }, [S, setState, saveContribAsync, teamId, toastMsg, logout]);

  const toggleInFlight = useRef(new Set<string>());

  const togglePenalty = useCallback(
    async (id: string) => {
      const key = 'penalty:' + id;
      if (toggleInFlight.current.has(key)) return;
      toggleInFlight.current.add(key);
      try {
        await togglePenaltyAsync(id);
      } catch (err) {
        reportActionError({ setState, toastMsg, onAuthError: logout }, err);
      } finally {
        toggleInFlight.current.delete(key);
      }
    },
    [togglePenaltyAsync, setState, toastMsg, logout],
  );

  const toggleContribution = useCallback(
    async (id: string) => {
      const key = 'contribution:' + id;
      if (toggleInFlight.current.has(key)) return;
      toggleInFlight.current.add(key);
      try {
        await toggleContributionAsync(id);
      } catch (err) {
        reportActionError({ setState, toastMsg, onAuthError: logout }, err);
      } finally {
        toggleInFlight.current.delete(key);
      }
    },
    [toggleContributionAsync, setState, toastMsg, logout],
  );

  const setStatsRange = useCallback(
    (range: DateRange | null) => {
      setState({ statsRange: range, stats: null });
      loadStats(range);
    },
    [setState, loadStats],
  );

  return {
    openTxForm,
    saveTx,
    deleteTx,
    openPenaltyCatalog,
    openPenaltyForm,
    savePenalty,
    deletePenaltyDef,
    openPenaltyAssign,
    savePenaltyAssign,
    deleteAssignment,
    openContribForm,
    saveContrib,
    togglePenalty,
    toggleContribution,
    setStatsRange,
    savingTx,
    savingPenalty,
    savingPenaltyAssign,
    savingContrib,
  };
}
