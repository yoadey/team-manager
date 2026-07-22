import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act, waitFor } from '@testing-library/react';
import { useAbsenceActions } from './useAbsenceActions';
import { createQueryWrapper } from '@/test/queryTestUtils';
import type { AppState } from '@/context/AppContext';
import type { AbsenceFormValues } from '../components/absenceFormSchema';

function makeState(overrides: Partial<AppState> = {}): AppState {
  return {
    phase: 'app',
    user: { id: 'u1', name: 'Test User', email: 'test@test.com', avatarColor: '#000', photo: null },
    activeTeamId: 'team1',
    sheet: {
      type: 'absenceForm',
      mode: 'edit',
      formInitial: { id: 'a1', from: '2026-01-01', to: '2026-01-02', reason: 'Urlaub' },
    } as never,
    busy: null,
    toast: null,
    route: 'home',
    finances: null,
    stats: null,
    statsRange: null,
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
      create: vi.fn().mockResolvedValue({ id: 'a1' }),
      remove: vi.fn().mockResolvedValue(undefined),
    },
  };
}

describe('useAbsenceActions', () => {
  let setState: ReturnType<typeof vi.fn>;
  let toastMsg: ReturnType<typeof vi.fn>;
  let loadNotifications: ReturnType<typeof vi.fn>;
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
    loadNotifications = vi.fn().mockResolvedValue(undefined);
    askConfirm = vi.fn((cfg) => cfg.onConfirm());
    logout = vi.fn();
    api = makeApi();
  });

  function renderActions() {
    return renderHook(
      () =>
        useAbsenceActions({
          api: api as never,
          S: () => stateRef,
          setState: setState as never,
          teamId: stateRef.activeTeamId,
          loadNotifications: loadNotifications as never,
          askConfirm: askConfirm as never,
          toastMsg: toastMsg as never,
          logout: logout as never,
        }),
      { wrapper: createQueryWrapper() },
    );
  }

  // Regression test: openAbsenceForm used to prefill a NEW absence's reason
  // field with the literal German word 'Urlaub' as an actual form VALUE
  // (not a placeholder), independent of the active UI locale -- an
  // English-locale user opening "Log absence" saw an already-filled German
  // word instead of the already-translated absenceReasonPlaceholder hint
  // (which never renders once the field has a value), and could save an
  // absence with that untranslated reason shown to every teammate.
  it('openAbsenceForm defaults a new absence reason to empty (not a hardcoded locale-specific value)', () => {
    const { result } = renderActions();
    act(() => {
      result.current.openAbsenceForm();
    });
    expect(stateRef.sheet!.formInitial).toMatchObject({ reason: '' });
  });

  it('openAbsenceForm preserves the existing reason when editing an absence', () => {
    const { result } = renderActions();
    act(() => {
      result.current.openAbsenceForm({ id: 'a1', from: '2026-01-01', to: '2026-01-02', reason: 'Injured knee' });
    });
    expect(stateRef.sheet!.formInitial).toMatchObject({ reason: 'Injured knee' });
  });

  it('saveAbsence updates an existing absence and shows toast', async () => {
    const { result } = renderActions();
    await act(async () => {
      await result.current.saveAbsence(stateRef.sheet!.formInitial as AbsenceFormValues);
    });
    expect(api.absences.update).toHaveBeenCalledWith(
      'a1',
      { from: '2026-01-01', to: '2026-01-02', reason: 'Urlaub' },
      'team1',
    );
    expect(toastMsg).toHaveBeenCalled();
    expect(loadNotifications).toHaveBeenCalled();
    expect(stateRef.sheet).toBeNull();
  });

  it('saveAbsence creates a new absence in create mode', async () => {
    stateRef = makeState({
      sheet: {
        type: 'absenceForm',
        mode: 'create',
        formInitial: { from: '2026-01-10', to: '2026-01-15', reason: 'Ski trip' },
      } as never,
    });
    const { result } = renderActions();
    await act(async () => {
      await result.current.saveAbsence(stateRef.sheet!.formInitial as AbsenceFormValues);
    });
    expect(api.absences.create).toHaveBeenCalledWith(
      expect.objectContaining({ teamId: 'team1', userId: 'u1', from: '2026-01-10', to: '2026-01-15' }),
    );
    expect(toastMsg).toHaveBeenCalled();
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
      savePromise = result.current.saveAbsence(stateRef.sheet!.formInitial as AbsenceFormValues);
    });
    await waitFor(() => expect(api.absences.update).toHaveBeenCalled());

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
    expect(loadNotifications).toHaveBeenCalled();
  });

  it('removeAbsence reports an error without removing on API failure', async () => {
    api.absences.remove.mockRejectedValueOnce(new Error('boom'));
    const { result } = renderActions();
    await act(async () => {
      result.current.removeAbsence('a1');
      await Promise.resolve();
    });
    expect(toastMsg).toHaveBeenCalled();
  });

  // Regression test: mirrors useDeleteEventMutation/useRemoveMemberMutation's
  // per-call teamId safeguard. The confirm sheet can still be open (and get
  // confirmed) after the user has switched to a different active team; the
  // delete must still target the team the confirm dialog was opened for.
  it('removeAbsence onConfirm deletes against the team the confirm dialog was opened for, even after a team switch', async () => {
    askConfirm = vi.fn();
    const { result, rerender } = renderActions();
    act(() => {
      result.current.removeAbsence('a1');
    });
    const cfg = askConfirm.mock.calls[0]![0];

    stateRef = { ...stateRef, activeTeamId: 'team2' };
    rerender();

    await act(async () => {
      await cfg.onConfirm();
    });
    expect(api.absences.remove).toHaveBeenCalledWith('a1', 'team1');
  });
});
