import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useAbsenceActions } from './useAbsenceActions';
import { useCalExportActions } from './useCalExportActions';
import type { AppState } from '@/context/AppContext';

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

describe('useAbsenceActions', () => {
  let setState: ReturnType<typeof vi.fn>;
  let toastMsg: ReturnType<typeof vi.fn>;
  let refreshEvents: ReturnType<typeof vi.fn>;
  let loadAbsences: ReturnType<typeof vi.fn>;
  let askConfirm: ReturnType<typeof vi.fn>;
  let logout: ReturnType<typeof vi.fn>;
  let api: {
    absences: { create: ReturnType<typeof vi.fn>; update: ReturnType<typeof vi.fn>; remove: ReturnType<typeof vi.fn> };
  };
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
    askConfirm = vi.fn();
    logout = vi.fn();
    api = {
      absences: {
        create: vi.fn().mockResolvedValue({ id: 'ab1' }),
        update: vi.fn().mockResolvedValue(undefined),
        remove: vi.fn().mockResolvedValue(undefined),
      },
    };
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

  it('openAbsenceForm sets create sheet with today as default dates', () => {
    const { result } = renderActions();
    act(() => {
      result.current.openAbsenceForm();
    });
    expect(setState).toHaveBeenCalledWith(
      expect.objectContaining({
        sheet: expect.objectContaining({ type: 'absenceForm', mode: 'create' }),
      }),
    );
  });

  it('openAbsenceForm sets edit sheet when absence passed', () => {
    const absence = { id: 'ab1', from: '2026-01-01', to: '2026-01-07', reason: 'Urlaub' };
    const { result } = renderActions();
    act(() => {
      result.current.openAbsenceForm(absence);
    });
    expect(setState).toHaveBeenCalledWith(
      expect.objectContaining({
        sheet: expect.objectContaining({ type: 'absenceForm', mode: 'edit' }),
        form: expect.objectContaining({ id: 'ab1', reason: 'Urlaub' }),
      }),
    );
  });

  it('saveAbsence shows toast when date range is invalid', async () => {
    stateRef = makeState({
      form: { from: '2026-02-01', to: '2026-01-01', reason: 'Test' },
      sheet: { type: 'absenceForm', mode: 'create' } as never,
    });
    const { result } = renderActions();
    await act(async () => {
      await result.current.saveAbsence();
    });
    expect(toastMsg).toHaveBeenCalled();
    expect(api.absences.create).not.toHaveBeenCalled();
  });

  it('saveAbsence creates absence in create mode', async () => {
    stateRef = makeState({
      form: { from: '2026-01-10', to: '2026-01-15', reason: 'Urlaub' },
      sheet: { type: 'absenceForm', mode: 'create' } as never,
    });
    const { result } = renderActions();
    await act(async () => {
      await result.current.saveAbsence();
    });
    expect(api.absences.create).toHaveBeenCalledWith(
      expect.objectContaining({ from: '2026-01-10', to: '2026-01-15', userId: 'u1' }),
    );
    expect(toastMsg).toHaveBeenCalledWith('Abwesenheit eingetragen');
  });

  it('saveAbsence updates absence in edit mode', async () => {
    stateRef = makeState({
      form: { id: 'ab1', from: '2026-01-10', to: '2026-01-20', reason: 'Krank' },
      sheet: { type: 'absenceForm', mode: 'edit' } as never,
    });
    const { result } = renderActions();
    await act(async () => {
      await result.current.saveAbsence();
    });
    expect(api.absences.update).toHaveBeenCalledWith('ab1', expect.objectContaining({ reason: 'Krank' }), 'team1');
    expect(toastMsg).toHaveBeenCalledWith('Abwesenheit aktualisiert');
  });

  it('removeAbsence calls askConfirm', () => {
    const { result } = renderActions();
    act(() => {
      result.current.removeAbsence('ab1');
    });
    expect(askConfirm).toHaveBeenCalledWith(
      expect.objectContaining({
        title: 'Abwesenheit löschen?',
        danger: true,
      }),
    );
  });

  it('removeAbsence onConfirm removes absence and shows toast', async () => {
    const { result } = renderActions();
    act(() => {
      result.current.removeAbsence('ab1');
    });
    const cfg = askConfirm.mock.calls[0][0];
    await act(async () => {
      await cfg.onConfirm();
    });
    expect(api.absences.remove).toHaveBeenCalledWith('ab1', 'team1');
    expect(toastMsg).toHaveBeenCalledWith('Abwesenheit entfernt');
  });
});

describe('useCalExportActions', () => {
  let setState: ReturnType<typeof vi.fn>;
  let toastMsg: ReturnType<typeof vi.fn>;
  let stateRef: AppState;

  beforeEach(() => {
    stateRef = makeState({
      events: [
        {
          id: 'ev1',
          title: 'Training',
          date: '2026-03-01',
          type: 'training',
          status: 'active',
          startTime: null,
          endTime: null,
          meetTime: null,
          location: 'Halle',
          note: 'Bring boots',
        },
        {
          id: 'ev2',
          title: 'Cancelled',
          date: '2026-03-05',
          type: 'event',
          status: 'cancelled',
          startTime: null,
          endTime: null,
          meetTime: null,
          location: null,
          note: null,
        },
      ] as never,
    });
    setState = vi.fn((patch) => {
      if (typeof patch === 'function') {
        const result = patch(stateRef);
        stateRef = { ...stateRef, ...result };
      } else {
        stateRef = { ...stateRef, ...patch };
      }
    });
    toastMsg = vi.fn();
  });

  function renderActions() {
    return renderHook(() =>
      useCalExportActions({
        S: () => stateRef,
        setState: setState as never,
        activeTeam: () => ({ id: 'team1', name: 'Test Team', short: 'TT' }) as never,
        toastMsg: toastMsg as never,
      }),
    );
  }

  it('openCalExport sets calExport sheet', () => {
    const { result } = renderActions();
    act(() => {
      result.current.openCalExport();
    });
    expect(setState).toHaveBeenCalledWith({ sheet: { type: 'calExport' } });
  });

  it('downloadIcs filters cancelled events and shows toast', () => {
    URL.createObjectURL = vi.fn().mockReturnValue('blob:test');
    URL.revokeObjectURL = vi.fn();
    const { result } = renderActions();
    act(() => {
      result.current.downloadIcs();
    });
    expect(toastMsg).toHaveBeenCalledWith('1 Termine als .ics exportiert');
  });

  it('copyCalUrl sets copied and shows toast', async () => {
    Object.assign(navigator, { clipboard: { writeText: vi.fn().mockResolvedValue(undefined) } });
    stateRef = makeState({ sheet: { type: 'calExport' } as never });
    const { result } = renderActions();
    await act(async () => {
      await result.current.copyCalUrl();
    });
    expect(toastMsg).toHaveBeenCalledWith('Abo-Link kopiert');
  });

  it('copyCalUrl shows an error toast when the clipboard write fails', async () => {
    Object.assign(navigator, { clipboard: { writeText: vi.fn().mockRejectedValue(new Error('denied')) } });
    stateRef = makeState({ sheet: { type: 'calExport' } as never });
    const { result } = renderActions();
    await act(async () => {
      await result.current.copyCalUrl();
    });
    expect(toastMsg).toHaveBeenCalledWith('Kopieren fehlgeschlagen');
  });

  // Regression test: the sheet update used to check only sheet.type ===
  // 'calExport', never the team. If the user switched teams and reopened
  // the calExport sheet (also type 'calExport') for the new team before a
  // slow clipboard write for the old team resolved, the stale resolution
  // would show "Copied!" on the new team's sheet even though nothing was
  // copied for it.
  it('does not mark a different team\'s calExport sheet as copied after a slow clipboard write resolves', async () => {
    let resolveWrite!: () => void;
    Object.assign(navigator, {
      clipboard: { writeText: vi.fn(() => new Promise<void>((resolve) => (resolveWrite = resolve))) },
    });
    stateRef = makeState({ sheet: { type: 'calExport' } as never });
    const { result } = renderActions();

    let copyPromise!: Promise<void>;
    act(() => {
      copyPromise = result.current.copyCalUrl();
    });

    // User switches teams and opens THAT team's own (also empty) calExport sheet.
    stateRef = { ...stateRef, activeTeamId: 'team2', sheet: { type: 'calExport' } as never };

    await act(async () => {
      resolveWrite();
      await copyPromise;
    });

    expect(stateRef.sheet).toEqual({ type: 'calExport' });
  });
});
