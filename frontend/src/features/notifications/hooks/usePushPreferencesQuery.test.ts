import { describe, it, expect, vi } from 'vitest';
import { renderHook, waitFor } from '@testing-library/react';
import { usePushPreferencesQuery } from './usePushPreferencesQuery';
import { createQueryWrapper } from '@/test/queryTestUtils';
import type { PushCategoryPreferences } from '../types';

function makePrefs(overrides: Partial<PushCategoryPreferences> = {}): PushCategoryPreferences {
  return { attendance: true, events: true, news: true, polls: true, absence: true, ...overrides };
}

describe('usePushPreferencesQuery', () => {
  it('is disabled (does not fetch) while there is no active team', () => {
    const api = { push: { getPreferences: vi.fn() } };
    renderHook(() => usePushPreferencesQuery(api as never, null), { wrapper: createQueryWrapper() });
    expect(api.push.getPreferences).not.toHaveBeenCalled();
  });

  it('is disabled when the caller passes enabled=false, even with a team id', () => {
    const api = { push: { getPreferences: vi.fn() } };
    renderHook(() => usePushPreferencesQuery(api as never, 'team1', false), { wrapper: createQueryWrapper() });
    expect(api.push.getPreferences).not.toHaveBeenCalled();
  });

  it('fetches the team-scoped preferences once a team id is provided', async () => {
    const api = { push: { getPreferences: vi.fn().mockResolvedValue(makePrefs({ news: false })) } };
    const { result } = renderHook(() => usePushPreferencesQuery(api as never, 'team1'), {
      wrapper: createQueryWrapper(),
    });
    await waitFor(() => expect(result.current.data?.news).toBe(false));
    expect(api.push.getPreferences).toHaveBeenCalledWith('team1');
  });
});
