import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act, waitFor } from '@testing-library/react';
import { useEventFormActions } from './useEventFormActions';
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

  it('openEventForm seeds the RSVP deadline field from an existing event, empty for a new one', () => {
    const { result } = renderActions();
    act(() => {
      result.current.openEventForm(null);
    });
    expect(stateRef.sheet!.formInitial).toMatchObject({ rsvpDeadline: '' });

    act(() => {
      result.current.openEventForm({
        id: 'ev1',
        type: 'training',
        title: 'Training',
        date: '2026-07-05',
        rsvpDeadline: '2026-07-04T18:30:00.000Z',
      } as never);
    });
    expect(stateRef.sheet!.formInitial).toMatchObject({ rsvpDeadline: '2026-07-04T18:30' });
  });

  it('saveEvent forwards rsvpDeadline as an ISO timestamp, parsed from the datetime-local form value', async () => {
    const api = { events: { create: vi.fn().mockResolvedValue({ id: 'ev1' }) } };
    const formValues = {
      type: 'training',
      title: 'Test',
      date: '2026-01-01',
      rsvpDeadline: '2025-12-31T18:00',
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
      expect.objectContaining({ rsvpDeadline: '2025-12-31T18:00:00.000Z' }),
    );
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
