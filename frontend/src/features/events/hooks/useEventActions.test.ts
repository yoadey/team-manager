import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useEventDetailActions, useEventActionFeatures } from './useEventActions';
import { createQueryWrapper } from '@/test/queryTestUtils';
import type { AppState } from '@/context/AppContext';

function makeState(overrides: Partial<AppState> = {}): AppState {
  return {
    phase: 'app',
    user: { id: 'u1', name: 'Test User', email: 'test@test.com', avatarColor: '#000', photo: null },
    activeTeamId: 'team1',
    sheet: null,
    busy: null,
    toast: null,
    route: 'home',
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
      addComment: vi.fn().mockResolvedValue({ id: 'c1', eventId: 'ev1', userId: 'u1', text: 'hi', createdAt: '' }),
      removeComment: vi.fn().mockResolvedValue(undefined),
      remove: vi.fn().mockResolvedValue(undefined),
      setStatus: vi.fn().mockResolvedValue({ id: 'ev1' }),
    },
    attendance: {
      listForEvent: vi.fn().mockResolvedValue([]),
      set: vi.fn().mockResolvedValue({ id: 'a1', eventId: 'ev1', userId: 'u1', status: 'yes', reason: '' }),
      setNomination: vi.fn().mockResolvedValue(true),
    },
  };
}

describe('useEventDetailActions', () => {
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
    askConfirm = vi.fn();
    logout = vi.fn();
    api = makeApi();
  });

  function renderActions() {
    return renderHook(
      () =>
        useEventDetailActions({
          api: api as never,
          S: () => stateRef,
          setState: setState as never,
          activeTeam: () => ({ id: 'team1', reasonVisibilityRoles: [] }) as never,
          myRoles: () => [],
          teamId: stateRef.activeTeamId,
          loadNotifications: loadNotifications as never,
          askConfirm: askConfirm as never,
          toastMsg: toastMsg as never,
          logout: logout as never,
        }),
      { wrapper: createQueryWrapper() },
    );
  }

  it('openEventDetail sets the sheet to the given eventId', () => {
    const { result } = renderActions();
    act(() => {
      result.current.openEventDetail('ev1');
    });
    expect(setState).toHaveBeenCalledWith({ sheet: { type: 'eventDetail', eventId: 'ev1' } });
  });

  it('setMyStatus calls attendance API and shows toast', async () => {
    stateRef = makeState({ sheet: { type: 'eventDetail', eventId: 'ev1' } as never });
    const { result } = renderActions();
    await act(async () => {
      await result.current.setMyStatus('ev1', 'yes');
    });
    expect(api.attendance.set).toHaveBeenCalledWith('ev1', 'u1', { status: 'yes', reason: '' }, 'team1');
    expect(loadNotifications).toHaveBeenCalled();
    expect(toastMsg).toHaveBeenCalledWith('Zugesagt');
  });

  it('setMyStatus passes the current reason through when kept', async () => {
    const { result } = renderActions();
    await act(async () => {
      await result.current.setMyStatus('ev1', 'yes', 'injured');
    });
    expect(api.attendance.set).toHaveBeenCalledWith('ev1', 'u1', { status: 'yes', reason: 'injured' }, 'team1');
  });

  it('setMyStatus shows "Abgesagt" toast for no status', async () => {
    const { result } = renderActions();
    await act(async () => {
      await result.current.setMyStatus('ev1', 'no');
    });
    expect(toastMsg).toHaveBeenCalledWith('Abgesagt');
  });

  // Regression test: a rapid double-tap on the RSVP buttons (or a user
  // switching Yes -> No before the first request resolves) used to fire two
  // concurrent api.attendance.set calls with no guard, unlike the sibling
  // setStatusFor (roster admin view), risking an out-of-order response
  // overwriting the UI with a stale status.
  it('setMyStatus ignores a second call while the first is still in flight', async () => {
    let resolveFirst!: (v: unknown) => void;
    api.attendance.set = vi
      .fn()
      .mockImplementationOnce(() => new Promise((resolve) => (resolveFirst = resolve)))
      .mockResolvedValue({});
    const { result } = renderActions();

    let firstDone = false;
    const first = act(async () => {
      await result.current.setMyStatus('ev1', 'yes').then(() => (firstDone = true));
    });
    await act(async () => {
      await result.current.setMyStatus('ev1', 'no');
    });

    expect(firstDone).toBe(false);
    expect(api.attendance.set).toHaveBeenCalledTimes(1);

    resolveFirst({});
    await first;
    expect(firstDone).toBe(true);
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
    const call = setState.mock.calls[0]![0];
    const patch = typeof call === 'function' ? call(stateRef) : call;
    expect(patch.sheet).toMatchObject({ type: 'comment', userId: 'u2', formInitial: 'injured' });
  });

  it('postEventComment calls addComment and returns true', async () => {
    stateRef = makeState({
      sheet: { type: 'eventDetail', eventId: 'ev1' } as never,
    });
    const { result } = renderActions();
    let ok = false;
    await act(async () => {
      ok = await result.current.postEventComment('ev1', 'Great match!');
    });
    expect(api.events.addComment).toHaveBeenCalledWith('ev1', 'Great match!', 'team1');
    expect(ok).toBe(true);
  });

  it('postEventComment does nothing when text is empty', async () => {
    const { result } = renderActions();
    await act(async () => {
      await result.current.postEventComment('ev1', '  ');
    });
    expect(api.events.addComment).not.toHaveBeenCalled();
  });

  // Regression test: removeEventComment used to call the API directly with
  // no confirmation, unlike the app's otherwise-universal confirm-before-
  // destroy convention (deleteEvent, removeMember, removeAbsence, etc.).
  it('removeEventComment asks for confirmation before calling the API', () => {
    const { result } = renderActions();
    act(() => {
      result.current.removeEventComment('ev1', 'c1');
    });
    expect(askConfirm).toHaveBeenCalledWith(expect.objectContaining({ danger: true }));
    expect(api.events.removeComment).not.toHaveBeenCalled();
  });

  it('removeEventComment calls the API once confirmed', async () => {
    const { result } = renderActions();
    act(() => {
      result.current.removeEventComment('ev1', 'c1');
    });
    const onConfirm = askConfirm.mock.calls[0]![0].onConfirm;
    await act(async () => {
      await onConfirm();
    });
    expect(api.events.removeComment).toHaveBeenCalledWith('c1', 'ev1', 'team1');
  });

  it('toggleNomination calls setNomination and shows toast', async () => {
    const { result } = renderActions();
    await act(async () => {
      await result.current.toggleNomination('ev1', 'u2', false);
    });
    expect(api.attendance.setNomination).toHaveBeenCalledWith('ev1', 'u2', true, 'team1');
    expect(loadNotifications).toHaveBeenCalled();
    expect(toastMsg).toHaveBeenCalledWith('Nominiert');
  });

  it('toggleNomination shows "Nicht nominiert" toast when removing', async () => {
    const { result } = renderActions();
    await act(async () => {
      await result.current.toggleNomination('ev1', 'u2', true);
    });
    expect(toastMsg).toHaveBeenCalledWith('Nicht nominiert');
  });

  // Regression test: same double-submission class as setMyStatus above --
  // toggleNomination had no inFlight guard at all, unlike setMyStatus/
  // setStatusFor, so a double-tap on the nominate icon fired two concurrent
  // setNomination calls for what the user perceives as a single click.
  it('toggleNomination ignores a second call for the same event/user while the first is still in flight', async () => {
    let resolveFirst!: (v: unknown) => void;
    api.attendance.setNomination = vi
      .fn()
      .mockImplementationOnce(() => new Promise((resolve) => (resolveFirst = resolve)))
      .mockResolvedValue(true);
    const { result } = renderActions();

    let firstDone = false;
    const first = act(async () => {
      await result.current.toggleNomination('ev1', 'u2', false).then(() => (firstDone = true));
    });
    await act(async () => {
      await result.current.toggleNomination('ev1', 'u2', false);
    });

    expect(firstDone).toBe(false);
    expect(api.attendance.setNomination).toHaveBeenCalledTimes(1);

    resolveFirst(true);
    await first;
    expect(firstDone).toBe(true);
  });

  it('submitComment sets attendance and reopens event detail', async () => {
    stateRef = makeState({
      sheet: { type: 'comment', eventId: 'ev1', userId: 'u2', status: 'no' } as never,
    });
    const { result } = renderActions();
    await act(async () => {
      await result.current.submitComment('injured');
    });
    expect(api.attendance.set).toHaveBeenCalledWith('ev1', 'u2', { status: 'no', reason: 'injured' }, 'team1');
    expect(loadNotifications).toHaveBeenCalled();
    expect(stateRef.sheet).toMatchObject({ type: 'eventDetail', eventId: 'ev1' });
  });

  // Regression: the sheet-identity check used to only verify activeTeamId,
  // so a slow submitComment could clobber whatever DIFFERENT sheet the user
  // had since opened while it was in flight.
  it('submitComment does not touch the sheet if the user opened something else while in flight', async () => {
    let resolveSet!: (v: unknown) => void;
    api.attendance.set = vi.fn(() => new Promise((resolve) => (resolveSet = resolve)));
    stateRef = makeState({
      sheet: { type: 'comment', eventId: 'ev1', userId: 'u2', status: 'no' } as never,
    });
    const { result } = renderActions();

    let submitPromise!: Promise<void>;
    await act(async () => {
      submitPromise = result.current.submitComment('injured');
      // Let the mutation's internal microtasks reach mutationFn (mutateAsync
      // doesn't invoke it synchronously) before resolveSet is assigned.
      await Promise.resolve();
    });

    const somethingElse = { type: 'teams' } as never;
    stateRef = { ...stateRef, sheet: somethingElse };

    await act(async () => {
      resolveSet({});
      await submitPromise;
    });

    expect(stateRef.sheet).toBe(somethingElse);
  });
});

describe('useEventActionFeatures', () => {
  let setState: ReturnType<typeof vi.fn>;
  let toastMsg: ReturnType<typeof vi.fn>;
  let loadNotifications: ReturnType<typeof vi.fn>;
  let askConfirm: ReturnType<typeof vi.fn>;
  let openEventDetail: ReturnType<typeof vi.fn>;
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
    askConfirm = vi.fn();
    openEventDetail = vi.fn();
    logout = vi.fn();
    api = makeApi();
  });

  function renderActions() {
    return renderHook(
      () =>
        useEventActionFeatures({
          api: api as never,
          S: () => stateRef,
          setState: setState as never,
          loadNotifications: loadNotifications as never,
          toastMsg: toastMsg as never,
          askConfirm: askConfirm as never,
          openEventDetail: openEventDetail as never,
          logout: logout as never,
        }),
      { wrapper: createQueryWrapper() },
    );
  }

  it('askEventAction opens seriesAction sheet for series events', () => {
    const event = { id: 'ev1', title: 'Test', seriesId: 's1' } as never;
    const { result } = renderActions();
    act(() => {
      result.current.askEventAction('cancel', event);
    });
    expect(setState).toHaveBeenCalled();
    const patch = setState.mock.calls[0]![0];
    const resolved = typeof patch === 'function' ? patch(stateRef) : patch;
    expect(resolved.sheet).toMatchObject({ type: 'seriesAction', action: 'cancel' });
  });

  it('askEventAction runs directly for non-series events (cancel)', async () => {
    const event = { id: 'ev1', title: 'Test', seriesId: null, teamId: 'team1' } as never;
    const { result } = renderActions();
    await act(async () => {
      await result.current.askEventAction('cancel', event);
    });
    expect(api.events.setStatus).toHaveBeenCalledWith('ev1', 'cancelled', 'single', 'team1');
    expect(loadNotifications).toHaveBeenCalled();
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
    const event = { id: 'ev1', title: 'Test', seriesId: null, teamId: 'team1' } as never;
    const { result } = renderActions();
    await act(async () => {
      await result.current.runEventAction('reactivate', event, 'single');
    });
    expect(api.events.setStatus).toHaveBeenCalledWith('ev1', 'active', 'single', 'team1');
    expect(toastMsg).toHaveBeenCalledWith('Termin aktiviert');
  });

  // Regression: onConfirm used to close the sheet unconditionally (once the
  // team still matched), so a slow delete for one event could clobber
  // whatever DIFFERENT sheet the user had since opened while it was in
  // flight.
  it('runEventAction delete does not touch the sheet if the user opened something else while in flight', async () => {
    let resolveRemove!: () => void;
    api.events.remove = vi.fn(() => new Promise<void>((resolve) => (resolveRemove = resolve)));
    const event = { id: 'ev1', title: 'Test', seriesId: null, teamId: 'team1' } as never;
    const { result } = renderActions();

    act(() => {
      result.current.runEventAction('delete', event, 'single');
    });
    const cfg = askConfirm.mock.calls[0]![0];

    let confirmPromise!: Promise<void>;
    await act(async () => {
      confirmPromise = cfg.onConfirm();
      // Let the mutation's internal microtasks reach mutationFn (mutateAsync
      // doesn't invoke it synchronously) before resolveRemove is assigned.
      await Promise.resolve();
    });

    const somethingElse = { type: 'teams' } as never;
    stateRef = { ...stateRef, sheet: somethingElse };

    await act(async () => {
      resolveRemove();
      await confirmPromise;
    });

    expect(stateRef.sheet).toBe(somethingElse);
  });

  // Same race for the cancel/reactivate branch, which (for a non-series
  // event) runs directly rather than through askConfirm.
  it('runEventAction reactivate does not touch the sheet if the user opened something else while in flight', async () => {
    let resolveStatus!: (v: unknown) => void;
    api.events.setStatus = vi.fn(() => new Promise((resolve) => (resolveStatus = resolve)));
    const event = { id: 'ev1', title: 'Test', seriesId: null, teamId: 'team1' } as never;
    stateRef = { ...stateRef, sheet: { type: 'eventDetail', eventId: 'ev1' } as never };
    const { result } = renderActions();

    let actionPromise!: Promise<void>;
    await act(async () => {
      actionPromise = result.current.runEventAction('reactivate', event, 'single');
      // Let the mutation's internal microtasks reach mutationFn (mutateAsync
      // doesn't invoke it synchronously) before resolveStatus is assigned.
      await Promise.resolve();
    });

    const somethingElse = { type: 'teams' } as never;
    stateRef = { ...stateRef, sheet: somethingElse };

    await act(async () => {
      resolveStatus({});
      await actionPromise;
    });

    expect(stateRef.sheet).toBe(somethingElse);
    expect(openEventDetail).not.toHaveBeenCalled();
  });

  // Regression: cancel/reactivate/delete must scope the API call to the
  // event's OWN team, not whatever team happens to be active right now --
  // the confirm sheet that triggers these can still be open after the user
  // has switched to a different active team.
  it('runEventAction reactivate scopes the API call to event.teamId, not the active team', async () => {
    stateRef = { ...stateRef, activeTeamId: 'team2' };
    const event = { id: 'ev1', title: 'Test', seriesId: null, teamId: 'team1' } as never;
    const { result } = renderActions();
    await act(async () => {
      await result.current.runEventAction('reactivate', event, 'single');
    });
    expect(api.events.setStatus).toHaveBeenCalledWith('ev1', 'active', 'single', 'team1');
  });

  it('runEventAction delete scopes the API call to event.teamId, not the active team', async () => {
    stateRef = { ...stateRef, activeTeamId: 'team2' };
    const event = { id: 'ev1', title: 'Test', seriesId: null, teamId: 'team1' } as never;
    const { result } = renderActions();
    await act(async () => {
      await result.current.runEventAction('delete', event, 'single');
    });
    const cfg = askConfirm.mock.calls[0]![0];
    await act(async () => {
      await cfg.onConfirm();
    });
    expect(api.events.remove).toHaveBeenCalledWith('ev1', 'single', 'team1');
  });
});
