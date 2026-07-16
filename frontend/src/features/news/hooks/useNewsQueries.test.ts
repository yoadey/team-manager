import { describe, it, expect, vi } from 'vitest';
import { renderHook, waitFor } from '@testing-library/react';
import { useNewsQuery } from './useNewsQueries';
import { createQueryWrapper, createTestQueryClient } from '@/test/queryTestUtils';
import type { NewsItem } from '../types';

function makeNews(overrides: Partial<NewsItem> = {}): NewsItem {
  return {
    id: 'n1',
    teamId: 'team1',
    title: 'Title',
    body: 'Body',
    authorId: 'u1',
    pinned: false,
    createdAt: '2026-01-01',
    ...overrides,
  };
}

describe('useNewsQuery', () => {
  it('is disabled (does not fetch) while there is no active team', () => {
    const api = { news: { list: vi.fn() } };
    renderHook(() => useNewsQuery(api as never, null), { wrapper: createQueryWrapper() });
    expect(api.news.list).not.toHaveBeenCalled();
  });

  it('fetches the team-scoped news list once a team id is provided', async () => {
    const api = { news: { list: vi.fn().mockResolvedValue([makeNews()]) } };
    const { result } = renderHook(() => useNewsQuery(api as never, 'team1'), { wrapper: createQueryWrapper() });
    await waitFor(() => expect(result.current.data).toHaveLength(1));
    expect(api.news.list).toHaveBeenCalledWith('team1');
  });

  // This is the scenario the pre-migration `loadNewsSeq`/`activeTeamId`
  // guards existed to defend against: a slow response for a previously-
  // selected team must not overwrite the newly-selected team's data. The
  // team-scoped query key makes it structurally impossible instead of
  // needing a manual sequence ref.
  it('discards a stale response for a previous team after switching teams', async () => {
    let resolveTeamA!: (v: NewsItem[]) => void;
    const teamAPromise = new Promise<NewsItem[]>((resolve) => (resolveTeamA = resolve));
    const api = {
      news: {
        list: vi.fn((teamId: string) =>
          teamId === 'teamA' ? teamAPromise : Promise.resolve([makeNews({ id: 'news-b' })]),
        ),
      },
    };
    const client = createTestQueryClient();
    const { result, rerender } = renderHook(({ teamId }) => useNewsQuery(api as never, teamId), {
      wrapper: createQueryWrapper(client),
      initialProps: { teamId: 'teamA' },
    });

    // User switches to teamB before teamA's request resolves.
    rerender({ teamId: 'teamB' });
    await waitFor(() => expect(result.current.data?.[0]?.id).toBe('news-b'));

    // teamA's stale response now arrives -- it must not overwrite teamB's data.
    resolveTeamA([makeNews({ id: 'news-a-stale' })]);
    await Promise.resolve();

    expect(result.current.data?.[0]?.id).toBe('news-b');
  });
});
