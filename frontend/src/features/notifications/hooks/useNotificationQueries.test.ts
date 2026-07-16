import { describe, it, expect, vi } from 'vitest';
import { renderHook, waitFor } from '@testing-library/react';
import { useNotificationsQuery } from './useNotificationQueries';
import { createQueryWrapper, createTestQueryClient } from '@/test/queryTestUtils';
import type { NotificationsResult } from '../types';

function makeResult(overrides: Partial<NotificationsResult> = {}): NotificationsResult {
  return {
    items: [{ id: 'n1', teamId: 'team1', type: 'news', createdAt: '2026-01-01', unread: true }],
    unreadCount: 1,
    ...overrides,
  };
}

describe('useNotificationsQuery', () => {
  it('is disabled (does not fetch) while there is no active team', () => {
    const api = { notifications: { list: vi.fn() } };
    renderHook(() => useNotificationsQuery(api as never, null), { wrapper: createQueryWrapper() });
    expect(api.notifications.list).not.toHaveBeenCalled();
  });

  it('fetches the team-scoped notification feed once a team id is provided', async () => {
    const api = { notifications: { list: vi.fn().mockResolvedValue(makeResult()) } };
    const { result } = renderHook(() => useNotificationsQuery(api as never, 'team1'), {
      wrapper: createQueryWrapper(),
    });
    await waitFor(() => expect(result.current.data?.unreadCount).toBe(1));
    expect(api.notifications.list).toHaveBeenCalledWith('team1');
  });

  // This is the scenario the pre-migration `activeTeamId` re-check inside
  // loadNotifications existed to defend against: a slow response for a
  // previously-selected team must not overwrite the newly-selected team's
  // data. The team-scoped query key makes it structurally impossible instead
  // of needing a manual check.
  it('discards a stale response for a previous team after switching teams', async () => {
    let resolveTeamA!: (v: NotificationsResult) => void;
    const teamAPromise = new Promise<NotificationsResult>((resolve) => (resolveTeamA = resolve));
    const api = {
      notifications: {
        list: vi.fn((teamId: string) =>
          teamId === 'teamA' ? teamAPromise : Promise.resolve(makeResult({ unreadCount: 5 })),
        ),
      },
    };
    const client = createTestQueryClient();
    const { result, rerender } = renderHook(({ teamId }) => useNotificationsQuery(api as never, teamId), {
      wrapper: createQueryWrapper(client),
      initialProps: { teamId: 'teamA' },
    });

    // User switches to teamB before teamA's request resolves.
    rerender({ teamId: 'teamB' });
    await waitFor(() => expect(result.current.data?.unreadCount).toBe(5));

    // teamA's stale response now arrives -- it must not overwrite teamB's data.
    resolveTeamA(makeResult({ unreadCount: 99 }));
    await Promise.resolve();

    expect(result.current.data?.unreadCount).toBe(5);
  });
});
