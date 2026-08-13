import { describe, it, expect, vi } from 'vitest';
import { renderHook, waitFor, act } from '@testing-library/react';
import { useStatsPreferencesActions } from './useStatsPreferencesActions';
import { createQueryWrapper } from '@/test/queryTestUtils';

function makeApi(overrides: Record<string, unknown> = {}) {
  return {
    statsPrefs: {
      getPreferences: vi.fn().mockResolvedValue({ range: null, presetId: null }),
      setPreferences: vi.fn().mockResolvedValue(undefined),
      listPresets: vi.fn().mockResolvedValue([]),
      createPreset: vi.fn(),
      updatePreset: vi.fn(),
      deletePreset: vi.fn().mockResolvedValue(undefined),
      ...overrides,
    },
  };
}

describe('useStatsPreferencesActions', () => {
  it('loads the caller preferences and saved presets for the active team', async () => {
    const preset = { id: 'p1', name: 'Saison 2026/27', from: '2026-08-01', to: '2027-05-31' };
    const api = makeApi({ listPresets: vi.fn().mockResolvedValue([preset]) });
    const { result } = renderHook(() => useStatsPreferencesActions(api as never, 'team1', vi.fn()), {
      wrapper: createQueryWrapper(),
    });
    await waitFor(() => expect(result.current.preferencesLoaded).toBe(true));
    expect(result.current.presets).toEqual([preset]);
  });

  it('saveSelection persists a full range with its preset id', async () => {
    const api = makeApi();
    const { result } = renderHook(() => useStatsPreferencesActions(api as never, 'team1', vi.fn()), {
      wrapper: createQueryWrapper(),
    });
    await waitFor(() => expect(result.current.preferencesLoaded).toBe(true));

    act(() => {
      result.current.saveSelection({ from: '2026-01-01', to: '2026-06-30' }, 'p1');
    });

    await waitFor(() => expect(api.statsPrefs.setPreferences).toHaveBeenCalledWith('team1', { from: '2026-01-01', to: '2026-06-30' }, 'p1'));
  });

  it('saveSelection is a no-op when the range is not fully specified', () => {
    const api = makeApi();
    const { result } = renderHook(() => useStatsPreferencesActions(api as never, 'team1', vi.fn()), {
      wrapper: createQueryWrapper(),
    });

    act(() => {
      result.current.saveSelection({ from: '2026-01-01', to: null }, null);
    });

    expect(api.statsPrefs.setPreferences).not.toHaveBeenCalled();
  });

  it('createPreset adds the new preset to the cached list', async () => {
    const created = { id: 'p2', name: 'Winterpause', from: '2026-12-01', to: '2027-01-31' };
    const api = makeApi({ createPreset: vi.fn().mockResolvedValue(created) });
    const { result } = renderHook(() => useStatsPreferencesActions(api as never, 'team1', vi.fn()), {
      wrapper: createQueryWrapper(),
    });
    await waitFor(() => expect(result.current.preferencesLoaded).toBe(true));

    await act(async () => {
      await result.current.createPreset('Winterpause', { from: '2026-12-01', to: '2027-01-31' });
    });

    expect(api.statsPrefs.createPreset).toHaveBeenCalledWith('team1', 'Winterpause', { from: '2026-12-01', to: '2027-01-31' });
    await waitFor(() => expect(result.current.presets).toEqual([created]));
  });

  it('deletePreset removes it from the cached list and clears a matching active presetId', async () => {
    const preset = { id: 'p1', name: 'Saison 2026/27', from: '2026-08-01', to: '2027-05-31' };
    const api = makeApi({
      listPresets: vi.fn().mockResolvedValue([preset]),
      getPreferences: vi.fn().mockResolvedValue({ range: { from: '2026-08-01', to: '2027-05-31' }, presetId: 'p1' }),
    });
    const { result } = renderHook(() => useStatsPreferencesActions(api as never, 'team1', vi.fn()), {
      wrapper: createQueryWrapper(),
    });
    await waitFor(() => expect(result.current.presets).toEqual([preset]));
    await waitFor(() => expect(result.current.preferences?.presetId).toBe('p1'));

    act(() => {
      result.current.deletePreset('p1');
    });

    await waitFor(() => expect(result.current.presets).toEqual([]));
    await waitFor(() => expect(result.current.preferences?.presetId).toBeNull());
  });
});
