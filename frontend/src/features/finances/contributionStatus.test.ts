import { describe, it, expect } from 'vitest';
import { contributionAmountStatus } from './contributionStatus';

describe('contributionAmountStatus', () => {
  it('is open when nothing has been paid', () => {
    expect(contributionAmountStatus(2500, 0)).toEqual({ status: 'open', displayAmount: 0, excess: 0 });
  });

  it('is partial when paidAmount is between 0 and amount', () => {
    expect(contributionAmountStatus(2500, 1000)).toEqual({ status: 'partial', displayAmount: 1000, excess: 0 });
  });

  it('is paid, not overpaid, at the exact amount boundary', () => {
    expect(contributionAmountStatus(2500, 2500)).toEqual({ status: 'paid', displayAmount: 2500, excess: 0 });
  });

  it('is overpaid when paidAmount exceeds amount, reporting the excess', () => {
    expect(contributionAmountStatus(2500, 3000)).toEqual({ status: 'overpaid', displayAmount: 3000, excess: 500 });
  });
});
