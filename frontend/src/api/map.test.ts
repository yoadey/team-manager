import { describe, it, expect } from 'vitest';
import {
  centsToEuros,
  eurosToCents,
  mapTransaction,
  mapPenalty,
  mapPenaltyAssignment,
  mapContribution,
  mapFinanceOverview,
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
    expect(o.openPenalties[0].amount).toBe(15);
  });
});
