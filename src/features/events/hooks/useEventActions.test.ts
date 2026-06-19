import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useEventDetailActions, useEventActionFeatures } from './useEventActions';
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

function makeApi() {
  return {
    events: {
      get: vi.fn().mockResolvedValue({ id: 'ev1', title: 'Test Event', date: '2026-01-01' }),
      listComments: vi.fn().mockResolvedValue([]),
      addComment: vi.fn().mockResolvedValue(undefined),
      removeComment: vi.fn().mockResolvedValue(undefined),
      remove: vi.fn().mockResolvedValue(undefined),
      setStatus: vi.fn().mockResolvedValue(undefined),
    },
    attendance: {
      listForEvent: vi.fn().mockResolvedValue([]),
      set: vi.fn().mockResolvedValue(undefined),
      setNomination: vi.fn().mockResolvedValue(undefined),
    },
  };
}

describe('useEventDetailActions', () => {
  let setState: ReturnType<typeof vi.fn>;
  let toastMsg: ReturnType<typeof vi.fn>;
  let refreshEvents: ReturnType<typeof vi.fn>;
  let setFormVal: ReturnType<typeof vi.fn>;
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
    setFormVal = vi.fn();
    api = makeApi();
  });

  function renderActions() {
    return renderHook(() =>
      useEventDetailActions({
        api: api as never,
        S: () => stateRef,
        setState: setState as never,
        activeTeam: () => ({ id: 'team1', reasonVisibilityRoles: [] }) as never,
        myRoles: () => [],
        refreshEvents: refreshEvents as never,
        setFormVal: setFormVal as never,
        toastMsg: toastMsg as never,
      }),
    );
  }

  it('openEventDetail sets sheet and loads event data', async () => {
    const { result } = renderActions();
    await act(async () => {
      await result.current.openEventDetail('ev1');
    });
    expect(api.events.get).toHaveBeenCalledWith('ev1');
    expect(api.attendance.listForEvent).toHaveBeenCalledWith('ev1');
    expect(api.events.listComments).toHaveBeenCalledWith('ev1');
    expect(setState).toHaveBeenCalledWith(
      expect.objectContaining({ sheet: expect.objectContaining({ type: 'eventDetail' }) }),
    );
  });

  it('setMyStatus calls attendance API and shows toast', async () => {
    stateRef = makeState({ sheet: { type: 'eventDetail', eventId: 'ev1', event: null, rows: [] } as never });
    const { result } = renderActions();
    await act(async () => {
      await result.current.setMyStatus('ev1', 'yes');
    });
    expect(api.attendance.set).toHaveBeenCalled();
    expect(refreshEvents).toHaveBeenCalled();
    expect(toastMsg).toHaveBeenCalledWith('Zugesagt');
  });

  it('setMyStatus shows "Abgesagt" toast for no status', async () => {
    const { result } = renderActions();
    await act(async () => {
      await result.current.setMyStatus('ev1', 'no');
    });
    expect(toastMsg).toHaveBeenCalledWith('Abgesagt');
  });

  it('openComment sets comment sheet', () => {
    const { result } = renderActions();
    act(() => {
      result.current.openComment({ id: 'ev1', title: 'Test' } as never, {
        userId: 'u2',
        name: 'Bob',
        status: 'yes',
        reason: 'injured',
      });
    });
    expect(setState).toHaveBeenCalledWith(expect.any(Function));
    const call = setState.mock.calls[0][0];
    const patch = typeof call === 'function' ? call(stateRef) : call;
    expect(patch.sheet).toMatchObject({ type: 'comment', userId: 'u2' });
    expect(patch.form).toMatchObject({ commentText: 'injured' });
  });

  it('postEventComment calls addComment and reloads detail', async () => {
    stateRef = makeState({
      form: { newEventComment: 'Great match!' },
      sheet: { type: 'eventDetail', eventId: 'ev1', event: null, rows: [] } as never,
    });
    const { result } = renderActions();
    await act(async () => {
      await result.current.postEventComment('ev1');
    });
    expect(api.events.addComment).toHaveBeenCalledWith('ev1', 'Great match!');
    expect(setFormVal).toHaveBeenCalledWith({ newEventComment: '' });
  });

  it('postEventComment does nothing when text is empty', async () => {
    stateRef = makeState({ form: { newEventComment: '  ' } });
    const { result } = renderActions();
    await act(async () => {
      await result.current.postEventComment('ev1');
    });
    expect(api.events.addComment).not.toHaveBeenCalled();
  });

  it('removeEventComment calls removeComment API', async () => {
    const { result } = renderActions();
    await act(async () => {
      await result.current.removeEventComment('ev1', 'c1');
    });
    expect(api.events.removeComment).toHaveBeenCalledWith('c1');
  });

  it('toggleNomination calls setNomination and shows toast', async () => {
    const { result } = renderActions();
    await act(async () => {
      await result.current.toggleNomination('ev1', 'u2', false);
    });
    expect(api.attendance.setNomination).toHaveBeenCalledWith('ev1', 'u2', true);
    expect(toastMsg).toHaveBeenCalledWith('Nominiert');
  });

  it('toggleNomination shows "Nicht nominiert" toast when removing', async () => {
    const { result } = renderActions();
    await act(async () => {
      await result.current.toggleNomination('ev1', 'u2', true);
    });
    expect(toastMsg).toHaveBeenCalledWith('Nicht nominiert');
  });

  it('reloadDetail handles API errors gracefully', async () => {
    api.events.get = vi.fn().mockRejectedValue(new Error('Network error'));
    const { result } = renderActions();
    await act(async () => {
      await result.current.reloadDetail('ev1');
    });
    expect(toastMsg).toHaveBeenCalled();
  });
});

describe('useEventActionFeatures', () => {
  let setState: ReturnType<typeof vi.fn>;
  let toastMsg: ReturnType<typeof vi.fn>;
  let refreshEvents: ReturnType<typeof vi.fn>;
  let askConfirm: ReturnType<typeof vi.fn>;
  let openEventDetail: ReturnType<typeof vi.fn>;
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
    askConfirm = vi.fn();
    openEventDetail = vi.fn().mockResolvedValue(undefined);
    api = makeApi();
  });

  function renderActions() {
    return renderHook(() =>
      useEventActionFeatures({
        api: api as never,
        S: () => stateRef,
        setState: setState as never,
        activeTeam: () => null,
        myRoles: () => [],
        refreshEvents: refreshEvents as never,
        setFormVal: vi.fn() as never,
        toastMsg: toastMsg as never,
        askConfirm: askConfirm as never,
        openEventDetail: openEventDetail as never,
      }),
    );
  }

  it('askEventAction opens seriesAction sheet for series events', () => {
    const event = { id: 'ev1', title: 'Test', seriesId: 's1' } as never;
    const { result } = renderActions();
    act(() => {
      result.current.askEventAction('cancel', event);
    });
    expect(setState).toHaveBeenCalled();
    const patch = setState.mock.calls[0][0];
    const resolved = typeof patch === 'function' ? patch(stateRef) : patch;
    expect(resolved.sheet).toMatchObject({ type: 'seriesAction', action: 'cancel' });
  });

  it('askEventAction runs directly for non-series events (cancel)', async () => {
    const event = { id: 'ev1', title: 'Test', seriesId: null } as never;
    const { result } = renderActions();
    await act(async () => {
      await result.current.askEventAction('cancel', event);
    });
    expect(api.events.setStatus).toHaveBeenCalledWith('ev1', 'cancelled', 'single');
    expect(toastMsg).toHaveBeenCalledWith('Termin abgesagt');
  });

  it('runEventAction with delete calls askConfirm', async () => {
    const event = { id: 'ev1', title: 'Test', seriesId: null } as never;
    const { result } = renderActions();
    await act(async () => {
      await result.current.runEventAction('delete', event, 'single');
    });
    expect(askConfirm).toHaveBeenCalledWith(expect.objectContaining({ danger: true }));
  });

  it('runEventAction with reactivate shows correct toast', async () => {
    const event = { id: 'ev1', title: 'Test', seriesId: null } as never;
    const { result } = renderActions();
    await act(async () => {
      await result.current.runEventAction('reactivate', event, 'single');
    });
    expect(api.events.setStatus).toHaveBeenCalledWith('ev1', 'active', 'single');
    expect(toastMsg).toHaveBeenCalledWith('Termin aktiviert');
  });
});
