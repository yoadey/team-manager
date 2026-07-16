import { describe, it, expect, vi } from 'vitest';
import { renderHook, waitFor } from '@testing-library/react';
import { useMembersQuery } from './useMemberQueries';
import { createQueryWrapper, createTestQueryClient } from '@/test/queryTestUtils';
import type { Member } from '../types';

function makeMember(overrides: Partial<Member> = {}): Member {
  return {
    membershipId: 'ms1',
    userId: 'u1',
    name: 'Alice',
    email: 'alice@test.com',
    phone: '',
    birthday: '',
    address: '',
    avatarColor: '#aaa',
    photo: null,
    group: '',
    roles: [],
    joinedAt: '2026-01-01',
    primaryRole: null,
    perms: {} as never,
    ...overrides,
  };
}

describe('useMembersQuery', () => {
  it('is disabled (does not fetch) while there is no active team', () => {
    const api = { members: { list: vi.fn() } };
    renderHook(() => useMembersQuery(api as never, null), { wrapper: createQueryWrapper() });
    expect(api.members.list).not.toHaveBeenCalled();
  });

  it('fetches the team-scoped member list once a team id is provided', async () => {
    const api = { members: { list: vi.fn().mockResolvedValue([makeMember()]) } };
    const { result } = renderHook(() => useMembersQuery(api as never, 'team1'), { wrapper: createQueryWrapper() });
    await waitFor(() => expect(result.current.data).toHaveLength(1));
    expect(api.members.list).toHaveBeenCalledWith('team1');
  });

  // This is the scenario the pre-migration `refreshMembersSeq`/`activeTeamId`
  // guards existed to defend against: a slow response for a previously-
  // selected team must not overwrite the newly-selected team's data. The
  // team-scoped query key makes it structurally impossible instead of
  // needing a manual sequence ref.
  it('discards a stale response for a previous team after switching teams', async () => {
    let resolveTeamA!: (v: Member[]) => void;
    const teamAPromise = new Promise<Member[]>((resolve) => (resolveTeamA = resolve));
    const api = {
      members: {
        list: vi.fn((teamId: string) =>
          teamId === 'teamA' ? teamAPromise : Promise.resolve([makeMember({ membershipId: 'ms-b' })]),
        ),
      },
    };
    const client = createTestQueryClient();
    const { result, rerender } = renderHook(({ teamId }) => useMembersQuery(api as never, teamId), {
      wrapper: createQueryWrapper(client),
      initialProps: { teamId: 'teamA' },
    });

    // User switches to teamB before teamA's request resolves.
    rerender({ teamId: 'teamB' });
    await waitFor(() => expect(result.current.data?.[0]?.membershipId).toBe('ms-b'));

    // teamA's stale response now arrives -- it must not overwrite teamB's data.
    resolveTeamA([makeMember({ membershipId: 'ms-a-stale' })]);
    await Promise.resolve();

    expect(result.current.data?.[0]?.membershipId).toBe('ms-b');
  });
});
