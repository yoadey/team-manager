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
