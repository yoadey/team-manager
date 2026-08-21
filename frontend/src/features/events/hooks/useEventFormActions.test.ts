import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act, waitFor } from '@testing-library/react';
import { useEventFormActions } from './useEventFormActions';
import { createQueryWrapper, createTestQueryClient } from '@/test/queryTestUtils';
import type { AppState } from '@/context/AppContext';
import { todayStr } from '@/styles/tokens';

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
    roles: [{ id: 'r1', name: 'Trainer' }],
    notifUnread: 0,
    notifications: [],
    primaryColor: '#000',
    ...overrides,
  } as unknown as AppState;
}

describe('useEventFormActions', () => {
  let stateRef: AppState;
  let setState: ReturnType<typeof vi.fn>;

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
  });

  function renderActions() {
    return renderHook(
      () =>
        useEventFormActions({
          api: {} as never,
          S: () => stateRef,
          setState: setState as never,
          teamId: stateRef.activeTeamId,
          loadNotifications: vi.fn().mockResolvedValue(undefined) as never,
          openEventDetail: vi.fn() as never,
          toastMsg: vi.fn() as never,
          logout: vi.fn() as never,
        }),
      { wrapper: createQueryWrapper() },
    );
  }

  // Regression test: a new event's location used to be prefilled with the
  // literal German venue name 'Tanzsporthalle Eilendorf' as an actual form
  // value, independent of the active team -- every club on the platform,
  // not just demo data, got this pre-filled and could publish a real event
  // with a bogus address if not manually cleared.
  it('openEventForm defaults a new event location to empty', () => {
    const { result } = renderActions();
    act(() => {
      result.current.openEventForm(null);
    });
    expect(stateRef.sheet!.formInitial).toMatchObject({ location: '' });
  });

  it('openEventForm pre-selects a new event date when a date is passed (e.g. calendar double-click)', () => {
    const { result } = renderActions();
    act(() => {
      result.current.openEventForm(null, '2026-05-01');
    });
    expect(stateRef.sheet!.formInitial).toMatchObject({ date: '2026-05-01' });
  });

  it('openEventForm defaults a new event date to today when no date is passed', () => {
    const { result } = renderActions();
    act(() => {
      result.current.openEventForm(null);
    });
    expect(stateRef.sheet!.formInitial).toMatchObject({ date: todayStr() });
  });

  it('openEventForm seeds the cancellation lead-time hours/minutes fields from an existing event, 0/0 for a new one', () => {
    const { result } = renderActions();
    act(() => {
      result.current.openEventForm(null);
    });
    expect(stateRef.sheet!.formInitial).toMatchObject({ cancelLeadHours: 0, cancelLeadMinutes: 0 });

    act(() => {
      result.current.openEventForm({
        id: 'ev1',
        type: 'training',
        title: 'Training',
        date: '2026-07-05',
        cancelLeadMinutes: 90,
      } as never);
    });
    expect(stateRef.sheet!.formInitial).toMatchObject({ cancelLeadHours: 1, cancelLeadMinutes: 30 });
  });

  it('duplicateEvent opens a create-mode form pre-filled from the source event, stripping seriesId and resetting the date', () => {
    const { result } = renderActions();
    act(() => {
      result.current.duplicateEvent(
        {
          id: 'ev1',
          seriesId: 'series1',
          type: 'training',
          title: 'Weekly Training',
          date: '2020-01-01',
          multiDayEndDate: null,
          location: 'Halle',
        } as never,
        '2026-05-01',
      );
    });
    expect(stateRef.sheet!.mode).toBe('create');
    expect(stateRef.sheet!.eventId).toBeUndefined();
    expect(stateRef.sheet!.formInitial).toMatchObject({
      title: 'Weekly Training',
      location: 'Halle',
      seriesId: null,
      recurring: false,
      date: '2026-05-01',
      multiDayEndDate: '',
    });
  });

  it('duplicateEvent preserves a multi-day span length, anchored to the new date', () => {
    const { result } = renderActions();
    act(() => {
      result.current.duplicateEvent(
        {
          id: 'ev1',
          seriesId: null,
          type: 'training',
          title: 'Camp',
          date: '2020-01-01',
          multiDayEndDate: '2020-01-03',
        } as never,
        '2026-05-01',
      );
    });
    expect(stateRef.sheet!.formInitial).toMatchObject({ date: '2026-05-01', multiDayEndDate: '2026-05-03' });
  });

  it("duplicateEvent recombines the source event's owning team into crossTeamIds when duplicating via a non-owning targeted team, instead of dropping it", () => {
    // Source event is owned by team1 and additionally targets team2;
    // event.crossTeamIds (as the server returns it) always excludes the
    // owning team, so it's just ['team2'] here regardless of which team's
    // URL fetched it. Duplicating while viewing via team2 (activeTeamId)
    // means the new event's implicit owner is team2 -- the original owner,
    // team1, must be recombined into the duplicate's crossTeamIds or it's
    // silently lost.
    stateRef = makeState({
      activeTeamId: 'team2',
      teams: [
        { id: 'team1', name: 'Team 1', myPerms: { events: 'write' } },
        { id: 'team2', name: 'Team 2', myPerms: { events: 'write' } },
      ] as never,
    });
    const { result } = renderActions();
    act(() => {
      result.current.duplicateEvent({
        id: 'ev1',
        teamId: 'team1',
        crossTeamIds: ['team2'],
        seriesId: null,
        type: 'training',
        title: 'Shared Training',
        date: '2020-01-01',
        multiDayEndDate: null,
      } as never);
    });
    expect(stateRef.sheet!.formInitial).toMatchObject({ crossTeamIds: ['team1'] });
  });

  it('duplicateEvent drops target teams the duplicating user lacks events:write in', () => {
    stateRef = makeState({
      activeTeamId: 'team1',
      teams: [
        { id: 'team2', name: 'Team 2', myPerms: { events: 'read' } },
        { id: 'team3', name: 'Team 3', myPerms: { events: 'write' } },
      ] as never,
    });
    const { result } = renderActions();
    act(() => {
      result.current.duplicateEvent({
        id: 'ev1',
        teamId: 'team1',
        crossTeamIds: ['team2', 'team3'],
        seriesId: null,
        type: 'training',
        title: 'Shared Training',
        date: '2020-01-01',
        multiDayEndDate: null,
      } as never);
    });
    expect(stateRef.sheet!.formInitial).toMatchObject({ crossTeamIds: ['team3'] });
  });

  it('saveEvent forwards cancelLeadMinutes as the combined hours+minutes total', async () => {
    const api = { events: { create: vi.fn().mockResolvedValue({ id: 'ev1' }) } };
    const formValues = {
      type: 'training',
      title: 'Test',
      date: '2026-01-01',
      cancelLeadHours: 2,
      cancelLeadMinutes: 15,
    } as never;
    stateRef = makeState({ sheet: { type: 'eventForm', mode: 'create', formInitial: formValues } as never });
    const { result } = renderHook(
      () =>
        useEventFormActions({
          api: api as never,
          S: () => stateRef,
          setState: setState as never,
          teamId: stateRef.activeTeamId,
          loadNotifications: vi.fn().mockResolvedValue(undefined) as never,
          openEventDetail: vi.fn() as never,
          toastMsg: vi.fn() as never,
          logout: vi.fn() as never,
        }),
      { wrapper: createQueryWrapper() },
    );
    await act(async () => {
      await result.current.saveEvent(formValues);
    });
    expect(api.events.create).toHaveBeenCalledWith('team1', expect.objectContaining({ cancelLeadMinutes: 135 }));
  });

  // Regression: saving an event that adds or removes a cross-team target
  // must invalidate every affected team's query cache, not just the team
  // the request was made through -- mirrors useEventStatusMutation's
  // identical allTeamIds fan-out for cancel/reactivate. team3 (removed by
  // this edit) wouldn't otherwise learn the event is gone from its list.
  it('saveEvent invalidates every current AND previously-targeted team when crossTeamIds changes', async () => {
    const api = {
      events: {
        update: vi.fn().mockResolvedValue({ id: 'ev1', teamId: 'team1', crossTeamIds: ['team2'] }),
      },
    };
    const formInitial = {
      type: 'training',
      title: 'Test',
      date: '2026-01-01',
      crossTeamIds: ['team3'],
    };
    stateRef = makeState({
      activeTeamId: 'team1',
      sheet: { type: 'eventForm', mode: 'edit', eventId: 'ev1', formInitial } as never,
    });
    const client = createTestQueryClient();
    const invalidateSpy = vi.spyOn(client, 'invalidateQueries');
    const { result } = renderHook(
      () =>
        useEventFormActions({
          api: api as never,
          S: () => stateRef,
          setState: setState as never,
          teamId: stateRef.activeTeamId,
          loadNotifications: vi.fn().mockResolvedValue(undefined) as never,
          openEventDetail: vi.fn() as never,
          toastMsg: vi.fn() as never,
          logout: vi.fn() as never,
        }),
      { wrapper: createQueryWrapper(client) },
    );
    await act(async () => {
      await result.current.saveEvent({ ...formInitial, crossTeamIds: ['team2'] } as never);
    });
    const invalidatedTeamIds = invalidateSpy.mock.calls.map((c) => (c[0] as { queryKey: unknown[] }).queryKey[1]);
    expect(invalidatedTeamIds).toEqual(expect.arrayContaining(['team1', 'team2', 'team3']));
  });

  it('saveEvent sends endDate instead of repeatWeeks when the "until date" recurrence mode is selected', async () => {
    const api = { events: { create: vi.fn().mockResolvedValue({ id: 'ev1' }) } };
    const formValues = {
      type: 'training',
      title: 'Test',
      date: '2026-01-01',
      recurring: true,
      repeatMode: 'until',
      repeatEndDate: '2026-03-01',
      repeatWeeks: 8,
    } as never;
    stateRef = makeState({ sheet: { type: 'eventForm', mode: 'create', formInitial: formValues } as never });
    const { result } = renderHook(
      () =>
        useEventFormActions({
          api: api as never,
          S: () => stateRef,
          setState: setState as never,
          teamId: stateRef.activeTeamId,
          loadNotifications: vi.fn().mockResolvedValue(undefined) as never,
          openEventDetail: vi.fn() as never,
          toastMsg: vi.fn() as never,
          logout: vi.fn() as never,
        }),
      { wrapper: createQueryWrapper() },
    );
    await act(async () => {
      await result.current.saveEvent(formValues);
    });
    const payload = api.events.create.mock.calls[0]![1] as Record<string, unknown>;
    expect(payload).toMatchObject({ endDate: '2026-03-01', recurring: true });
    expect(payload).not.toHaveProperty('repeatWeeks');
  });

  it('saveEvent forwards multiDayEndDate for a non-recurring event', async () => {
    const api = { events: { create: vi.fn().mockResolvedValue({ id: 'ev1' }) } };
    const formValues = {
      type: 'training',
      title: 'Camp',
      date: '2026-01-01',
      multiDayEndDate: '2026-01-03',
      recurring: false,
    } as never;
    stateRef = makeState({ sheet: { type: 'eventForm', mode: 'create', formInitial: formValues } as never });
    const { result } = renderHook(
      () =>
        useEventFormActions({
          api: api as never,
          S: () => stateRef,
          setState: setState as never,
          teamId: stateRef.activeTeamId,
          loadNotifications: vi.fn().mockResolvedValue(undefined) as never,
          openEventDetail: vi.fn() as never,
          toastMsg: vi.fn() as never,
          logout: vi.fn() as never,
        }),
      { wrapper: createQueryWrapper() },
    );
    await act(async () => {
      await result.current.saveEvent(formValues);
    });
    expect(api.events.create).toHaveBeenCalledWith(
      'team1',
      expect.objectContaining({ multiDayEndDate: '2026-01-03' }),
    );
  });

  it('saveEvent clears multiDayEndDate for a recurring event even if the field still holds a value', async () => {
    const api = { events: { create: vi.fn().mockResolvedValue({ id: 'ev1' }) } };
    const formValues = {
      type: 'training',
      title: 'Weekly',
      date: '2026-01-01',
      multiDayEndDate: '2026-01-03',
      recurring: true,
      repeatMode: 'weeks',
      repeatWeeks: 8,
    } as never;
    stateRef = makeState({ sheet: { type: 'eventForm', mode: 'create', formInitial: formValues } as never });
    const { result } = renderHook(
      () =>
        useEventFormActions({
          api: api as never,
          S: () => stateRef,
          setState: setState as never,
          teamId: stateRef.activeTeamId,
          loadNotifications: vi.fn().mockResolvedValue(undefined) as never,
          openEventDetail: vi.fn() as never,
          toastMsg: vi.fn() as never,
          logout: vi.fn() as never,
        }),
      { wrapper: createQueryWrapper() },
    );
    await act(async () => {
      await result.current.saveEvent(formValues);
    });
    expect(api.events.create).toHaveBeenCalledWith('team1', expect.objectContaining({ multiDayEndDate: '' }));
  });

  it('saveEvent creates the event and reports savingEvent while the mutation is in flight', async () => {
    let resolveCreate!: (v: unknown) => void;
    const api = { events: { create: vi.fn(() => new Promise((resolve) => (resolveCreate = resolve))) } };
    const formValues = { type: 'training', title: 'Test', date: '2026-01-01', nominatedRoleIds: ['r1'] } as never;
    stateRef = makeState({
      sheet: { type: 'eventForm', mode: 'create', formInitial: formValues } as never,
    });
    const { result } = renderHook(
      () =>
        useEventFormActions({
          api: api as never,
          S: () => stateRef,
          setState: setState as never,
          teamId: stateRef.activeTeamId,
          loadNotifications: vi.fn().mockResolvedValue(undefined) as never,
          openEventDetail: vi.fn() as never,
          toastMsg: vi.fn() as never,
          logout: vi.fn() as never,
        }),
      { wrapper: createQueryWrapper() },
    );

    let savePromise!: Promise<void>;
    act(() => {
      savePromise = result.current.saveEvent(formValues);
    });
    await waitFor(() => expect(result.current.savingEvent).toBe(true));

    await act(async () => {
      resolveCreate({ id: 'ev1' });
      await savePromise;
    });
    expect(api.events.create).toHaveBeenCalledWith('team1', expect.objectContaining({ title: 'Test' }));
    await waitFor(() => expect(result.current.savingEvent).toBe(false));
  });
});
