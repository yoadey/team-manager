import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act, waitFor } from '@testing-library/react';
import { useFinanceActions } from './useFinanceActions';
import { createQueryWrapper, createTestQueryClient } from '@/test/queryTestUtils';
import { queryKeys } from '@/query/keys';
import type { AppState } from '@/context/AppContext';
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
    form: {},
    formErrors: {},
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
      togglePenaltyPaid: vi.fn().mockResolvedValue(undefined),
      toggleContribution: vi.fn().mockResolvedValue(undefined),
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
    expect(setState).toHaveBeenCalledWith(expect.objectContaining({ form: expect.objectContaining({ category: '' }) }));
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

  it('saveTx shows toast when title is empty', async () => {
    stateRef = makeState({ form: { title: '', amount: '10', type: 'income', category: 'Test' } });
    const { result } = renderActions();
    await act(async () => {
      await result.current.saveTx();
    });
    expect(toastMsg).toHaveBeenCalledWith(expect.stringContaining('fehlt'), undefined, 'error');
    expect(api.finances.addTransaction).not.toHaveBeenCalled();
  });

  it('saveTx shows toast when amount is invalid', async () => {
    stateRef = makeState({ form: { title: 'Test', amount: 'abc', type: 'income', category: 'Test' } });
    const { result } = renderActions();
    await act(async () => {
      await result.current.saveTx();
    });
    expect(toastMsg).toHaveBeenCalled();
    expect(api.finances.addTransaction).not.toHaveBeenCalled();
  });

  it('saveTx creates transaction in create mode', async () => {
    stateRef = makeState({
      sheet: { type: 'txForm', mode: 'create' } as never,
      form: { title: 'Beitrag Jan', amount: '50', type: 'income', category: 'Beiträge' },
    });
    const { result } = renderActions();
    await act(async () => {
      await result.current.saveTx();
    });
    expect(api.finances.addTransaction).toHaveBeenCalled();
    expect(toastMsg).toHaveBeenCalledWith('Buchung gespeichert');
  });

  it('saveTx updates transaction in edit mode', async () => {
    stateRef = makeState({
      sheet: { type: 'txForm', mode: 'edit' } as never,
      form: { id: 'tx1', title: 'Updated', amount: '75', type: 'expense', category: 'Ausrüstung' },
    });
    const { result } = renderActions();
    await act(async () => {
      await result.current.saveTx();
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
    stateRef = makeState({
      sheet: { type: 'txForm', mode: 'edit' } as never,
      form: { id: 'tx1', title: 'Updated', amount: '75', type: 'expense', category: 'Ausrüstung' },
    });
    const { result } = renderActions();

    let savePromise!: Promise<void>;
    act(() => {
      savePromise = result.current.saveTx();
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

  // Regression test: openPenaltyForm/openPenaltyAssign/openContribForm used
  // to leave a prior sheet's formErrors in place, unlike openTxForm/
  // openEventForm/openNewsForm/openPollForm, which all reset it -- reopening
  // any of these three after a validation error on a *different* record
  // showed that stale error under a freshly-loaded, valid form.
  it('openPenaltyForm clears a stale formErrors from a previous sheet', () => {
    stateRef = makeState({ formErrors: { label: 'Bezeichnung fehlt.' } });
    const { result } = renderActions();
    act(() => {
      result.current.openPenaltyForm();
    });
    expect(stateRef.formErrors).toEqual({});
  });

  it('savePenalty shows toast when label is empty', async () => {
    stateRef = makeState({
      form: { label: '', amount: '10' },
      sheet: { type: 'penaltyForm', mode: 'create' } as never,
    });
    const { result } = renderActions();
    await act(async () => {
      await result.current.savePenalty();
    });
    expect(toastMsg).toHaveBeenCalled();
    expect(api.finances.createPenalty).not.toHaveBeenCalled();
  });

  it('savePenalty creates penalty in create mode', async () => {
    stateRef = makeState({
      sheet: { type: 'penaltyForm', mode: 'create', back: null } as never,
      form: { label: 'Zu spät', amount: '5' },
    });
    const { result } = renderActions();
    await act(async () => {
      await result.current.savePenalty();
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

  it('savePenaltyAssign shows toast when userId is missing', async () => {
    stateRef = makeState({ form: { userId: '', penaltyId: 'p1' } });
    const { result } = renderActions();
    await act(async () => {
      await result.current.savePenaltyAssign();
    });
    expect(toastMsg).toHaveBeenCalledWith('Bitte Person wählen.', undefined, 'error');
  });

  it('savePenaltyAssign shows toast when penaltyId is missing', async () => {
    stateRef = makeState({ form: { userId: 'u1', penaltyId: '' } });
    const { result } = renderActions();
    await act(async () => {
      await result.current.savePenaltyAssign();
    });
    expect(toastMsg).toHaveBeenCalledWith('Bitte Strafe wählen.', undefined, 'error');
  });

  it('savePenaltyAssign assigns penalty when valid', async () => {
    stateRef = makeState({ form: { userId: 'u1', penaltyId: 'p1' } });
    const { result } = renderActions();
    await act(async () => {
      await result.current.savePenaltyAssign();
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
    const onConfirm = askConfirm.mock.calls[0][0].onConfirm;
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
    const onConfirm = askConfirm.mock.calls[0][0].onConfirm;

    stateRef = { ...stateRef, activeTeamId: 'team2' };
    rerender();

    await act(async () => {
      await onConfirm();
    });
    expect(api.finances.deleteAssignment).toHaveBeenCalledWith('a1', 'team1');
  });

  it('saveContrib validates label', async () => {
    stateRef = makeState({ form: { label: '', amount: '10', id: 'c1' } });
    const { result } = renderActions();
    await act(async () => {
      await result.current.saveContrib();
    });
    expect(toastMsg).toHaveBeenCalled();
    expect(api.finances.updateContribution).not.toHaveBeenCalled();
  });

  it('saveContrib updates contribution when valid', async () => {
    stateRef = makeState({ form: { label: 'Monatsbeitrag', amount: '20', id: 'c1' } });
    const { result } = renderActions();
    await act(async () => {
      await result.current.saveContrib();
    });
    expect(api.finances.updateContribution).toHaveBeenCalledWith(
      'c1',
      expect.objectContaining({ label: 'Monatsbeitrag' }),
      'team1',
    );
    expect(toastMsg).toHaveBeenCalledWith('Beitrag gespeichert');
  });

  it('togglePenalty calls togglePenaltyPaid', async () => {
    const { result } = renderActions();
    await act(async () => {
      await result.current.togglePenalty('a1');
    });
    expect(api.finances.togglePenaltyPaid).toHaveBeenCalledWith('a1', 'team1');
  });

  it('toggleContribution calls toggleContribution API', async () => {
    const { result } = renderActions();
    await act(async () => {
      await result.current.toggleContribution('c1');
    });
    expect(api.finances.toggleContribution).toHaveBeenCalledWith('c1', 'team1');
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
        sheet: { type: 'contribForm' },
      }),
    );
  });

  it('openContribForm clears a stale formErrors from a previous sheet', () => {
    stateRef = makeState({ formErrors: { label: 'Bezeichnung fehlt.' } });
    const c = { id: 'c1', label: 'Beitrag', amount: 20 } as never;
    const { result } = renderActions();
    act(() => {
      result.current.openContribForm(c);
    });
    expect(stateRef.formErrors).toEqual({});
  });

  it('openPenaltyAssign clears a stale formErrors from a previous sheet', () => {
    stateRef = makeState({ formErrors: { userId: 'Person erforderlich.' } });
    const { result } = renderActions();
    act(() => {
      result.current.openPenaltyAssign();
    });
    expect(stateRef.formErrors).toEqual({});
  });
});
