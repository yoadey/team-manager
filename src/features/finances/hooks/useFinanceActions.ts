/* eslint-disable @typescript-eslint/no-explicit-any */
import { useCallback } from 'react';
import type { api as defaultApi } from '@/services/serviceLayer';
import type { DateRange } from '@/types';
import type { AppState } from '@/context/AppContext';
import { validateMoneyAmount, validateRequiredText } from '@/utils/validation';
import { reportActionError } from '@/utils/errors';

type SetState = (patch: Partial<AppState> | ((s: AppState) => Partial<AppState>)) => void;

type FinanceFeatureDeps = {
  api: typeof defaultApi;
  S: () => AppState;
  setState: SetState;
  loadFinances: () => Promise<void>;
  loadStats: (range?: DateRange | null) => Promise<void>;
  refreshMembers: () => Promise<void>;
  askConfirm: (cfg: {
    title: string;
    message: string;
    confirmLabel?: string;
    danger?: boolean;
    onConfirm: () => void | Promise<void>;
  }) => void;
  toastMsg: (m: string) => void;
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
}: FinanceFeatureDeps) {
  const openTxForm = useCallback(
    (tx?: any) => {
      const f = tx
        ? { id: tx.id, type: tx.type, title: tx.title, amount: String(tx.amount), category: tx.category }
        : { type: 'income', title: '', amount: '', category: 'Beiträge' };
      setState({ sheet: { type: 'txForm', mode: tx ? 'edit' : 'create' }, form: f });
    },
    [setState],
  );

  const saveTx = useCallback(async () => {
    const f = S().form;
    const title = validateRequiredText(f.title, 'Bezeichnung der Buchung fehlt.');
    if (!title.ok) {
      toastMsg(title.message!);
      return;
    }
    const amount = validateMoneyAmount(f.amount, { field: 'Betrag der Buchung', positive: true });
    if (!amount.ok) {
      toastMsg(amount.message!);
      return;
    }
    setState({ busy: 'save' });
    try {
      if (S().sheet!.mode === 'edit')
        await api.finances.updateTransaction(f.id, {
          type: f.type,
          title: title.value!,
          amount: amount.value!,
          category: f.category,
        });
      else
        await api.finances.addTransaction(S().activeTeamId!, {
          type: f.type,
          title: title.value!,
          amount: amount.value!,
          category: f.category,
        });
      await loadFinances();
      setState({ busy: null, sheet: null });
      toastMsg('Buchung gespeichert');
    } catch (err) {
      reportActionError({ setState, toastMsg }, err, 'error.save');
    }
  }, [api, S, setState, loadFinances, toastMsg]);

  const deleteTx = useCallback(
    async (id: string) => {
      setState({ busy: 'delete' });
      try {
        await api.finances.deleteTransaction(id);
        await loadFinances();
        setState({ busy: null, sheet: null });
        toastMsg('Buchung gelöscht');
      } catch (err) {
        reportActionError({ setState, toastMsg }, err, 'error.delete');
      }
    },
    [api, loadFinances, setState, toastMsg],
  );

  const openPenaltyCatalog = useCallback(() => setState({ sheet: { type: 'penaltyCatalog' } }), [setState]);

  const openPenaltyForm = useCallback(
    (p?: any) =>
      setState((st) => ({
        sheet: {
          type: 'penaltyForm',
          mode: p ? 'edit' : 'create',
          back: st.sheet && st.sheet.type === 'penaltyCatalog' ? st.sheet : null,
        },
        form: p ? { id: p.id, label: p.label, amount: String(p.amount) } : { label: '', amount: '' },
      })),
    [setState],
  );

  const savePenalty = useCallback(async () => {
    const f = S().form;
    const label = validateRequiredText(f.label, 'Bezeichnung der Strafe fehlt.');
    if (!label.ok) {
      toastMsg(label.message!);
      return;
    }
    const amount = validateMoneyAmount(f.amount, { field: 'Betrag der Strafe', positive: true });
    if (!amount.ok) {
      toastMsg(amount.message!);
      return;
    }
    const sh = S().sheet!;
    const back = sh.back || null;
    const create = sh.mode === 'create';
    setState({ busy: 'save' });
    try {
      if (create) await api.finances.createPenalty(S().activeTeamId!, { label: label.value!, amount: amount.value! });
      else await api.finances.updatePenalty(f.id, { label: label.value!, amount: amount.value! });
      await loadFinances();
      setState({ busy: null, sheet: back });
      toastMsg(create ? 'Strafe hinzugefügt' : 'Strafe gespeichert');
    } catch (err) {
      reportActionError({ setState, toastMsg }, err, 'error.save');
    }
  }, [api, S, setState, loadFinances, toastMsg]);

  const deletePenaltyDef = useCallback(
    (id: string) =>
      askConfirm({
        title: 'Strafe entfernen?',
        message: 'Diese Strafe wird aus dem Katalog entfernt. Bereits erfasste Strafen bleiben erhalten.',
        confirmLabel: 'Entfernen',
        danger: true,
        onConfirm: async () => {
          try {
            await api.finances.deletePenalty(id);
            await loadFinances();
            setState({ sheet: { type: 'penaltyCatalog' } });
            toastMsg('Strafe entfernt');
          } catch (err) {
            reportActionError({ setState, toastMsg }, err, 'error.delete');
          }
        },
      }),
    [api, askConfirm, loadFinances, setState, toastMsg],
  );

  const openPenaltyAssign = useCallback(() => {
    if (!S().members || !S().members.length) refreshMembers();
    const f = S().finances;
    const first = f && f.penalties[0] ? f.penalties[0].id : null;
    setState({ sheet: { type: 'penaltyAssign' }, form: { userId: '', penaltyId: first } });
  }, [S, refreshMembers, setState]);

  const savePenaltyAssign = useCallback(async () => {
    const f = S().form;
    if (!f.userId) {
      toastMsg('Bitte Person wählen');
      return;
    }
    if (!f.penaltyId) {
      toastMsg('Bitte Strafe wählen');
      return;
    }
    setState({ busy: 'save' });
    try {
      await api.finances.assignPenalty(S().activeTeamId!, { userId: f.userId, penaltyId: f.penaltyId });
      await loadFinances();
      setState({ busy: null, sheet: null });
      toastMsg('Strafe erfasst');
    } catch (err) {
      reportActionError({ setState, toastMsg }, err, 'error.save');
    }
  }, [api, S, setState, loadFinances, toastMsg]);

  const deleteAssignment = useCallback(
    async (id: string) => {
      try {
        await api.finances.deleteAssignment(id);
        await loadFinances();
        toastMsg('Strafe gelöscht');
      } catch (err) {
        reportActionError({ setState, toastMsg }, err, 'error.delete');
      }
    },
    [api, loadFinances, setState, toastMsg],
  );

  const openContribForm = useCallback(
    (c: any) =>
      setState({
        sheet: { type: 'contribForm' },
        form: { id: c.id, label: c.label, amount: String(c.amount) },
      }),
    [setState],
  );

  const saveContrib = useCallback(async () => {
    const f = S().form;
    const label = validateRequiredText(f.label, 'Bezeichnung des Beitrags fehlt.');
    if (!label.ok) {
      toastMsg(label.message!);
      return;
    }
    const amount = validateMoneyAmount(f.amount, { field: 'Betrag des Beitrags', positive: true });
    if (!amount.ok) {
      toastMsg(amount.message!);
      return;
    }
    setState({ busy: 'save' });
    try {
      await api.finances.updateContribution(f.id, { label: label.value!, amount: amount.value! });
      await loadFinances();
      setState({ busy: null, sheet: null });
      toastMsg('Beitrag gespeichert');
    } catch (err) {
      reportActionError({ setState, toastMsg }, err, 'error.save');
    }
  }, [api, S, setState, loadFinances, toastMsg]);

  const togglePenalty = useCallback(
    async (id: string) => {
      try {
        await api.finances.togglePenaltyPaid(id);
        await loadFinances();
      } catch (err) {
        reportActionError({ setState, toastMsg }, err);
      }
    },
    [api, loadFinances, setState, toastMsg],
  );

  const toggleContribution = useCallback(
    async (id: string) => {
      try {
        await api.finances.toggleContribution(id);
        await loadFinances();
      } catch (err) {
        reportActionError({ setState, toastMsg }, err);
      }
    },
    [api, loadFinances, setState, toastMsg],
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
