import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useNotificationActions } from './useNotificationActions';
import { AuthError } from '@/utils/errors';
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
    notifUnread: 3,
    notifications: null,
    primaryColor: '#000',
    ...overrides,
  } as unknown as AppState;
}

describe('useNotificationActions', () => {
  let setState: ReturnType<typeof vi.fn>;
  let toastMsg: ReturnType<typeof vi.fn>;
  let loadNotifications: ReturnType<typeof vi.fn>;
  let logout: ReturnType<typeof vi.fn>;
  let api: { notifications: { markSeen: ReturnType<typeof vi.fn> } };
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
    logout = vi.fn();
    api = {
      notifications: {
        markSeen: vi.fn().mockResolvedValue(undefined),
      },
    };
  });

  function renderActions() {
    return renderHook(() =>
      useNotificationActions({
        api: api as never,
        S: () => stateRef,
        setState: setState as never,
        loadNotifications: loadNotifications as never,
        toastMsg: toastMsg as never,
        logout: logout as never,
      }),
    );
  }

  it('openNotifications sets sheet and loads notifications when not loaded', async () => {
    const { result } = renderActions();
    await act(async () => {
      result.current.openNotifications();
    });
    expect(setState).toHaveBeenCalledWith({ sheet: { type: 'notifications' }, notifFilter: 'all' });
    expect(loadNotifications).toHaveBeenCalled();
    expect(api.notifications.markSeen).toHaveBeenCalledWith('team1');
  });

  it('openNotifications does nothing when no activeTeamId', () => {
    stateRef = makeState({ activeTeamId: null as never });
    const { result } = renderActions();
    act(() => {
      result.current.openNotifications();
    });
    expect(setState).not.toHaveBeenCalled();
  });

  it('openNotifications skips loadNotifications when already loaded', async () => {
    stateRef = makeState({
      notifications: [{ id: 'n1', title: 'Test', unread: true }] as never,
    });
    const { result } = renderActions();
    await act(async () => {
      result.current.openNotifications();
    });
    expect(loadNotifications).not.toHaveBeenCalled();
    expect(api.notifications.markSeen).toHaveBeenCalled();
  });

  it('openNotifications marks notifications as read', async () => {
    stateRef = makeState({
      notifications: [{ id: 'n1', title: 'Test', unread: true }] as never,
    });
    const { result } = renderActions();
    await act(async () => {
      result.current.openNotifications();
    });
    expect(stateRef.notifUnread).toBe(0);
  });

  it('openNotifications shows toast on error', async () => {
    api.notifications.markSeen = vi.fn().mockRejectedValue(new Error('Network error'));
    const { result } = renderActions();
    await act(async () => {
      result.current.openNotifications();
    });
    await new Promise((r) => setTimeout(r, 10));
    expect(toastMsg).toHaveBeenCalled();
  });

  it('openNotifications triggers logout on a 401 (expired session)', async () => {
    api.notifications.markSeen = vi.fn().mockRejectedValue(new AuthError());
    const { result } = renderActions();
    await act(async () => {
      result.current.openNotifications();
    });
    await new Promise((r) => setTimeout(r, 10));
    expect(logout).toHaveBeenCalled();
  });

  it('setNotifFilter updates notifFilter in state', () => {
    const { result } = renderActions();
    act(() => {
      result.current.setNotifFilter('unread' as never);
    });
    expect(setState).toHaveBeenCalledWith({ notifFilter: 'unread' });
  });
});
