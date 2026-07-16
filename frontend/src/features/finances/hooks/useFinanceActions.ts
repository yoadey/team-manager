import { useCallback, useRef } from 'react';
import type { api as defaultApi } from '@/services';
import type { DateRange } from '@/types';
import type { AppState, ConfirmConfig } from '@/context/AppContext';
import type { Penalty, Contribution, Transaction } from '../types';
import { reportActionError } from '@/utils/errors';
import { clearBusyIfOwned } from '@/utils/forms';
import { t } from '@/i18n';
import type { TxFormValues } from '../components/txFormSchema';
import type { PenaltyFormValues } from '../components/penaltyFormSchema';
import type { PenaltyAssignFormValues } from '../components/penaltyAssignFormSchema';
import type { ContribFormValues } from '../components/contribFormSchema';
import { validateMoneyAmount, MAX_MONEY_AMOUNT_EUROS } from '@/utils/validation';

type SetState = (patch: Partial<AppState> | ((s: AppState) => Partial<AppState>)) => void;

type FinanceFeatureDeps = {
  api: typeof defaultApi;
  S: () => AppState;
  setState: SetState;
  loadFinances: () => Promise<void>;
  loadStats: (range?: DateRange | null) => Promise<void>;
  refreshMembers: () => Promise<void>;
  askConfirm: (cfg: ConfirmConfig) => void;
  toastMsg: (m: string, action?: { label: string; fn: () => void }, kind?: 'success' | 'error') => void;
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
        : { type: 'income', title: '', amount: '', category: '' };
      setState({ sheet: { type: 'txForm', mode: tx ? 'edit' : 'create' }, form: f, formErrors: {} });
    },
    [setState],
  );

  const saveTx = useCallback(async (fProp?: any) => {
    const f = fProp !== undefined ? fProp : (S().form as TxFormValues);
    if (!f.title || !f.title.trim()) {
      toastMsg(t('finances.txFieldTitleError'), undefined, 'error');
      return;
    }
    const amountRes = validateMoneyAmount(f.amount, { positive: true, max: MAX_MONEY_AMOUNT_EUROS });
    if (!amountRes.ok) {
      toastMsg(amountRes.message!, undefined, 'error');
      return;
    }
    const amountVal = amountRes.value!;
    const sh = S().sheet;
    const mode = sh ? sh.mode : 'create';
    const teamId = S().activeTeamId!;
    setState({ busy: 'save' });
    try {
      if (mode === 'edit')
        await api.finances.updateTransaction(
          f.id!,
          {
            type: f.type,
            title: f.title.trim(),
            amount: amountVal,
            category: f.category || '',
          },
          teamId,
        );
      else
        await api.finances.addTransaction(teamId, {
          type: f.type,
          title: f.title.trim(),
          amount: amountVal,
          category: f.category || '',
        });
      await loadFinances();
      clearBusyIfOwned(S, setState, 'save');
      // Don't close a sheet the user has since opened for a different team
      // after switching away mid-request, or one they've since opened for a
      // different entity (same team) while this save was in flight.
      if (S().activeTeamId === teamId && S().sheet === sh) setState({ sheet: null });
      toastMsg(t('finances.toastTxSaved'));
    } catch (err) {
      reportActionError({ setState, toastMsg, onAuthError: logout, S, busyOwner: 'save' }, err, 'error.save');
      if (fProp !== undefined) throw err;
    } finally {
      setState({ busy: null });
    }
  }, [api, S, setState, loadFinances, toastMsg, logout]);

  const deleteTx = useCallback(
    async (id: string) => {
      const sh = S().sheet;
      const teamId = S().activeTeamId!;
      setState({ busy: 'delete' });
      try {
        await api.finances.deleteTransaction(id, teamId);
        await loadFinances();
        if (S().activeTeamId === teamId && S().sheet === sh) setState({ sheet: null });
        toastMsg(t('finances.toastTxDeleted'));
      } catch (err) {
        reportActionError({ setState, toastMsg, onAuthError: logout, S }, err, 'error.delete');
      } finally {
        setState({ busy: null });
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
        formErrors: {},
      })),
    [setState],
  );

  const savePenalty = useCallback(async (fProp?: any) => {
    const f = fProp !== undefined ? fProp : (S().form as PenaltyFormValues);
    if (!f.label || !f.label.trim()) {
      toastMsg(t('finances.penaltyFieldLabelError'), undefined, 'error');
      return;
    }
    const amountRes = validateMoneyAmount(f.amount, { positive: true, max: MAX_MONEY_AMOUNT_EUROS });
    if (!amountRes.ok) {
      toastMsg(amountRes.message!, undefined, 'error');
      return;
    }
    const amountVal = amountRes.value!;
    const sh = S().sheet!;
    const back = sh.back || null;
    const create = sh.mode === 'create';
    const teamId = S().activeTeamId!;
    setState({ busy: 'save' });
    try {
      if (create) await api.finances.createPenalty(teamId, { label: f.label.trim(), amount: amountVal });
      else await api.finances.updatePenalty(f.id!, { label: f.label.trim(), amount: amountVal }, teamId);
      await loadFinances();
      clearBusyIfOwned(S, setState, 'save');
      if (S().activeTeamId === teamId && S().sheet === sh) setState({ sheet: back });
      toastMsg(create ? t('finances.toastPenaltyAdded') : t('finances.toastPenaltySaved'));
    } catch (err) {
      reportActionError({ setState, toastMsg, onAuthError: logout, S, busyOwner: 'save' }, err, 'error.save');
      if (fProp !== undefined) throw err;
    } finally {
      setState({ busy: null });
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
          const sh = S().sheet;
          const teamId = S().activeTeamId!;
          try {
            await api.finances.deletePenalty(id, teamId);
            await loadFinances();
            if (S().activeTeamId === teamId && S().sheet === sh) setState({ sheet: { type: 'penaltyCatalog' } });
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
    const first = f && f.penalties[0] ? f.penalties[0].id : '';
    const form: PenaltyAssignFormValues = { userId: '', penaltyId: first };
    setState({ sheet: { type: 'penaltyAssign' }, form, formErrors: {} });
  }, [S, refreshMembers, setState]);

  const savePenaltyAssign = useCallback(async (fProp?: any) => {
    const f = fProp !== undefined ? fProp : (S().form as PenaltyAssignFormValues);
    if (!f.userId) {
      toastMsg(t('finances.assignPersonError'), undefined, 'error');
      return;
    }
    if (!f.penaltyId) {
      toastMsg(t('finances.assignPenaltyError'), undefined, 'error');
      return;
    }
    const sh = S().sheet;
    const teamId = S().activeTeamId!;
    setState({ busy: 'save' });
    try {
      await api.finances.assignPenalty(teamId, { userId: f.userId, penaltyId: f.penaltyId });
      await loadFinances();
      clearBusyIfOwned(S, setState, 'save');
      if (S().activeTeamId === teamId && S().sheet === sh) setState({ sheet: null });
      toastMsg(t('finances.toastPenaltyAssigned'));
    } catch (err) {
      reportActionError({ setState, toastMsg, onAuthError: logout, S, busyOwner: 'save' }, err, 'error.save');
      if (fProp !== undefined) throw err;
    } finally {
      setState({ busy: null });
    }
  }, [api, S, setState, loadFinances, toastMsg, logout]);

  const deleteAssignment = useCallback(
    (id: string) =>
      askConfirm({
        title: t('finances.assignmentDeleteTitle'),
        message: t('finances.assignmentDeleteMsg'),
        confirmLabel: t('finances.assignmentDeleteConfirm'),
        danger: true,
        onConfirm: async () => {
          try {
            await api.finances.deleteAssignment(id, S().activeTeamId!);
            await loadFinances();
            toastMsg(t('finances.toastPenaltyAssignDeleted'));
          } catch (err) {
            reportActionError({ setState, toastMsg, onAuthError: logout }, err, 'error.delete');
          }
        },
      }),
    [api, S, askConfirm, loadFinances, setState, toastMsg, logout],
  );

  const openContribForm = useCallback(
    (c: Contribution) => {
      const form: ContribFormValues = { id: c.id, label: c.label, amount: String(c.amount) };
      setState({ sheet: { type: 'contribForm' }, form, formErrors: {} });
    },
    [setState],
  );

  const saveContrib = useCallback(async (fProp?: any) => {
    const f = fProp !== undefined ? fProp : (S().form as ContribFormValues);
    if (!f.label || !f.label.trim()) {
      toastMsg(t('finances.contribFieldLabelError'), undefined, 'error');
      return;
    }
    const amountRes = validateMoneyAmount(f.amount, { positive: true, max: MAX_MONEY_AMOUNT_EUROS });
    if (!amountRes.ok) {
      toastMsg(amountRes.message!, undefined, 'error');
      return;
    }
    const amountVal = amountRes.value!;
    const sh = S().sheet;
    const teamId = S().activeTeamId!;
    setState({ busy: 'save' });
    try {
      await api.finances.updateContribution(f.id, { label: f.label.trim(), amount: amountVal }, teamId);
      await loadFinances();
      clearBusyIfOwned(S, setState, 'save');
      if (S().activeTeamId === teamId && S().sheet === sh) setState({ sheet: null });
      toastMsg(t('finances.toastContribSaved'));
    } catch (err) {
      reportActionError({ setState, toastMsg, onAuthError: logout, S, busyOwner: 'save' }, err, 'error.save');
      if (fProp !== undefined) throw err;
    } finally {
      setState({ busy: null });
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
