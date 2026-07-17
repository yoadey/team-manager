import { describe, expect, it } from 'vitest';
import { validateMoneyAmount } from './validation';

describe('validateMoneyAmount', () => {
  it('accepts a plain numeric string', () => {
    expect(validateMoneyAmount('12.50')).toEqual({ ok: true, value: 12.5 });
  });

  it('accepts German comma decimals by normalising to a dot', () => {
    expect(validateMoneyAmount('12,99')).toEqual({ ok: true, value: 12.99 });
  });

  it('rounds to two decimal places (currency precision)', () => {
    expect(validateMoneyAmount('1.005')).toEqual({ ok: true, value: 1.01 });
  });

  it('rejects an empty value', () => {
    const result = validateMoneyAmount('   ');
    expect(result.ok).toBe(false);
    expect(result.message).toContain('fehlt');
  });

  it('rejects a non-numeric value', () => {
    const result = validateMoneyAmount('abc');
    expect(result.ok).toBe(false);
    expect(result.message).toContain('gültige Zahl');
  });

  it('rejects non-positive values when positive is required', () => {
    expect(validateMoneyAmount('0', { positive: true }).ok).toBe(false);
    expect(validateMoneyAmount('-5', { positive: true }).ok).toBe(false);
  });

  it('shows the generic missing message when value is empty', () => {
    const result = validateMoneyAmount('');
    expect(result.ok).toBe(false);
    expect(result.message).toContain('fehlt');
  });

  it('rejects negative values by default but allows zero', () => {
    expect(validateMoneyAmount('-1').ok).toBe(false);
    expect(validateMoneyAmount('0')).toEqual({ ok: true, value: 0 });
  });

  it('rejects zero when allowZero is explicitly false', () => {
    expect(validateMoneyAmount('0', { allowZero: false }).ok).toBe(false);
  });

  // Regression test: validateMoneyAmount had no upper bound, unlike the
  // backend's amount cap (100000000 cents / €1,000,000) on
  // CreateTransactionRequest/CreatePenaltyRequest/UpdateContributionRequest,
  // so an accidental extra digit passed client-side validation and only
  // failed with a raw, unlocalized backend error string.
  it('rejects amounts above an explicit max', () => {
    const result = validateMoneyAmount('1000000.01', { positive: true, max: 1000000 });
    expect(result.ok).toBe(false);
  });

  it('accepts an amount exactly at the max', () => {
    expect(validateMoneyAmount('1000000', { positive: true, max: 1000000 })).toEqual({ ok: true, value: 1000000 });
  });

  it('has no upper bound when max is not specified', () => {
    expect(validateMoneyAmount('99999999', { positive: true }).ok).toBe(true);
  });
});
