import { describe, it, expect } from 'vitest';
import {
  centsToEuros,
  eurosToCents,
  mapTransaction,
  mapPenalty,
  mapPenaltyAssignment,
  mapContribution,
  mapFinanceOverview,
  mapUser,
  mapTeam,
  mapMemberStat,
  mapEventStat,
  mapStatsOverview,
  mapPoll,
  mapTeamEvent,
} from './map';

describe('centsToEuros / eurosToCents', () => {
  it('converts cents to euros', () => {
    expect(centsToEuros(1050)).toBe(10.5);
    expect(centsToEuros(0)).toBe(0);
    expect(centsToEuros(1)).toBeCloseTo(0.01);
  });

  it('converts euros to cents, rounding to the nearest cent', () => {
    expect(eurosToCents(10.5)).toBe(1050);
    expect(eurosToCents(0)).toBe(0);
    expect(eurosToCents(0.1 + 0.2)).toBe(30); // classic float sum (0.30000000000000004) rounds cleanly
  });

  it('round-trips without drift for typical amounts', () => {
    for (const euros of [0, 0.01, 1, 9.99, 42.5, 1234.56]) {
      expect(centsToEuros(eurosToCents(euros))).toBeCloseTo(euros, 9);
    }
  });
});

describe('finance mappers convert amount fields from cents to euros', () => {
  it('mapTransaction', () => {
    const t = mapTransaction({
      id: 'tx1',
      teamId: 'team1',
      type: 'expense',
      title: 'Balls',
      amount: 4250,
      date: '2025-01-01',
    });
    expect(t.amount).toBe(42.5);
  });

  it('mapPenalty', () => {
    const p = mapPenalty({ id: 'p1', teamId: 'team1', label: 'Late', amount: 500 });
    expect(p.amount).toBe(5);
  });

  it('mapPenaltyAssignment preserves undefined amount', () => {
    const a = mapPenaltyAssignment({
      id: 'a1',
      teamId: 'team1',
      userId: 'u1',
      penaltyId: 'p1',
      paid: false,
      date: '2025-01-01',
    });
    expect(a.amount).toBeUndefined();
  });

  it('mapPenaltyAssignment converts a present amount', () => {
    const a = mapPenaltyAssignment({
      id: 'a1',
      teamId: 'team1',
      userId: 'u1',
      penaltyId: 'p1',
      paid: false,
      date: '2025-01-01',
      amount: 1500,
    });
    expect(a.amount).toBe(15);
  });

  it('mapContribution', () => {
    const c = mapContribution({
      id: 'c1',
      teamId: 'team1',
      userId: 'u1',
      month: '2025-01',
      amount: 2500,
      status: 'open',
    });
    expect(c.amount).toBe(25);
  });

  it('mapFinanceOverview converts balance/income/expense/openPenaltySum and nested amounts', () => {
    const o = mapFinanceOverview({
      balance: 30000,
      income: 50000,
      expense: 20000,
      transactions: [],
      penalties: [],
      assignments: [],
      openPenalties: [{ userId: 'u1', name: 'Alice', avatarColor: '#fff', amount: 1500 }],
      openPenaltySum: 1500,
      contributions: [],
      contribOpen: 0,
    });
    expect(o.balance).toBe(300);
    expect(o.income).toBe(500);
    expect(o.expense).toBe(200);
    expect(o.openPenaltySum).toBe(15);
    expect(o.openPenalties[0]!.amount).toBe(15);
  });
});

describe('mapUser / mapTeam resolve hasPhoto/hasLogo to a display URL', () => {
  it('mapUser returns null photo when hasPhoto is false or absent', () => {
    const base = { id: 'u1', name: 'Alice', email: 'a@x.com', avatarColor: '#000' };
    expect(mapUser({ ...base, hasPhoto: false }).photo).toBeNull();
    expect(mapUser(base).photo).toBeNull();
  });

  it('mapUser builds a /auth/me/photo URL when hasPhoto is true', () => {
    const u = mapUser({ id: 'u1', name: 'Alice', email: 'a@x.com', avatarColor: '#000', hasPhoto: true });
    expect(u.photo).toMatch(/^.*\/api\/v1\/auth\/me\/photo\?v=\d+$/);
  });

  it('mapTeam returns null photo/logo when hasPhoto/hasLogo are false or absent', () => {
    const base = { id: 't1', name: 'Team' };
    const t = mapTeam(base);
    expect(t.photo).toBeNull();
    expect(t.logo).toBeNull();
  });

  it('mapTeam builds per-team photo/logo URLs when hasPhoto/hasLogo are true', () => {
    const t = mapTeam({ id: 't1', name: 'Team', hasPhoto: true, hasLogo: true });
    expect(t.photo).toMatch(new RegExp(`/api/v1/teams/t1/photo\\?v=\\d+$`));
    expect(t.logo).toMatch(new RegExp(`/api/v1/teams/t1/logo\\?v=\\d+$`));
  });
});

// The backend's quote/pct/avg attendance fields are 0-1 fractions
// (internal/stats/service.go's quote()); every UI consumer (Stats.tsx,
// MemberSheets.tsx) renders them as a 0-100 percentage. Regression coverage
// for a bug where the real service layer passed these through unscaled,
// making e.g. a 50% attendance rate render as "0.5%" and always fall into
// the worst color bucket.
describe('stats mappers convert 0-1 fractions to 0-100 percentages', () => {
  it('mapMemberStat scales quote', () => {
    expect(mapMemberStat({ userId: 'u1', name: 'Alice', avatarColor: '#000', quote: 0.5, counted: 4, yes: 2 }).quote).toBe(50);
    expect(mapMemberStat({ userId: 'u1', name: 'Alice', avatarColor: '#000', quote: 1, counted: 4, yes: 4 }).quote).toBe(100);
  });

  it('mapMemberStat maps counted:0 to quote:null ("no data"), not 0%', () => {
    // A member with 0 counted events attended none of zero, not 0% of some —
    // Stats.tsx renders these as distinct states (gray "–" vs. red "0%").
    expect(mapMemberStat({ userId: 'u1', name: 'Alice', avatarColor: '#000', quote: 0, counted: 0, yes: 0 }).quote).toBeNull();
  });

  it('mapEventStat scales pct and passes enough through unchanged', () => {
    const s = mapEventStat({
      id: 'e1',
      title: 'Training',
      type: 'training',
      date: '2024-06-01',
      yes: 3,
      nominated: 4,
      pct: 0.75,
      enough: true,
    });
    expect(s.pct).toBe(75);
    expect(s.enough).toBe(true);
  });

  it('mapStatsOverview scales avg and nested member/event fractions', () => {
    const o = mapStatsOverview({
      avg: 0.667,
      members: [{ userId: 'u1', name: 'Alice', avatarColor: '#000', quote: 0.5, counted: 4, yes: 2 }],
      events: [
        { id: 'e1', title: 'Training', type: 'training', date: '2024-06-01', yes: 3, nominated: 4, pct: 0.75, enough: true },
      ],
      pastCount: 1,
      from: '2024-01-01',
      to: '2024-06-30',
    });
    expect(o.avg).toBe(67);
    expect(o.members[0]!.quote).toBe(50);
    expect(o.events[0]!.pct).toBe(75);
  });
});

// PollsPage.tsx computes `voted = !!p.myVote`, and !![] is true in JS —
// coalescing the backend's null "not voted" sentinel to [] made every unvoted
// poll render as if the user had already voted.
describe('mapPoll preserves myVote:null as the "not voted" sentinel', () => {
  const base = {
    id: 'p1',
    question: 'Q?',
    multiple: false,
    anonymous: false,
    createdAt: '2024-01-01',
    totalVotes: 0,
    options: [],
  };

  it('maps a missing/null myVote to null, not []', () => {
    // The generated type declares myVote as string[] | undefined (no
    // `nullable: true` in the OpenAPI schema), but the real backend's JSON
    // can genuinely be null (a nil Go slice) — hence the cast to simulate
    // what actually arrives over the wire.
    expect(mapPoll({ ...base, myVote: null } as unknown as Parameters<typeof mapPoll>[0]).myVote).toBeNull();
  });

  it('maps a real vote array through unchanged', () => {
    expect(mapPoll({ ...base, myVote: ['opt1'] }).myVote).toEqual(['opt1']);
  });
});

// events.Service.{ListEvents,GetEvent} populate TeamEvent.myReason from the
// caller's own attendance row (internal/events/service.go). mapTeamEvent used
// to always return '', silently discarding it against the real backend —
// useEventActions.ts's setMyStatus() reads `ev.myReason` to preserve the
// user's existing reason across a quick yes/no/maybe tap, so this erased
// saved attendance reasons every time, and EventDetailSheet.tsx's
// comment-reason indicator always rendered as "no reason given".
describe('mapTeamEvent preserves myReason from the backend', () => {
  const baseEvent = {
    id: 'e1',
    teamId: 't1',
    type: 'training',
    title: 'Training',
    date: '2024-01-01',
    recurring: false,
    status: 'active',
    summary: { yes: 0, no: 0, maybe: 0, pending: 0, notNominated: 0, nominated: 0, total: 0 },
  };

  it('passes a present myReason through unchanged', () => {
    expect(
      mapTeamEvent({ ...baseEvent, myReason: 'Grippe, kuriere mich aus' } as unknown as Parameters<
        typeof mapTeamEvent
      >[0]).myReason,
    ).toBe('Grippe, kuriere mich aus');
  });

  it('maps a missing myReason to the empty-string default', () => {
    expect(mapTeamEvent(baseEvent as unknown as Parameters<typeof mapTeamEvent>[0]).myReason).toBe('');
  });
});
