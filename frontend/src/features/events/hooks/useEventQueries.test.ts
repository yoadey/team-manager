import { describe, it, expect, vi } from 'vitest';
import { renderHook, waitFor } from '@testing-library/react';
import { useEventsQuery, useEventDetailQuery } from './useEventQueries';
import { createQueryWrapper, createTestQueryClient } from '@/test/queryTestUtils';
import type { TeamEvent } from '../types';

function makeEvent(overrides: Partial<TeamEvent> = {}): TeamEvent {
  return {
    id: 'ev1',
    teamId: 'team1',
    type: 'training',
    title: 'Training',
    date: '2026-01-05',
    location: '',
    note: '',
    meetTime: null,
    startTime: '18:00',
    endTime: null,
    meetTimeMandatory: false,
    responseMode: 'opt_in',
    recurring: false,
    seriesId: null,
    status: 'active',
    summary: { yes: 0, no: 0, maybe: 0, pending: 0 } as never,
    myStatus: 'pending',
    myAuto: false,
    myReason: '',
    ...overrides,
  };
}

describe('useEventsQuery', () => {
  it('is disabled (does not fetch) while there is no active team', () => {
    const api = { events: { list: vi.fn() } };
    renderHook(() => useEventsQuery(api as never, null), { wrapper: createQueryWrapper() });
    expect(api.events.list).not.toHaveBeenCalled();
  });

  it('fetches the team-scoped event list once a team id is provided', async () => {
    const api = { events: { list: vi.fn().mockResolvedValue([makeEvent()]) } };
    const { result } = renderHook(() => useEventsQuery(api as never, 'team1'), { wrapper: createQueryWrapper() });
    await waitFor(() => expect(result.current.data).toHaveLength(1));
    expect(api.events.list).toHaveBeenCalledWith('team1', 'all');
  });

  // This is the scenario the pre-migration `refreshEventsSeq`/`activeTeamId`
  // guards existed to defend against: a slow response for a previously-
  // selected team must not overwrite the newly-selected team's data. The
  // team-scoped query key makes it structurally impossible instead of
  // needing a manual sequence ref.
  it('discards a stale response for a previous team after switching teams', async () => {
    let resolveTeamA!: (v: TeamEvent[]) => void;
    const teamAPromise = new Promise<TeamEvent[]>((resolve) => (resolveTeamA = resolve));
    const api = {
      events: {
        list: vi.fn((teamId: string) =>
          teamId === 'teamA' ? teamAPromise : Promise.resolve([makeEvent({ id: 'ev-b', teamId: 'teamB' })]),
        ),
      },
    };
    const client = createTestQueryClient();
    const { result, rerender } = renderHook(({ teamId }) => useEventsQuery(api as never, teamId), {
      wrapper: createQueryWrapper(client),
      initialProps: { teamId: 'teamA' },
    });

    // User switches to teamB before teamA's request resolves.
    rerender({ teamId: 'teamB' });
    await waitFor(() => expect(result.current.data?.[0]?.id).toBe('ev-b'));

    // teamA's stale response now arrives -- it must not overwrite teamB's data.
    resolveTeamA([makeEvent({ id: 'ev-a-stale', teamId: 'teamA' })]);
    await Promise.resolve();

    expect(result.current.data?.[0]?.id).toBe('ev-b');
  });
});

describe('useEventDetailQuery', () => {
  it('is disabled while eventId is not set', () => {
    const api = { events: { get: vi.fn(), listComments: vi.fn() }, attendance: { listForEvent: vi.fn() } };
    renderHook(() => useEventDetailQuery(api as never, 'team1', null), { wrapper: createQueryWrapper() });
    expect(api.events.get).not.toHaveBeenCalled();
  });

  it('fetches event, attendance rows and comments together', async () => {
    const event = makeEvent();
    const api = {
      events: { get: vi.fn().mockResolvedValue(event), listComments: vi.fn().mockResolvedValue([]) },
      attendance: { listForEvent: vi.fn().mockResolvedValue([]) },
    };
    const { result } = renderHook(() => useEventDetailQuery(api as never, 'team1', 'ev1'), {
      wrapper: createQueryWrapper(),
    });
    await waitFor(() => expect(result.current.data?.event).toEqual(event));
    expect(api.events.get).toHaveBeenCalledWith('ev1', 'team1');
    expect(api.attendance.listForEvent).toHaveBeenCalledWith('ev1', 'team1');
    expect(api.events.listComments).toHaveBeenCalledWith('ev1', 'team1');
  });

  it('resolves event: null for a confirmed-missing event, distinct from still-loading', async () => {
    const api = {
      events: { get: vi.fn().mockResolvedValue(null), listComments: vi.fn().mockResolvedValue([]) },
      attendance: { listForEvent: vi.fn().mockResolvedValue([]) },
    };
    const { result } = renderHook(() => useEventDetailQuery(api as never, 'team1', 'ev1'), {
      wrapper: createQueryWrapper(),
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data?.event).toBeNull();
  });
});
