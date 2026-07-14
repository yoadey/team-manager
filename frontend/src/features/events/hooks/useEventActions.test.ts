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
    setFormVal = vi.fn();
    askConfirm = vi.fn();
    logout = vi.fn();
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
        askConfirm: askConfirm as never,
        toastMsg: toastMsg as never,
        logout: logout as never,
      }),
    );
  }

  it('openEventDetail sets sheet and loads event data', async () => {
    const { result } = renderActions();
    await act(async () => {
      await result.current.openEventDetail('ev1');
    });
    expect(api.events.get).toHaveBeenCalledWith('ev1', 'team1');
    expect(api.attendance.listForEvent).toHaveBeenCalledWith('ev1', 'team1');
    expect(api.events.listComments).toHaveBeenCalledWith('ev1', 'team1');
    expect(setState).toHaveBeenCalledWith(
      expect.objectContaining({ sheet: expect.objectContaining({ type: 'eventDetail' }) }),
    );
  });

  // Regression test: reloadDetail used to set `event: null` on a confirmed
  // 404 with no other signal, indistinguishable from the initial "still
  // loading" null -- EventDetailSheet rendered an infinite spinner either
  // way. reloadDetail must now flag eventNotFound so the component can tell
  // the two apart.
  it('openEventDetail sets eventNotFound when the event is confirmed missing', async () => {
    api.events.get = vi.fn().mockResolvedValue(null);
    const { result } = renderActions();
    await act(async () => {
      await result.current.openEventDetail('ev1');
    });
    expect(stateRef.sheet).toMatchObject({ event: null, eventNotFound: true });
  });

  it('openEventDetail does not set eventNotFound when the event loads successfully', async () => {
    const { result } = renderActions();
    await act(async () => {
      await result.current.openEventDetail('ev1');
    });
    expect(stateRef.sheet).toMatchObject({ eventNotFound: false });
  });

  // Regression test: a thrown fetch (e.g. events:none after a permission
  // downgrade, reached via a deep link/bookmark/back-forward into
  // /events/<id> -- ensureRouteData's own permission pre-check doesn't cover
  // this path) never reached the success branch that sets eventNotFound, so
  // EventDetailSheet's `if (!e) { ... return <SpinnerBox /> }` spun forever.
  // reloadDetail must close the sheet on any failure instead.
  it('openEventDetail closes the sheet instead of spinning forever when the fetch throws', async () => {
    api.events.get = vi.fn().mockRejectedValue(new Error('boom'));
    const { result } = renderActions();
    await act(async () => {
      await result.current.openEventDetail('ev1');
    });
    expect(stateRef.sheet).toBeNull();
    expect(toastMsg).toHaveBeenCalled();
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

  // Regression test: a rapid double-tap on the RSVP buttons (or a user
  // switching Yes -> No before the first request resolves) used to fire two
  // concurrent api.attendance.set calls with no guard, unlike the sibling
  // setStatusFor (roster admin view), risking an out-of-order response
  // overwriting the UI with a stale status.
  it('setMyStatus ignores a second call while the first is still in flight', async () => {
    let resolveFirst!: () => void;
    api.attendance.set = vi
      .fn()
      .mockImplementationOnce(() => new Promise<void>((resolve) => (resolveFirst = resolve)))
      .mockResolvedValue(undefined);
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

    resolveFirst();
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
    expect(api.events.addComment).toHaveBeenCalledWith('ev1', 'Great match!', 'team1');
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
    const onConfirm = askConfirm.mock.calls[0][0].onConfirm;
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
  // setNomination calls (and two refreshEvents/reloadDetail round-trips) for
  // what the user perceives as a single click.
  it('toggleNomination ignores a second call for the same event/user while the first is still in flight', async () => {
    let resolveFirst!: () => void;
    api.attendance.setNomination = vi
      .fn()
      .mockImplementationOnce(() => new Promise<void>((resolve) => (resolveFirst = resolve)))
      .mockResolvedValue(undefined);
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

    resolveFirst();
    await first;
    expect(firstDone).toBe(true);
  });

  it('reloadDetail handles API errors gracefully', async () => {
    api.events.get = vi.fn().mockRejectedValue(new Error('Network error'));
    const { result } = renderActions();
    await act(async () => {
      await result.current.reloadDetail('ev1');
    });
    expect(toastMsg).toHaveBeenCalled();
  });

  it('a slow reloadDetail for an old event cannot overwrite a newer, already-open event', async () => {
    // ev1's fetch resolves only after ev2 has already opened and resolved,
    // simulating a user quickly switching from one event's detail sheet to
    // another before the first request settles.
    let resolveEv1: (v: { id: string; title: string; date: string }) => void;
    const ev1Promise = new Promise<{ id: string; title: string; date: string }>((resolve) => {
      resolveEv1 = resolve;
    });
    api.events.get = vi.fn((eventId: string) => {
      if (eventId === 'ev1') return ev1Promise;
      return Promise.resolve({ id: 'ev2', title: 'Event Two', date: '2026-02-02' });
    });

    const { result } = renderActions();

    // Start loading ev1 but don't await it yet.
    const ev1Load = act(async () => {
      await result.current.openEventDetail('ev1');
    });

    // ev2 opens and finishes first.
    await act(async () => {
      await result.current.openEventDetail('ev2');
    });
    expect(stateRef.sheet).toMatchObject({ type: 'eventDetail', eventId: 'ev2' });

    // Now let ev1's stale response resolve.
    resolveEv1!({ id: 'ev1', title: 'Event One', date: '2026-01-01' });
    await ev1Load;

    // The sheet must still show ev2 — the stale ev1 response must not overwrite it.
    expect(stateRef.sheet).toMatchObject({ type: 'eventDetail', eventId: 'ev2' });
    expect((stateRef.sheet as { event?: { id: string } } | null)?.event?.id).toBe('ev2');
  });

  // Regression test: the eventId check above only guards against the sheet
  // having switched to a DIFFERENT event while a reloadDetail call was in
  // flight. It does NOT catch two reloadDetail calls for the SAME event
  // racing each other -- e.g. two attendance updates on different rows both
  // triggering a reload for the same open event. If the network responds
  // out of request order, the older call's stale response could still
  // overwrite the newer call's fresher data even though the sheet never
  // changed identity.
  it('a slow reloadDetail cannot overwrite a newer reload for the SAME event', async () => {
    let resolveFirst!: (v: { id: string; title: string; date: string }) => void;
    const firstPromise = new Promise<{ id: string; title: string; date: string }>((resolve) => {
      resolveFirst = resolve;
    });
    api.events.get = vi
      .fn()
      .mockReturnValueOnce(firstPromise)
      .mockResolvedValueOnce({ id: 'ev1', title: 'Fresh Title', date: '2026-03-03' });

    const { result } = renderActions();
    stateRef = makeState({ sheet: { type: 'eventDetail', eventId: 'ev1', event: null, rows: [] } as never });

    const firstReload = act(async () => {
      await result.current.reloadDetail('ev1');
    });

    // Second (newer) reload for the SAME event resolves immediately.
    await act(async () => {
      await result.current.reloadDetail('ev1');
    });
    expect((stateRef.sheet as { event?: { title: string } } | null)?.event?.title).toBe('Fresh Title');

    // The first call's stale response now arrives -- it must not overwrite
    // the second, fresher call's already-applied result.
    resolveFirst!({ id: 'ev1', title: 'Stale Title', date: '2026-01-01' });
    await firstReload;

    expect((stateRef.sheet as { event?: { title: string } } | null)?.event?.title).toBe('Fresh Title');
  });

  it('submitComment sets attendance and reopens event detail', async () => {
    stateRef = makeState({
      sheet: { type: 'comment', eventId: 'ev1', userId: 'u2', status: 'no' } as never,
      form: { commentText: 'injured' },
    });
    const { result } = renderActions();
    await act(async () => {
      await result.current.submitComment();
    });
    expect(api.attendance.set).toHaveBeenCalledWith('ev1', 'u2', { status: 'no', reason: 'injured' }, 'team1');
    expect(stateRef.sheet).toMatchObject({ type: 'eventDetail', eventId: 'ev1' });
  });

  // Regression: the sheet-identity check used to only verify activeTeamId,
  // so a slow submitComment could clobber whatever DIFFERENT sheet the user
  // had since opened while it was in flight.
  it('submitComment does not touch the sheet if the user opened something else while in flight', async () => {
    let resolveSet!: () => void;
    api.attendance.set = vi.fn(() => new Promise<void>((resolve) => (resolveSet = resolve)));
    stateRef = makeState({
      sheet: { type: 'comment', eventId: 'ev1', userId: 'u2', status: 'no' } as never,
      form: { commentText: 'injured' },
    });
    const { result } = renderActions();

    let submitPromise!: Promise<void>;
    act(() => {
      submitPromise = result.current.submitComment();
    });

    const somethingElse = { type: 'teams' } as never;
    stateRef = { ...stateRef, sheet: somethingElse };

    await act(async () => {
      resolveSet();
      await submitPromise;
    });

    expect(stateRef.sheet).toBe(somethingElse);
  });
});

describe('useEventActionFeatures', () => {
  let setState: ReturnType<typeof vi.fn>;
  let toastMsg: ReturnType<typeof vi.fn>;
  let refreshEvents: ReturnType<typeof vi.fn>;
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
    refreshEvents = vi.fn().mockResolvedValue(undefined);
    askConfirm = vi.fn();
    openEventDetail = vi.fn().mockResolvedValue(undefined);
    logout = vi.fn();
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
        logout: logout as never,
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
    const event = { id: 'ev1', title: 'Test', seriesId: null, teamId: 'team1' } as never;
    const { result } = renderActions();
    await act(async () => {
      await result.current.askEventAction('cancel', event);
    });
    expect(api.events.setStatus).toHaveBeenCalledWith('ev1', 'cancelled', 'single', 'team1');
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
    const cfg = askConfirm.mock.calls[0][0];

    let confirmPromise!: Promise<void>;
    act(() => {
      confirmPromise = cfg.onConfirm();
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
    let resolveStatus!: () => void;
    api.events.setStatus = vi.fn(() => new Promise<void>((resolve) => (resolveStatus = resolve)));
    const event = { id: 'ev1', title: 'Test', seriesId: null, teamId: 'team1' } as never;
    stateRef = { ...stateRef, sheet: { type: 'eventDetail', eventId: 'ev1' } as never };
    const { result } = renderActions();

    let actionPromise!: Promise<void>;
    act(() => {
      actionPromise = result.current.runEventAction('reactivate', event, 'single');
    });

    const somethingElse = { type: 'teams' } as never;
    stateRef = { ...stateRef, sheet: somethingElse };

    await act(async () => {
      resolveStatus();
      await actionPromise;
    });

    expect(stateRef.sheet).toBe(somethingElse);
    expect(openEventDetail).not.toHaveBeenCalled();
  });
});
