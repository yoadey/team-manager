import { describe, it, expect, vi } from 'vitest';
import { renderHook, waitFor } from '@testing-library/react';
import { useFinanceOverviewQuery } from './useFinanceQueries';
import { createQueryWrapper, createTestQueryClient } from '@/test/queryTestUtils';
import type { FinanceOverview } from '../types';

function makeOverview(overrides: Partial<FinanceOverview> = {}): FinanceOverview {
  return {
    balance: 0,
    income: 0,
    expense: 0,
    transactions: [],
    penalties: [],
    assignments: [],
    openPenalties: [],
    openPenaltySum: 0,
    contributions: [],
    contribOpen: 0,
    ...overrides,
  };
}

describe('useFinanceOverviewQuery', () => {
  it('is disabled (does not fetch) while there is no active team', () => {
    const api = { finances: { overview: vi.fn() } };
    renderHook(() => useFinanceOverviewQuery(api as never, null), { wrapper: createQueryWrapper() });
    expect(api.finances.overview).not.toHaveBeenCalled();
  });

  it('fetches the team-scoped finance overview once a team id is provided', async () => {
    const api = { finances: { overview: vi.fn().mockResolvedValue(makeOverview({ balance: 42 })) } };
    const { result } = renderHook(() => useFinanceOverviewQuery(api as never, 'team1'), {
      wrapper: createQueryWrapper(),
    });
    await waitFor(() => expect(result.current.data?.balance).toBe(42));
    expect(api.finances.overview).toHaveBeenCalledWith('team1');
  });

  // This is the scenario the pre-migration `loadFinancesSeq`/`activeTeamId`
  // guards existed to defend against: a slow response for a previously-
  // selected team must not overwrite the newly-selected team's data. The
  // team-scoped query key makes it structurally impossible instead of
  // needing a manual sequence ref.
  it('discards a stale response for a previous team after switching teams', async () => {
    let resolveTeamA!: (v: FinanceOverview) => void;
    const teamAPromise = new Promise<FinanceOverview>((resolve) => (resolveTeamA = resolve));
    const api = {
      finances: {
        overview: vi.fn((teamId: string) =>
          teamId === 'teamA' ? teamAPromise : Promise.resolve(makeOverview({ balance: 2 })),
        ),
      },
    };
    const client = createTestQueryClient();
    const { result, rerender } = renderHook(({ teamId }) => useFinanceOverviewQuery(api as never, teamId), {
      wrapper: createQueryWrapper(client),
      initialProps: { teamId: 'teamA' },
    });

    // User switches to teamB before teamA's request resolves.
    rerender({ teamId: 'teamB' });
    await waitFor(() => expect(result.current.data?.balance).toBe(2));

    // teamA's stale response now arrives -- it must not overwrite teamB's data.
    resolveTeamA(makeOverview({ balance: 1 }));
    await Promise.resolve();

    expect(result.current.data?.balance).toBe(2);
  });
});
