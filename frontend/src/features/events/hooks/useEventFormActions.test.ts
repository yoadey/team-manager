import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useEventFormActions } from './useEventFormActions';
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
    return renderHook(() =>
      useEventFormActions({
        api: {} as never,
        S: () => stateRef,
        setState: setState as never,
        refreshEvents: vi.fn().mockResolvedValue(undefined) as never,
        openEventDetail: vi.fn().mockResolvedValue(undefined) as never,
        toastMsg: vi.fn() as never,
        logout: vi.fn() as never,
      }),
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
    expect(stateRef.form).toMatchObject({ location: '' });
  });
});
