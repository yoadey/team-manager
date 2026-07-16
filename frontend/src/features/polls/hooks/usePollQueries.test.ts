import { describe, it, expect, vi } from 'vitest';
import { renderHook, waitFor } from '@testing-library/react';
import { usePollsQuery } from './usePollQueries';
import { createQueryWrapper, createTestQueryClient } from '@/test/queryTestUtils';
import type { Poll } from '../types';

function makePoll(overrides: Partial<Poll> = {}): Poll {
  return {
    id: 'poll1',
    question: 'Question?',
    options: [],
    multiple: false,
    anonymous: false,
    createdAt: '2026-01-01',
    totalVotes: 0,
    myVote: null,
    ...overrides,
  };
}

describe('usePollsQuery', () => {
  it('is disabled (does not fetch) while there is no active team', () => {
    const api = { polls: { list: vi.fn() } };
    renderHook(() => usePollsQuery(api as never, null), { wrapper: createQueryWrapper() });
    expect(api.polls.list).not.toHaveBeenCalled();
  });

  it('fetches the team-scoped poll list once a team id is provided', async () => {
    const api = { polls: { list: vi.fn().mockResolvedValue([makePoll()]) } };
    const { result } = renderHook(() => usePollsQuery(api as never, 'team1'), { wrapper: createQueryWrapper() });
    await waitFor(() => expect(result.current.data).toHaveLength(1));
    expect(api.polls.list).toHaveBeenCalledWith('team1');
  });

  // This is the scenario the pre-migration `loadPollsSeq`/`activeTeamId`
  // guards existed to defend against: a slow response for a previously-
  // selected team must not overwrite the newly-selected team's data. The
  // team-scoped query key makes it structurally impossible instead of
  // needing a manual sequence ref.
  it('discards a stale response for a previous team after switching teams', async () => {
    let resolveTeamA!: (v: Poll[]) => void;
    const teamAPromise = new Promise<Poll[]>((resolve) => (resolveTeamA = resolve));
    const api = {
      polls: {
        list: vi.fn((teamId: string) =>
          teamId === 'teamA' ? teamAPromise : Promise.resolve([makePoll({ id: 'poll-b' })]),
        ),
      },
    };
    const client = createTestQueryClient();
    const { result, rerender } = renderHook(({ teamId }) => usePollsQuery(api as never, teamId), {
      wrapper: createQueryWrapper(client),
      initialProps: { teamId: 'teamA' },
    });

    // User switches to teamB before teamA's request resolves.
    rerender({ teamId: 'teamB' });
    await waitFor(() => expect(result.current.data?.[0]?.id).toBe('poll-b'));

    // teamA's stale response now arrives -- it must not overwrite teamB's data.
    resolveTeamA([makePoll({ id: 'poll-a-stale' })]);
    await Promise.resolve();

    expect(result.current.data?.[0]?.id).toBe('poll-b');
  });
});
