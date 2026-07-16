import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useNotificationActions } from './useNotificationActions';
import { createQueryWrapper, createTestQueryClient } from '@/test/queryTestUtils';
import { queryKeys } from '@/query/keys';
import { AuthError } from '@/utils/errors';
import type { AppState } from '@/context/AppContext';
import type { NotificationsResult } from '../types';

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
    finances: null,
    stats: null,
    statsRange: null,
    teams: [],
    roles: [],
    primaryColor: '#000',
    ...overrides,
  } as unknown as AppState;
}

describe('useNotificationActions', () => {
  let setState: ReturnType<typeof vi.fn>;
  let toastMsg: ReturnType<typeof vi.fn>;
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
    logout = vi.fn();
    api = {
      notifications: {
        markSeen: vi.fn().mockResolvedValue(undefined),
      },
    };
  });

  function renderActions() {
    return renderHook(
      () =>
        useNotificationActions({
          api: api as never,
          setState: setState as never,
          teamId: stateRef.activeTeamId,
          toastMsg: toastMsg as never,
          logout: logout as never,
        }),
      { wrapper: createQueryWrapper() },
    );
  }

  it('openNotifications sets sheet and marks notifications seen', async () => {
    const { result } = renderActions();
    await act(async () => {
      result.current.openNotifications();
    });
    expect(setState).toHaveBeenCalledWith({ sheet: { type: 'notifications' }, notifFilter: 'all' });
    expect(api.notifications.markSeen).toHaveBeenCalledWith('team1');
  });

  it('openNotifications does nothing when no activeTeamId', () => {
    stateRef = makeState({ activeTeamId: null as never });
    const { result } = renderActions();
    act(() => {
      result.current.openNotifications();
    });
    expect(setState).not.toHaveBeenCalled();
    expect(api.notifications.markSeen).not.toHaveBeenCalled();
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

  // The pre-migration behavior marked notifications read in-memory without a
  // refetch (markSeen doesn't change which notifications exist, only their
  // `unread` flag), so this asserts the mutation writes that result directly
  // into the query cache rather than merely invalidating it.
  it('openNotifications marks every cached notification as read and zeroes the unread count', async () => {
    const client = createTestQueryClient();
    const seeded: NotificationsResult = {
      items: [
        { id: 'n1', teamId: 'team1', type: 'news', createdAt: '2026-01-01', unread: true },
        { id: 'n2', teamId: 'team1', type: 'poll', createdAt: '2026-01-02', unread: true },
      ],
      unreadCount: 2,
    };
    client.setQueryData(queryKeys.notifications('team1'), seeded);

    const { result } = renderHook(
      () =>
        useNotificationActions({
          api: api as never,
          setState: setState as never,
          teamId: stateRef.activeTeamId,
          toastMsg: toastMsg as never,
          logout: logout as never,
        }),
      { wrapper: createQueryWrapper(client) },
    );
    await act(async () => {
      result.current.openNotifications();
    });

    const cached = client.getQueryData<NotificationsResult>(queryKeys.notifications('team1'));
    expect(cached?.unreadCount).toBe(0);
    expect(cached?.items.every((n) => !n.unread)).toBe(true);
  });

  it('setNotifFilter updates notifFilter in state', () => {
    const { result } = renderActions();
    act(() => {
      result.current.setNotifFilter('unread' as never);
    });
    expect(setState).toHaveBeenCalledWith({ notifFilter: 'unread' });
  });
});
