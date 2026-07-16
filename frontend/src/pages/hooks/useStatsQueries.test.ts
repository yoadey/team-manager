import { describe, it, expect, vi } from 'vitest';
import { renderHook, waitFor } from '@testing-library/react';
import { useStatsQuery } from './useStatsQueries';
import { createQueryWrapper, createTestQueryClient } from '@/test/queryTestUtils';
import type { StatsOverview } from '@/types';

function makeStats(overrides: Partial<StatsOverview> = {}): StatsOverview {
  return {
    avg: 50,
    members: [],
    events: [],
    pastCount: 0,
    from: null,
    to: null,
    ...overrides,
  };
}

describe('useStatsQuery', () => {
  it('is disabled (does not fetch) while there is no active team', () => {
    const api = { stats: { teamOverview: vi.fn() } };
    renderHook(() => useStatsQuery(api as never, null, null), { wrapper: createQueryWrapper() });
    expect(api.stats.teamOverview).not.toHaveBeenCalled();
  });

  it('fetches the team-and-range-scoped stats overview once a team id is provided', async () => {
    const api = { stats: { teamOverview: vi.fn().mockResolvedValue(makeStats({ avg: 75 })) } };
    const range = { from: '2026-01-01', to: '2026-03-01' };
    const { result } = renderHook(() => useStatsQuery(api as never, 'team1', range), {
      wrapper: createQueryWrapper(),
    });
    await waitFor(() => expect(result.current.data?.avg).toBe(75));
    expect(api.stats.teamOverview).toHaveBeenCalledWith('team1', range);
  });

  // This is the scenario the pre-migration `loadStatsSeq`/`activeTeamId`
  // guards existed to defend against: a slow response for a previously-
  // selected team must not overwrite the newly-selected team's data. The
  // team-scoped query key makes it structurally impossible instead of
  // needing a manual sequence ref.
  it('discards a stale response for a previous team after switching teams', async () => {
    let resolveTeamA!: (v: StatsOverview) => void;
    const teamAPromise = new Promise<StatsOverview>((resolve) => (resolveTeamA = resolve));
    const api = {
      stats: {
        teamOverview: vi.fn((teamId: string) =>
          teamId === 'teamA' ? teamAPromise : Promise.resolve(makeStats({ avg: 20 })),
        ),
      },
    };
    const client = createTestQueryClient();
    const { result, rerender } = renderHook(({ teamId }) => useStatsQuery(api as never, teamId, null), {
      wrapper: createQueryWrapper(client),
      initialProps: { teamId: 'teamA' },
    });

    // User switches to teamB before teamA's request resolves.
    rerender({ teamId: 'teamB' });
    await waitFor(() => expect(result.current.data?.avg).toBe(20));

    // teamA's stale response now arrives -- it must not overwrite teamB's data.
    resolveTeamA(makeStats({ avg: 99 }));
    await Promise.resolve();

    expect(result.current.data?.avg).toBe(20);
  });

  // Unlike every other migrated vertical's query, this one also varies by a
  // second dimension (the date range) -- a range change must swap to a
  // different cache entry the same way a team switch does, rather than
  // reusing/overwriting the previous range's cached data.
  it('discards a stale response for a previous range after switching ranges', async () => {
    let resolveRangeA!: (v: StatsOverview) => void;
    const rangeAPromise = new Promise<StatsOverview>((resolve) => (resolveRangeA = resolve));
    const rangeA = { from: '2026-01-01', to: '2026-02-01' };
    const rangeB = { from: '2026-02-01', to: '2026-03-01' };
    const api = {
      stats: {
        teamOverview: vi.fn((_teamId: string, range: typeof rangeA) =>
          range === rangeA ? rangeAPromise : Promise.resolve(makeStats({ avg: 20 })),
        ),
      },
    };
    const client = createTestQueryClient();
    const { result, rerender } = renderHook(({ range }) => useStatsQuery(api as never, 'team1', range), {
      wrapper: createQueryWrapper(client),
      initialProps: { range: rangeA },
    });

    // User switches to a different range before rangeA's request resolves.
    rerender({ range: rangeB });
    await waitFor(() => expect(result.current.data?.avg).toBe(20));

    // rangeA's stale response now arrives -- it must not overwrite rangeB's data.
    resolveRangeA(makeStats({ avg: 99 }));
    await Promise.resolve();

    expect(result.current.data?.avg).toBe(20);
  });
});
