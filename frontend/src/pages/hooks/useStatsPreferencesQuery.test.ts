import { describe, it, expect, vi } from 'vitest';
import { renderHook, waitFor } from '@testing-library/react';
import { useStatsPreferencesQuery, useStatsPresetsQuery } from './useStatsPreferencesQuery';
import { createQueryWrapper } from '@/test/queryTestUtils';

describe('useStatsPreferencesQuery', () => {
  it('is disabled (does not fetch) while there is no active team', () => {
    const api = { statsPrefs: { getPreferences: vi.fn() } };
    renderHook(() => useStatsPreferencesQuery(api as never, null), { wrapper: createQueryWrapper() });
    expect(api.statsPrefs.getPreferences).not.toHaveBeenCalled();
  });

  it('fetches the team-scoped preferences once a team id is provided', async () => {
    const api = {
      statsPrefs: { getPreferences: vi.fn().mockResolvedValue({ range: { from: '2026-01-01', to: '2026-06-30' }, presetId: null }) },
    };
    const { result } = renderHook(() => useStatsPreferencesQuery(api as never, 'team1'), {
      wrapper: createQueryWrapper(),
    });
    await waitFor(() => expect(result.current.data?.range?.from).toBe('2026-01-01'));
    expect(api.statsPrefs.getPreferences).toHaveBeenCalledWith('team1');
  });
});

describe('useStatsPresetsQuery', () => {
  it('is disabled (does not fetch) while there is no active team', () => {
    const api = { statsPrefs: { listPresets: vi.fn() } };
    renderHook(() => useStatsPresetsQuery(api as never, null), { wrapper: createQueryWrapper() });
    expect(api.statsPrefs.listPresets).not.toHaveBeenCalled();
  });

  it('fetches the team-scoped presets once a team id is provided', async () => {
    const preset = { id: 'p1', name: 'Saison 2026/27', from: '2026-08-01', to: '2027-05-31' };
    const api = { statsPrefs: { listPresets: vi.fn().mockResolvedValue([preset]) } };
    const { result } = renderHook(() => useStatsPresetsQuery(api as never, 'team1'), {
      wrapper: createQueryWrapper(),
    });
    await waitFor(() => expect(result.current.data).toEqual([preset]));
    expect(api.statsPrefs.listPresets).toHaveBeenCalledWith('team1');
  });
});
