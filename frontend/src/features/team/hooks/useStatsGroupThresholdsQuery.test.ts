import { describe, it, expect, vi } from 'vitest';
import { renderHook, waitFor } from '@testing-library/react';
import { useStatsGroupThresholdsQuery } from './useStatsGroupThresholdsQuery';
import { createQueryWrapper, createTestQueryClient } from '@/test/queryTestUtils';
import { queryKeys } from '@/query/keys';

describe('useStatsGroupThresholdsQuery', () => {
  it('is disabled (does not fetch) while there is no active team', () => {
    const api = { stats: { groupThresholds: vi.fn() } };
    renderHook(() => useStatsGroupThresholdsQuery(api as never, null), { wrapper: createQueryWrapper() });
    expect(api.stats.groupThresholds).not.toHaveBeenCalled();
  });

  it("fetches the team's thresholds under the team-scoped key", async () => {
    const thresholds = [{ roleId: 'r1', mode: 'count' as const, value: 8 }];
    const api = { stats: { groupThresholds: vi.fn().mockResolvedValue(thresholds) } };
    const client = createTestQueryClient();
    const { result } = renderHook(() => useStatsGroupThresholdsQuery(api as never, 'team1'), {
      wrapper: createQueryWrapper(client),
    });
    await waitFor(() => expect(result.current.data).toEqual(thresholds));
    expect(api.stats.groupThresholds).toHaveBeenCalledWith('team1');
    expect(client.getQueryData(queryKeys.statsGroupThresholds('team1'))).toEqual(thresholds);
  });

  it('keeps each team in its own cache entry', async () => {
    const api = {
      stats: {
        groupThresholds: vi.fn((teamId: string) =>
          Promise.resolve([{ roleId: 'r_' + teamId, mode: 'percent' as const, value: 50 }]),
        ),
      },
    };
    const { result, rerender } = renderHook(({ teamId }) => useStatsGroupThresholdsQuery(api as never, teamId), {
      wrapper: createQueryWrapper(),
      initialProps: { teamId: 'team1' },
    });
    await waitFor(() => expect(result.current.data?.[0]?.roleId).toBe('r_team1'));
    rerender({ teamId: 'team2' });
    await waitFor(() => expect(result.current.data?.[0]?.roleId).toBe('r_team2'));
  });

  // The route is settings-gated: a member without settings:read gets a 403,
  // which must surface as "no data" (the settings sheet then simply shows no
  // threshold section) rather than throw during render.
  it('leaves data undefined when the request is rejected', async () => {
    const api = { stats: { groupThresholds: vi.fn().mockRejectedValue(new Error('forbidden')) } };
    const { result } = renderHook(() => useStatsGroupThresholdsQuery(api as never, 'team1'), {
      wrapper: createQueryWrapper(),
    });
    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.data).toBeUndefined();
  });
});
