import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act, waitFor } from '@testing-library/react';
import { useFinanceActions } from './useFinanceActions';
import { createQueryWrapper, createTestQueryClient } from '@/test/queryTestUtils';
import { queryKeys } from '@/query/keys';
import type { AppState } from '@/context/AppContext';
import type { TxFormValues } from '../components/txFormSchema';
import type { PenaltyFormValues } from '../components/penaltyFormSchema';
import type { PenaltyAssignFormValues } from '../components/penaltyAssignFormSchema';
import type { ContribGroupEditFormValues } from '../components/contribGroupEditFormSchema';
import type { Contribution, FinanceOverview } from '../types';
import type { QueryClient } from '@tanstack/react-query';
import { todayStr } from '@/styles/tokens';

function makeOverview(overrides: Partial<FinanceOverview> = {}): FinanceOverview {
  return {
    balance: 0,
    income: 0,
    expense: 0,
    transactions: [],
    penalties: [],
    assignments: [],
    openPenalties: [],
    openPenaltySum: 0,
    contributions: [],
    contribOpen: 0,
    ...overrides,
  };
}

function makeState(overrides: Partial<AppState> = {}): AppState {
  return {
    phase: 'app',
    user: { id: 'u1', name: 'Test User', email: 'test@test.com', avatarColor: '#000', photo: null },
    activeTeamId: 'team1',
    sheet: null,
    busy: null,
    toast: null,
    route: 'home',
    events: [],
    stats: null,
    statsRange: null,
    news: [],
    polls: [],
    teams: [],
    roles: [],
    notifUnread: 0,
    notifications: [],
    primaryColor: '#000',
    ...overrides,
  } as unknown as AppState;
}

function makeApi() {
  return {
    finances: {
      addTransaction: vi.fn().mockResolvedValue({ id: 'tx1' }),
      updateTransaction: vi.fn().mockResolvedValue(undefined),
      deleteTransaction: vi.fn().mockResolvedValue(undefined),
      createPenalty: vi.fn().mockResolvedValue({ id: 'p1' }),
      updatePenalty: vi.fn().mockResolvedValue(undefined),
      deletePenalty: vi.fn().mockResolvedValue(undefined),
      assignPenalty: vi.fn().mockResolvedValue(undefined),
      deleteAssignment: vi.fn().mockResolvedValue(undefined),
      updateContribution: vi.fn().mockResolvedValue(undefined),
      createContributions: vi.fn().mockResolvedValue([{ id: 'c1' }]),
      deleteContribution: vi.fn().mockResolvedValue(undefined),
    },
  };
}

describe('useFinanceActions', () => {
  let setState: ReturnType<typeof vi.fn>;
  let toastMsg: ReturnType<typeof vi.fn>;
  let askConfirm: ReturnType<typeof vi.fn>;
  let logout: ReturnType<typeof vi.fn>;
  let api: ReturnType<typeof makeApi>;
  let stateRef: AppState;
  let client: QueryClient;

  beforeEach(() => {
    stateRef = makeState();
    setState = vi.fn((patch) => {
      if (typeof patch === 'function') {
        const result = patch(stateRef);
        stateRef = { ...stateRef, ...result };
      } else {
        stateRef = { ...stateRef, ...patch };
      }
    });
    toastMsg = vi.fn();
    askConfirm = vi.fn();
    logout = vi.fn();
    api = makeApi();
    client = createTestQueryClient();
    client.setQueryData(queryKeys.finances('team1'), makeOverview({ penalties: [{ id: 'p1' } as never] }));
  });

  function renderActions() {
    return renderHook(
      () =>
        useFinanceActions({
          api: api as never,
          S: () => stateRef,
          setState: setState as never,
          teamId: stateRef.activeTeamId,
          askConfirm: askConfirm as never,
          toastMsg: toastMsg as never,
          logout: logout as never,
        }),
      { wrapper: createQueryWrapper(client) },
    );
  }

  it('openTxForm sets create sheet with empty form', () => {
    const { result } = renderActions();
    act(() => {
      result.current.openTxForm();
    });
    expect(setState).toHaveBeenCalledWith(
      expect.objectContaining({
        sheet: expect.objectContaining({ type: 'txForm', mode: 'create' }),
      }),
    );
  });

  // Regression test: a new transaction's category used to be prefilled with
  // the literal German word 'Beiträge' as an actual form VALUE, independent
  // of the active UI locale -- same bug class as the round-75 absence-reason
  // fix. An English-locale user creating a transaction saw an already-filled
  // German word instead of the already-translated txCategoryPlaceholder
  // hint (which never renders once the field has a value).
  it('openTxForm defaults a new transaction category to empty (not a hardcoded locale-specific value)', () => {
    const { result } = renderActions();
    act(() => {
      result.current.openTxForm();
    });
    expect(setState).toHaveBeenCalledWith(
      expect.objectContaining({
        sheet: expect.objectContaining({ formInitial: expect.objectContaining({ category: '' }) }),
      }),
    );
  });

  it('openTxForm defaults a new transaction date to today', () => {
    const { result } = renderActions();
    act(() => {
      result.current.openTxForm();
    });
    expect(setState).toHaveBeenCalledWith(
      expect.objectContaining({
        sheet: expect.objectContaining({ formInitial: expect.objectContaining({ date: todayStr() }) }),
      }),
    );
  });

  it('openTxForm carries an existing transaction date into the edit form', () => {
    const tx = { id: 'tx1', type: 'income', title: 'Test', amount: 50, category: '', date: '2025-03-01' } as never;
    const { result } = renderActions();
    act(() => {
      result.current.openTxForm(tx);
    });
    expect(setState).toHaveBeenCalledWith(
      expect.objectContaining({
        sheet: expect.objectContaining({ formInitial: expect.objectContaining({ date: '2025-03-01' }) }),
      }),
    );
  });

  it('openTxFormForContribution pre-links a create-mode form to the contribution', () => {
    const c = {
      id: 'c1',
      label: 'Monatsbeitrag',
      amount: 20,
      paidAmount: 5,
      userId: 'u1',
      teamId: 'team1',
    } as Contribution;
    const { result } = renderActions();
    act(() => {
      result.current.openTxFormForContribution(c);
    });
    expect(setState).toHaveBeenCalledWith({
      sheet: {
        type: 'txForm',
        mode: 'create',
        formInitial: {
          type: 'income',
          title: 'Monatsbeitrag',
          amount: '15',
          category: '',
          date: todayStr(),
          contributionId: 'c1',
          penaltyAssignmentId: '',
          note: '',
        },
      },
    });
  });

  it('openTxForm sets edit sheet when transaction passed', () => {
    const tx = { id: 'tx1', type: 'income', title: 'Test', amount: 50, category: 'Beiträge' } as never;
    const { result } = renderActions();
    act(() => {
      result.current.openTxForm(tx);
    });
    expect(setState).toHaveBeenCalledWith(
      expect.objectContaining({
        sheet: expect.objectContaining({ type: 'txForm', mode: 'edit' }),
      }),
    );
  });

  // Regression: openTxForm's edit-mode formInitial used to omit
  // contributionId/penaltyAssignmentId, so TxFormSheet's "linked to" info
  // could never resolve a linked contribution/penalty for an already-linked
  // transaction reopened for editing -- see finance-detail-linked-entries.
  it('openTxForm carries an existing contributionId/penaltyAssignmentId into the edit form', () => {
    const tx = {
      id: 'tx1',
      type: 'income',
      title: 'Beitrag',
      amount: 25,
      category: 'Beiträge',
      contributionId: 'c1',
      penaltyAssignmentId: null,
    } as never;
    const { result } = renderActions();
    act(() => {
      result.current.openTxForm(tx);
    });
    expect(setState).toHaveBeenCalledWith(
      expect.objectContaining({
        sheet: expect.objectContaining({
          formInitial: expect.objectContaining({ contributionId: 'c1', penaltyAssignmentId: '' }),
        }),
      }),
    );
  });

  it('saveTx creates transaction in create mode', async () => {
    const formValues = { title: 'Beitrag Jan', amount: '50', type: 'income', category: 'Beiträge' } as TxFormValues;
    stateRef = makeState({
      sheet: { type: 'txForm', mode: 'create', formInitial: formValues } as never,
    });
    const { result } = renderActions();
    await act(async () => {
      await result.current.saveTx(formValues);
    });
    expect(api.finances.addTransaction).toHaveBeenCalled();
    expect(toastMsg).toHaveBeenCalledWith('Buchung gespeichert');
  });

  it('saveTx includes the chosen date in the payload', async () => {
    const formValues = {
      title: 'Beitrag Jan',
      amount: '50',
      type: 'income',
      category: 'Beiträge',
      date: '2025-05-10',
    } as TxFormValues;
    stateRef = makeState({
      sheet: { type: 'txForm', mode: 'create', formInitial: formValues } as never,
    });
    const { result } = renderActions();
    await act(async () => {
      await result.current.saveTx(formValues);
    });
    expect(api.finances.addTransaction).toHaveBeenCalledWith('team1', expect.objectContaining({ date: '2025-05-10' }));
  });

  it('saveTx updates transaction in edit mode', async () => {
    const formValues = {
      id: 'tx1',
      title: 'Updated',
      amount: '75',
      type: 'expense',
      category: 'Ausrüstung',
    } as TxFormValues;
    stateRef = makeState({
      sheet: { type: 'txForm', mode: 'edit', formInitial: formValues } as never,
    });
    const { result } = renderActions();
    await act(async () => {
      await result.current.saveTx(formValues);
    });
    expect(api.finances.updateTransaction).toHaveBeenCalledWith(
      'tx1',
      expect.objectContaining({ title: 'Updated' }),
      'team1',
    );
  });

  // Regression: a slow saveTx used to unconditionally close the sheet once
  // it resolved, as long as the team hadn't changed -- so closing this
  // transaction's edit form and opening a different sheet (e.g. a different
  // transaction, same team) while the save was still in flight would get
  // silently clobbered by the stale save once it finally resolved.
  it('saveTx does not touch the sheet if the user opened something else while the save was in flight', async () => {
    let resolveUpdate!: () => void;
    api.finances.updateTransaction = vi.fn(() => new Promise<void>((resolve) => (resolveUpdate = resolve)));
    const formValues = {
      id: 'tx1',
      title: 'Updated',
      amount: '75',
      type: 'expense',
      category: 'Ausrüstung',
    } as TxFormValues;
    stateRef = makeState({
      sheet: { type: 'txForm', mode: 'edit', formInitial: formValues } as never,
    });
    const { result } = renderActions();

    let savePromise!: Promise<void>;
    act(() => {
      savePromise = result.current.saveTx(formValues);
    });
    await waitFor(() => expect(api.finances.updateTransaction).toHaveBeenCalled());

    const otherTxForm = { type: 'txForm', mode: 'edit' } as never;
    stateRef = { ...stateRef, sheet: otherTxForm };

    await act(async () => {
      resolveUpdate();
      await savePromise;
    });

    expect(stateRef.sheet).toBe(otherTxForm);
  });

  it('deleteTx calls deleteTransaction and shows toast', async () => {
    const { result } = renderActions();
    await act(async () => {
      await result.current.deleteTx('tx1');
    });
    expect(api.finances.deleteTransaction).toHaveBeenCalledWith('tx1', 'team1');
    expect(toastMsg).toHaveBeenCalledWith('Buchung gelöscht');
  });

  it('openPenaltyCatalog sets penaltyCatalog sheet', () => {
    const { result } = renderActions();
    act(() => {
      result.current.openPenaltyCatalog();
    });
    expect(setState).toHaveBeenCalledWith({ sheet: { type: 'penaltyCatalog' } });
  });

  it('openPenaltyForm sets create sheet when no penalty', () => {
    const { result } = renderActions();
    act(() => {
      result.current.openPenaltyForm();
    });
    expect(setState).toHaveBeenCalled();
  });

  it('savePenalty creates penalty in create mode', async () => {
    const formValues = { label: 'Zu spät', amount: '5' } as PenaltyFormValues;
    stateRef = makeState({
      sheet: { type: 'penaltyForm', mode: 'create', back: null, formInitial: formValues } as never,
    });
    const { result } = renderActions();
    await act(async () => {
      await result.current.savePenalty(formValues);
    });
    expect(api.finances.createPenalty).toHaveBeenCalled();
    expect(toastMsg).toHaveBeenCalledWith('Strafe hinzugefügt');
  });

  it('deletePenaltyDef calls askConfirm', () => {
    const { result } = renderActions();
    act(() => {
      result.current.deletePenaltyDef('p1');
    });
    expect(askConfirm).toHaveBeenCalledWith(expect.objectContaining({ danger: true }));
  });

  it('savePenaltyAssign assigns penalty when valid', async () => {
    const { result } = renderActions();
    await act(async () => {
      await result.current.savePenaltyAssign({ userId: 'u1', penaltyId: 'p1' } as PenaltyAssignFormValues);
    });
    expect(api.finances.assignPenalty).toHaveBeenCalled();
    expect(toastMsg).toHaveBeenCalledWith('Strafe erfasst');
  });

  // Regression test: deleteAssignment used to call the API directly with no
  // confirmation, unlike every other destructive action in this file
  // (deletePenaltyDef etc.), so a single misclick permanently deleted a
  // penalty-assignment record with no "are you sure."
  it('deleteAssignment asks for confirmation before calling the API', () => {
    const { result } = renderActions();
    act(() => {
      result.current.deleteAssignment('a1');
    });
    expect(askConfirm).toHaveBeenCalledWith(expect.objectContaining({ danger: true }));
    expect(api.finances.deleteAssignment).not.toHaveBeenCalled();
  });

  it('deleteAssignment calls the API once confirmed', async () => {
    const { result } = renderActions();
    act(() => {
      result.current.deleteAssignment('a1');
    });
    const onConfirm = askConfirm.mock.calls[0]![0].onConfirm;
    await act(async () => {
      await onConfirm();
    });
    expect(api.finances.deleteAssignment).toHaveBeenCalledWith('a1', 'team1');
  });

  // Regression test: mirrors useDeleteEventMutation/useRemoveMemberMutation's
  // per-call teamId safeguard. The confirm sheet can still be open (and get
  // confirmed) after the user has switched to a different active team; the
  // delete must still target the team the confirm dialog was opened for.
  it('deleteAssignment deletes against the team the confirm dialog was opened for, even after a team switch', async () => {
    const { result, rerender } = renderActions();
    act(() => {
      result.current.deleteAssignment('a1');
    });
    const onConfirm = askConfirm.mock.calls[0]![0].onConfirm;

    stateRef = { ...stateRef, activeTeamId: 'team2' };
    rerender();

    await act(async () => {
      await onConfirm();
    });
    expect(api.finances.deleteAssignment).toHaveBeenCalledWith('a1', 'team1');
  });


  it('editContribGroup fans the update out over every row in the group', async () => {
    const { result } = renderActions();
    const rows = [{ id: 'c1' }, { id: 'c2' }] as Contribution[];
    await act(async () => {
      await result.current.editContribGroup(rows, {
        label: 'Monatsbeitrag',
        amount: '20',
        dueDate: '',
      } as ContribGroupEditFormValues);
    });
    expect(api.finances.updateContribution).toHaveBeenCalledWith(
      'c1',
      expect.objectContaining({ label: 'Monatsbeitrag' }),
      'team1',
    );
    expect(api.finances.updateContribution).toHaveBeenCalledWith(
      'c2',
      expect.objectContaining({ label: 'Monatsbeitrag' }),
      'team1',
    );
    expect(toastMsg).toHaveBeenCalledWith('Beitrag aktualisiert');
  });

  it('editContribGroup passes dueDate through when set', async () => {
    const { result } = renderActions();
    const rows = [{ id: 'c1' }] as Contribution[];
    await act(async () => {
      await result.current.editContribGroup(rows, {
        label: 'Monatsbeitrag',
        amount: '20',
        dueDate: '2026-06-30',
      } as ContribGroupEditFormValues);
    });
    expect(api.finances.updateContribution).toHaveBeenCalledWith(
      'c1',
      expect.objectContaining({ dueDate: '2026-06-30' }),
      'team1',
    );
  });

  it('editContribGroup reports a partial failure without clearing all failures', async () => {
    (api.finances.updateContribution as ReturnType<typeof vi.fn>).mockImplementationOnce(() =>
      Promise.reject(new Error('boom')),
    );
    const { result } = renderActions();
    const rows = [{ id: 'c1' }, { id: 'c2' }] as Contribution[];
    await act(async () => {
      await result.current.editContribGroup(rows, {
        label: 'Monatsbeitrag',
        amount: '20',
        dueDate: '',
      } as ContribGroupEditFormValues);
    });
    expect(toastMsg).toHaveBeenCalledWith('1 von 2 Beiträgen konnten nicht gespeichert werden', undefined, 'error');
  });

  it('saveContribCreate fans out to createContributions with the selected members', async () => {
    const { result } = renderActions();
    await act(async () => {
      await result.current.saveContribCreate({
        label: 'Mitgliedsbeitrag Juli',
        amount: '25',
        dueDate: '',
        userIds: ['u1', 'u2'],
      });
    });
    expect(api.finances.createContributions).toHaveBeenCalledWith('team1', {
      label: 'Mitgliedsbeitrag Juli',
      amount: 25,
      userIds: ['u1', 'u2'],
    });
    expect(toastMsg).toHaveBeenCalledWith('Beitrag angelegt');
  });

  it('openContribCreate opens the contribCreate sheet', () => {
    const { result } = renderActions();
    act(() => {
      result.current.openContribCreate();
    });
    expect(setState).toHaveBeenCalledWith(
      expect.objectContaining({
        sheet: expect.objectContaining({ type: 'contribCreate' }),
      }),
    );
  });

  it('setStatsRange updates state', () => {
    const { result } = renderActions();
    const range = { from: '2026-01-01', to: '2026-12-31' } as never;
    act(() => {
      result.current.setStatsRange(range);
    });
    expect(setState).toHaveBeenCalledWith({ statsRange: range });
  });

  it('openContribDetail sets contribDetail sheet', () => {
    const c = { id: 'c1', label: 'Beitrag', amount: 20 } as never;
    const { result } = renderActions();
    act(() => {
      result.current.openContribDetail(c);
    });
    expect(setState).toHaveBeenCalledWith(
      expect.objectContaining({
        sheet: expect.objectContaining({ type: 'contribDetail', formInitial: { id: 'c1' } }),
      }),
    );
  });

  it('openContribGroupEdit sets contribGroupEdit sheet prefilled from the first row', () => {
    const rows = [
      { id: 'c1', label: 'Monatsbeitrag', amount: 20, description: 'desc', dueDate: '2026-06-30' },
      { id: 'c2', label: 'Monatsbeitrag', amount: 20, description: 'desc', dueDate: '2026-06-30' },
    ] as Contribution[];
    const { result } = renderActions();
    act(() => {
      result.current.openContribGroupEdit(rows);
    });
    expect(setState).toHaveBeenCalledWith(
      expect.objectContaining({
        sheet: expect.objectContaining({
          type: 'contribGroupEdit',
          formInitial: { label: 'Monatsbeitrag', amount: '20', description: 'desc', dueDate: '2026-06-30' },
          contribGroupRows: rows,
        }),
      }),
    );
  });
});
