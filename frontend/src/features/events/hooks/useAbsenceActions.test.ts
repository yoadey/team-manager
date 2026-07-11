import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useAbsenceActions } from './useAbsenceActions';
import type { AppState } from '@/context/AppContext';

function makeState(overrides: Partial<AppState> = {}): AppState {
  return {
    phase: 'app',
    user: { id: 'u1', name: 'Test User', email: 'test@test.com', avatarColor: '#000', photo: null },
    activeTeamId: 'team1',
    sheet: { type: 'absenceForm', mode: 'edit' } as never,
    form: { id: 'a1', from: '2026-01-01', to: '2026-01-02', reason: 'Urlaub' },
    formErrors: {},
    busy: null,
    toast: null,
    route: 'home',
    events: [],
    members: [],
    finances: null,
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
    absences: {
      update: vi.fn().mockResolvedValue(undefined),
      create: vi.fn().mockResolvedValue(undefined),
      remove: vi.fn().mockResolvedValue(undefined),
    },
  };
}

describe('useAbsenceActions', () => {
  let setState: ReturnType<typeof vi.fn>;
  let toastMsg: ReturnType<typeof vi.fn>;
  let refreshEvents: ReturnType<typeof vi.fn>;
  let loadAbsences: ReturnType<typeof vi.fn>;
  let askConfirm: ReturnType<typeof vi.fn>;
  let logout: ReturnType<typeof vi.fn>;
  let api: ReturnType<typeof makeApi>;
  let stateRef: AppState;

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
    refreshEvents = vi.fn().mockResolvedValue(undefined);
    loadAbsences = vi.fn().mockResolvedValue(undefined);
    askConfirm = vi.fn((cfg) => cfg.onConfirm());
    logout = vi.fn();
    api = makeApi();
  });

  function renderActions() {
    return renderHook(() =>
      useAbsenceActions({
        api: api as never,
        S: () => stateRef,
        setState: setState as never,
        refreshEvents: refreshEvents as never,
        loadAbsences: loadAbsences as never,
        askConfirm: askConfirm as never,
        toastMsg: toastMsg as never,
        logout: logout as never,
      }),
    );
  }

  it('saveAbsence updates an existing absence and shows toast', async () => {
    const { result } = renderActions();
    await act(async () => {
      await result.current.saveAbsence();
    });
    expect(api.absences.update).toHaveBeenCalledWith(
      'a1',
      { from: '2026-01-01', to: '2026-01-02', reason: 'Urlaub' },
      'team1',
    );
    expect(toastMsg).toHaveBeenCalled();
    expect(stateRef.sheet).toBeNull();
  });

  it('saveAbsence shows toast without calling the API on invalid date range', async () => {
    stateRef = makeState({ form: { id: 'a1', from: '2026-01-05', to: '2026-01-01', reason: 'x' } });
    const { result } = renderActions();
    await act(async () => {
      await result.current.saveAbsence();
    });
    expect(toastMsg).toHaveBeenCalled();
    expect(api.absences.update).not.toHaveBeenCalled();
  });

  // Regression: saveAbsence used to close the sheet unconditionally (once
  // the team still matched), so a slow save could clobber whatever
  // DIFFERENT sheet the user had since opened while it was in flight.
  it('saveAbsence does not touch the sheet if the user opened something else while in flight', async () => {
    let resolveUpdate!: () => void;
    api.absences.update = vi.fn(() => new Promise<void>((resolve) => (resolveUpdate = resolve)));
    const { result } = renderActions();

    let savePromise!: Promise<void>;
    act(() => {
      savePromise = result.current.saveAbsence();
    });

    const somethingElse = { type: 'teams' } as never;
    stateRef = { ...stateRef, sheet: somethingElse };

    await act(async () => {
      resolveUpdate();
      await savePromise;
    });

    expect(stateRef.sheet).toBe(somethingElse);
  });

  it('removeAbsence asks for confirmation, then removes and shows toast', async () => {
    const { result } = renderActions();
    await act(async () => {
      result.current.removeAbsence('a1');
      await Promise.resolve();
    });
    expect(askConfirm).toHaveBeenCalledWith(expect.objectContaining({ danger: true }));
    expect(api.absences.remove).toHaveBeenCalledWith('a1', 'team1');
    expect(toastMsg).toHaveBeenCalled();
  });

  it('removeAbsence reports an error without removing on API failure', async () => {
    api.absences.remove.mockRejectedValueOnce(new Error('boom'));
    const { result } = renderActions();
    await act(async () => {
      result.current.removeAbsence('a1');
      await Promise.resolve();
    });
    expect(toastMsg).toHaveBeenCalled();
    expect(refreshEvents).not.toHaveBeenCalled();
  });
});
