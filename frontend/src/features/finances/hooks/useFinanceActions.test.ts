import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act, waitFor } from '@testing-library/react';
import { useFinanceActions } from './useFinanceActions';
import { createQueryWrapper, createTestQueryClient } from '@/test/queryTestUtils';
import { queryKeys } from '@/query/keys';
import type { AppState } from '@/context/AppContext';
import type { TxFormValues } from '../components/txFormSchema';
import type { PenaltyFormValues } from '../components/penaltyFormSchema';
import type { PenaltyAssignFormValues } from '../components/penaltyAssignFormSchema';
import type { ContribFormValues } from '../components/contribFormSchema';
import type { FinanceOverview } from '../types';
import type { QueryClient } from '@tanstack/react-query';

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
      setPenaltyPaid: vi.fn().mockResolvedValue(undefined),
      setContributionPaid: vi.fn().mockResolvedValue(undefined),
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


  it('saveContrib updates contribution when valid', async () => {
    const { result } = renderActions();
    await act(async () => {
      await result.current.saveContrib({ label: 'Monatsbeitrag', amount: '20', id: 'c1' } as ContribFormValues);
    });
    expect(api.finances.updateContribution).toHaveBeenCalledWith(
      'c1',
      expect.objectContaining({ label: 'Monatsbeitrag' }),
      'team1',
    );
    expect(toastMsg).toHaveBeenCalledWith('Beitrag gespeichert');
  });

  it('setPenaltyPaid calls setPenaltyPaid with the desired value', async () => {
    const { result } = renderActions();
    await act(async () => {
      await result.current.setPenaltyPaid('a1', true);
    });
    expect(api.finances.setPenaltyPaid).toHaveBeenCalledWith('a1', 'team1', true);
  });

  it('setContributionPaid calls setContributionPaid with the desired value', async () => {
    const { result } = renderActions();
    await act(async () => {
      await result.current.setContributionPaid('c1', false);
    });
    expect(api.finances.setContributionPaid).toHaveBeenCalledWith('c1', 'team1', false);
  });

  it('setStatsRange updates state', () => {
    const { result } = renderActions();
    const range = { from: '2026-01-01', to: '2026-12-31' } as never;
    act(() => {
      result.current.setStatsRange(range);
    });
    expect(setState).toHaveBeenCalledWith({ statsRange: range });
  });

  it('openContribForm sets contribForm sheet', () => {
    const c = { id: 'c1', label: 'Beitrag', amount: 20 } as never;
    const { result } = renderActions();
    act(() => {
      result.current.openContribForm(c);
    });
    expect(setState).toHaveBeenCalledWith(
      expect.objectContaining({
        sheet: expect.objectContaining({ type: 'contribForm' }),
      }),
    );
  });
});
