import { describe, it, expect, vi } from 'vitest';
import { renderHook, waitFor, act } from '@testing-library/react';
import { usePushPreferencesActions } from './usePushPreferencesActions';
import { createQueryWrapper } from '@/test/queryTestUtils';
import type { PushCategoryPreferences } from '../types';

function makePrefs(overrides: Partial<PushCategoryPreferences> = {}): PushCategoryPreferences {
  return { attendance: true, events: true, news: true, polls: true, absence: true, ...overrides };
}

describe('usePushPreferencesActions', () => {
  it('defaults to everything enabled before the query resolves', () => {
    const api = { push: { getPreferences: vi.fn(() => new Promise(() => {})), setPreferences: vi.fn() } };
    const { result } = renderHook(() => usePushPreferencesActions(api as never, 'team1', vi.fn(), true), {
      wrapper: createQueryWrapper(),
    });
    expect(result.current.prefs).toEqual(makePrefs());
    expect(result.current.isLoading).toBe(true);
  });

  it('loads the team-scoped stored preferences', async () => {
    const api = {
      push: { getPreferences: vi.fn().mockResolvedValue(makePrefs({ polls: false })), setPreferences: vi.fn() },
    };
    const { result } = renderHook(() => usePushPreferencesActions(api as never, 'team1', vi.fn(), true), {
      wrapper: createQueryWrapper(),
    });
    await waitFor(() => expect(result.current.isLoading).toBe(false));
    expect(result.current.prefs.polls).toBe(false);
  });

  it('setCategory saves the whole preference object with only the toggled category changed', async () => {
    const setPreferences = vi.fn().mockResolvedValue(undefined);
    const api = { push: { getPreferences: vi.fn().mockResolvedValue(makePrefs()), setPreferences } };
    const { result } = renderHook(() => usePushPreferencesActions(api as never, 'team1', vi.fn(), true), {
      wrapper: createQueryWrapper(),
    });
    await waitFor(() => expect(result.current.isLoading).toBe(false));

    await act(async () => {
      result.current.setCategory('news', false);
      await waitFor(() => expect(setPreferences).toHaveBeenCalled());
    });

    expect(setPreferences).toHaveBeenCalledWith('team1', makePrefs({ news: false }));
    await waitFor(() => expect(result.current.prefs.news).toBe(false));
  });

  it('reports a toast error and leaves cached preferences unchanged when saving fails', async () => {
    const toastMsg = vi.fn();
    const setPreferences = vi.fn().mockRejectedValue(new Error('network down'));
    const api = { push: { getPreferences: vi.fn().mockResolvedValue(makePrefs()), setPreferences } };
    const { result } = renderHook(() => usePushPreferencesActions(api as never, 'team1', toastMsg, true), {
      wrapper: createQueryWrapper(),
    });
    await waitFor(() => expect(result.current.isLoading).toBe(false));

    await act(async () => {
      result.current.setCategory('events', false);
      await waitFor(() => expect(toastMsg).toHaveBeenCalled());
    });

    expect(result.current.prefs.events).toBe(true);
  });
});
