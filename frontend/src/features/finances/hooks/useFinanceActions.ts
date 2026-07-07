import { useCallback, useRef } from 'react';
import type { api as defaultApi } from '@/services/serviceLayer';
import type { DateRange } from '@/types';
import type { AppState, ConfirmConfig } from '@/context/AppContext';
import type {
  Contribution,
  ContribFormValues,
  Penalty,
  PenaltyAssignFormValues,
  PenaltyFormValues,
  Transaction,
  TxFormValues,
} from '../types';
import { validateMoneyAmount, validateRequiredText } from '@/utils/validation';
import { reportActionError } from '@/utils/errors';
import { t } from '@/i18n';

type SetState = (patch: Partial<AppState> | ((s: AppState) => Partial<AppState>)) => void;

type FinanceFeatureDeps = {
  api: typeof defaultApi;
  S: () => AppState;
  setState: SetState;
  loadFinances: () => Promise<void>;
  loadStats: (range?: DateRange | null) => Promise<void>;
  refreshMembers: () => Promise<void>;
  askConfirm: (cfg: ConfirmConfig) => void;
  toastMsg: (m: string) => void;
  logout: () => void;
};

export function useFinanceActions({
  api,
  S,
  setState,
  loadFinances,
  loadStats,
  refreshMembers,
  askConfirm,
  toastMsg,
  logout,
}: FinanceFeatureDeps) {
  const openTxForm = useCallback(
    (tx?: Transaction) => {
      const f: TxFormValues = tx
        ? { id: tx.id, type: tx.type, title: tx.title, amount: String(tx.amount), category: tx.category }
        : { type: 'income', title: '', amount: '', category: 'Beiträge' };
      setState({ sheet: { type: 'txForm', mode: tx ? 'edit' : 'create' }, form: f, formErrors: {} });
    },
    [setState],
  );

  const saveTx = useCallback(async () => {
    const f = S().form as TxFormValues;
    const title = validateRequiredText(f.title, t('finances.txFieldTitleError'));
    if (!title.ok) {
      toastMsg(title.message!);
      return;
    }
    const amount = validateMoneyAmount(f.amount, { positive: true });
    if (!amount.ok) {
      toastMsg(amount.message!);
      return;
    }
    const teamId = S().activeTeamId!;
    setState({ busy: 'save' });
    try {
      if (S().sheet!.mode === 'edit')
        await api.finances.updateTransaction(
          f.id!,
          {
            type: f.type,
            title: title.value!,
            amount: amount.value!,
            category: f.category,
          },
          teamId,
        );
      else
        await api.finances.addTransaction(teamId, {
          type: f.type,
          title: title.value!,
          amount: amount.value!,
          category: f.category,
        });
      await loadFinances();
      setState({ busy: null });
      // Don't close a sheet the user has since opened for a different team
      // after switching away mid-request.
      if (S().activeTeamId === teamId) setState({ sheet: null });
      toastMsg(t('finances.toastTxSaved'));
    } catch (err) {
      reportActionError({ setState, toastMsg, onAuthError: logout }, err, 'error.save');
    }
  }, [api, S, setState, loadFinances, toastMsg, logout]);

  const deleteTx = useCallback(
    async (id: string) => {
      const teamId = S().activeTeamId!;
      setState({ busy: 'delete' });
      try {
        await api.finances.deleteTransaction(id, teamId);
        await loadFinances();
        setState({ busy: null });
        if (S().activeTeamId === teamId) setState({ sheet: null });
        toastMsg(t('finances.toastTxDeleted'));
      } catch (err) {
        reportActionError({ setState, toastMsg, onAuthError: logout }, err, 'error.delete');
      }
    },
    [api, S, loadFinances, setState, toastMsg, logout],
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
      })),
    [setState],
  );

  const savePenalty = useCallback(async () => {
    const f = S().form as PenaltyFormValues;
    const label = validateRequiredText(f.label, t('finances.penaltyFieldLabelError'));
    if (!label.ok) {
      toastMsg(label.message!);
      return;
    }
    const amount = validateMoneyAmount(f.amount, { positive: true });
    if (!amount.ok) {
      toastMsg(amount.message!);
      return;
    }
    const sh = S().sheet!;
    const back = sh.back || null;
    const create = sh.mode === 'create';
    const teamId = S().activeTeamId!;
    setState({ busy: 'save' });
    try {
      if (create) await api.finances.createPenalty(teamId, { label: label.value!, amount: amount.value! });
      else await api.finances.updatePenalty(f.id!, { label: label.value!, amount: amount.value! }, teamId);
      await loadFinances();
      setState({ busy: null });
      // Don't navigate away from a sheet the user has since opened for a
      // different team after switching away mid-request.
      if (S().activeTeamId === teamId) setState({ sheet: back });
      toastMsg(create ? t('finances.toastPenaltyAdded') : t('finances.toastPenaltySaved'));
    } catch (err) {
      reportActionError({ setState, toastMsg, onAuthError: logout }, err, 'error.save');
    }
  }, [api, S, setState, loadFinances, toastMsg, logout]);

  const deletePenaltyDef = useCallback(
    (id: string) =>
      askConfirm({
        title: t('finances.penaltyDeleteTitle'),
        message: t('finances.penaltyDeleteMsg'),
        confirmLabel: t('finances.penaltyDeleteConfirm'),
        danger: true,
        onConfirm: async () => {
          const teamId = S().activeTeamId!;
          try {
            await api.finances.deletePenalty(id, teamId);
            await loadFinances();
            if (S().activeTeamId === teamId) setState({ sheet: { type: 'penaltyCatalog' } });
            toastMsg(t('finances.toastPenaltyRemoved'));
          } catch (err) {
            reportActionError({ setState, toastMsg, onAuthError: logout }, err, 'error.delete');
          }
        },
      }),
    [api, S, askConfirm, loadFinances, setState, toastMsg, logout],
  );

  const openPenaltyAssign = useCallback(() => {
    const members = S().members;
    if (!members || !members.length) refreshMembers();
    const f = S().finances;
    const first = f && f.penalties[0] ? f.penalties[0].id : null;
    const form: PenaltyAssignFormValues = { userId: '', penaltyId: first };
    setState({ sheet: { type: 'penaltyAssign' }, form });
  }, [S, refreshMembers, setState]);

  const savePenaltyAssign = useCallback(async () => {
    const f = S().form as PenaltyAssignFormValues;
    if (!f.userId) {
      toastMsg(t('finances.assignPersonError'));
      return;
    }
    if (!f.penaltyId) {
      toastMsg(t('finances.assignPenaltyError'));
      return;
    }
    const teamId = S().activeTeamId!;
    setState({ busy: 'save' });
    try {
      await api.finances.assignPenalty(teamId, { userId: f.userId, penaltyId: f.penaltyId });
      await loadFinances();
      setState({ busy: null });
      if (S().activeTeamId === teamId) setState({ sheet: null });
      toastMsg(t('finances.toastPenaltyAssigned'));
    } catch (err) {
      reportActionError({ setState, toastMsg, onAuthError: logout }, err, 'error.save');
    }
  }, [api, S, setState, loadFinances, toastMsg, logout]);

  const deleteAssignment = useCallback(
    async (id: string) => {
      try {
        await api.finances.deleteAssignment(id, S().activeTeamId!);
        await loadFinances();
        toastMsg(t('finances.toastPenaltyAssignDeleted'));
      } catch (err) {
        reportActionError({ setState, toastMsg, onAuthError: logout }, err, 'error.delete');
      }
    },
    [api, S, loadFinances, setState, toastMsg, logout],
  );

  const openContribForm = useCallback(
    (c: Contribution) => {
      const form: ContribFormValues = { id: c.id, label: c.label, amount: String(c.amount) };
      setState({ sheet: { type: 'contribForm' }, form });
    },
    [setState],
  );

  const saveContrib = useCallback(async () => {
    const f = S().form as ContribFormValues;
    const label = validateRequiredText(f.label, t('finances.contribFieldLabelError'));
    if (!label.ok) {
      toastMsg(label.message!);
      return;
    }
    const amount = validateMoneyAmount(f.amount, { positive: true });
    if (!amount.ok) {
      toastMsg(amount.message!);
      return;
    }
    const teamId = S().activeTeamId!;
    setState({ busy: 'save' });
    try {
      await api.finances.updateContribution(f.id, { label: label.value!, amount: amount.value! }, teamId);
      await loadFinances();
      setState({ busy: null });
      if (S().activeTeamId === teamId) setState({ sheet: null });
      toastMsg(t('finances.toastContribSaved'));
    } catch (err) {
      reportActionError({ setState, toastMsg, onAuthError: logout }, err, 'error.save');
    }
  }, [api, S, setState, loadFinances, toastMsg, logout]);

  const toggleInFlight = useRef(new Set<string>());

  const togglePenalty = useCallback(
    async (id: string) => {
      const key = 'penalty:' + id;
      if (toggleInFlight.current.has(key)) return;
      toggleInFlight.current.add(key);
      try {
        await api.finances.togglePenaltyPaid(id, S().activeTeamId!);
        await loadFinances();
      } catch (err) {
        reportActionError({ setState, toastMsg, onAuthError: logout }, err);
      } finally {
        toggleInFlight.current.delete(key);
      }
    },
    [api, S, loadFinances, setState, toastMsg, logout],
  );

  const toggleContribution = useCallback(
    async (id: string) => {
      const key = 'contribution:' + id;
      if (toggleInFlight.current.has(key)) return;
      toggleInFlight.current.add(key);
      try {
        await api.finances.toggleContribution(id, S().activeTeamId!);
        await loadFinances();
      } catch (err) {
        reportActionError({ setState, toastMsg, onAuthError: logout }, err);
      } finally {
        toggleInFlight.current.delete(key);
      }
    },
    [api, S, loadFinances, setState, toastMsg, logout],
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
  };
}
