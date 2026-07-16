import { describe, it, expect, vi } from 'vitest';
import { renderHook, waitFor } from '@testing-library/react';
import { useAbsencesQuery } from './useAbsenceQueries';
import { createQueryWrapper, createTestQueryClient } from '@/test/queryTestUtils';
import type { Absence } from '../types';

function makeAbsence(overrides: Partial<Absence> = {}): Absence {
  return {
    id: 'abs1',
    userId: 'u1',
    from: '2026-01-01',
    to: '2026-01-05',
    reason: 'Urlaub',
    createdAt: '2026-01-01',
    ...overrides,
  };
}

describe('useAbsencesQuery', () => {
  it('is disabled (does not fetch) while there is no active team', () => {
    const api = { absences: { listForTeam: vi.fn() } };
    renderHook(() => useAbsencesQuery(api as never, null), { wrapper: createQueryWrapper() });
    expect(api.absences.listForTeam).not.toHaveBeenCalled();
  });

  it("is disabled (does not fetch) while the caller's own enabled flag is false", () => {
    const api = { absences: { listForTeam: vi.fn() } };
    renderHook(() => useAbsencesQuery(api as never, 'team1', false), { wrapper: createQueryWrapper() });
    expect(api.absences.listForTeam).not.toHaveBeenCalled();
  });

  it('fetches the team-scoped absence list once a team id is provided', async () => {
    const api = { absences: { listForTeam: vi.fn().mockResolvedValue([makeAbsence()]) } };
    const { result } = renderHook(() => useAbsencesQuery(api as never, 'team1'), { wrapper: createQueryWrapper() });
    await waitFor(() => expect(result.current.data).toHaveLength(1));
    expect(api.absences.listForTeam).toHaveBeenCalledWith('team1');
  });

  // This is the scenario the pre-migration `loadAbsencesSeq`/`activeTeamId`
  // guards existed to defend against: a slow response for a previously-
  // selected team must not overwrite the newly-selected team's data. The
  // team-scoped query key makes it structurally impossible instead of
  // needing a manual sequence ref.
  it('discards a stale response for a previous team after switching teams', async () => {
    let resolveTeamA!: (v: Absence[]) => void;
    const teamAPromise = new Promise<Absence[]>((resolve) => (resolveTeamA = resolve));
    const api = {
      absences: {
        listForTeam: vi.fn((teamId: string) =>
          teamId === 'teamA' ? teamAPromise : Promise.resolve([makeAbsence({ id: 'abs-b' })]),
        ),
      },
    };
    const client = createTestQueryClient();
    const { result, rerender } = renderHook(({ teamId }) => useAbsencesQuery(api as never, teamId), {
      wrapper: createQueryWrapper(client),
      initialProps: { teamId: 'teamA' },
    });

    // User switches to teamB before teamA's request resolves.
    rerender({ teamId: 'teamB' });
    await waitFor(() => expect(result.current.data?.[0]?.id).toBe('abs-b'));

    // teamA's stale response now arrives -- it must not overwrite teamB's data.
    resolveTeamA([makeAbsence({ id: 'abs-a-stale' })]);
    await Promise.resolve();

    expect(result.current.data?.[0]?.id).toBe('abs-b');
  });
});
