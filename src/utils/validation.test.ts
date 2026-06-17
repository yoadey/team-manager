import { describe, expect, it } from 'vitest';
import {
  validateDateRange,
  validateEventForm,
  validateMoneyAmount,
  validatePollForm,
  validateRequiredText,
} from './validation';

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

  it('uses the provided field name in error messages', () => {
    const result = validateMoneyAmount('', { field: 'Strafbetrag' });
    expect(result.message).toBe('Strafbetrag fehlt.');
  });

  it('rejects negative values by default but allows zero', () => {
    expect(validateMoneyAmount('-1').ok).toBe(false);
    expect(validateMoneyAmount('0')).toEqual({ ok: true, value: 0 });
  });

  it('rejects zero when allowZero is explicitly false', () => {
    expect(validateMoneyAmount('0', { allowZero: false }).ok).toBe(false);
  });
});

describe('validateDateRange', () => {
  it('accepts a valid ordered range', () => {
    expect(validateDateRange('2024-06-01', '2024-06-10')).toEqual({
      ok: true,
      value: { from: '2024-06-01', to: '2024-06-10' },
    });
  });

  it('accepts a single-day range', () => {
    expect(validateDateRange('2024-06-01', '2024-06-01').ok).toBe(true);
  });

  it('rejects a missing start or end date', () => {
    expect(validateDateRange('', '2024-06-10').message).toContain('Startdatum fehlt');
    expect(validateDateRange('2024-06-01', '').message).toContain('Enddatum fehlt');
  });

  it('rejects calendar-invalid dates', () => {
    // 2023 is not a leap year, so the 29th of February cannot exist.
    expect(validateDateRange('2023-02-29', '2023-03-01').ok).toBe(false);
  });

  it('rejects an end date before the start date', () => {
    const result = validateDateRange('2024-06-10', '2024-06-01');
    expect(result.ok).toBe(false);
    expect(result.message).toContain('nicht vor dem Startdatum');
  });
});

describe('validateEventForm', () => {
  const baseValidForm = {
    title: 'Training',
    date: '2024-06-15',
    meetT: '19:15',
    startT: '19:30',
    endT: '21:30',
    recurring: false,
    repeatWeeks: 8,
  };

  it('accepts a fully valid form', () => {
    expect(validateEventForm(baseValidForm)).toEqual({ ok: true, value: { repeatWeeks: 8 } });
  });

  it('requires a title', () => {
    expect(validateEventForm({ ...baseValidForm, title: '  ' }).message).toContain('Titel');
  });

  it('requires a valid date', () => {
    expect(validateEventForm({ ...baseValidForm, date: '' }).message).toContain('Datum');
    expect(validateEventForm({ ...baseValidForm, date: '2024-13-40' }).ok).toBe(false);
  });

  it('rejects an end time that is not after the start time', () => {
    const result = validateEventForm({ ...baseValidForm, startT: '20:00', endT: '20:00' });
    expect(result.ok).toBe(false);
    expect(result.message).toContain('Ende muss nach dem Beginn');
  });

  it('rejects a meeting time after the start time', () => {
    const result = validateEventForm({ ...baseValidForm, meetT: '19:45', startT: '19:30' });
    expect(result.ok).toBe(false);
    expect(result.message).toContain('Treffzeit darf nicht nach dem Beginn');
  });

  it('enforces repeat-week bounds only for recurring create forms', () => {
    expect(validateEventForm({ ...baseValidForm, recurring: true, repeatWeeks: 1 }).ok).toBe(false);
    expect(validateEventForm({ ...baseValidForm, recurring: true, repeatWeeks: 27 }).ok).toBe(false);
    expect(validateEventForm({ ...baseValidForm, recurring: true, repeatWeeks: 8 }).ok).toBe(true);
  });

  it('ignores recurrence validation in edit mode', () => {
    // Edits target a single occurrence, so series bounds must not block saving.
    const result = validateEventForm({ ...baseValidForm, recurring: true, repeatWeeks: 1 }, 'edit');
    expect(result.ok).toBe(true);
  });

  it('allows an event without any time fields', () => {
    const result = validateEventForm({ title: 'Grillen', date: '2024-07-01' });
    expect(result.ok).toBe(true);
  });
});

describe('validatePollForm', () => {
  it('accepts a question with at least two distinct options', () => {
    expect(validatePollForm({ question: 'Farbe?', opt0: 'Rot', opt1: 'Blau' })).toEqual({
      ok: true,
      value: { question: 'Farbe?', options: ['Rot', 'Blau'] },
    });
  });

  it('requires a question', () => {
    expect(validatePollForm({ opt0: 'Rot', opt1: 'Blau' }).message).toContain('Frage');
  });

  it('requires at least two non-empty options', () => {
    expect(validatePollForm({ question: 'Farbe?', opt0: 'Rot', opt1: '  ' }).ok).toBe(false);
  });

  it('rejects duplicate options case-insensitively', () => {
    const result = validatePollForm({ question: 'Farbe?', opt0: 'Rot', opt1: 'rot' });
    expect(result.ok).toBe(false);
    expect(result.message).toContain('doppelt');
  });
});

describe('validateRequiredText', () => {
  it('trims and returns valid text', () => {
    expect(validateRequiredText('  Hallo  ', 'fehlt')).toEqual({ ok: true, value: 'Hallo' });
  });

  it('fails with the supplied message for blank input', () => {
    expect(validateRequiredText('   ', 'Pflichtfeld fehlt')).toEqual({
      ok: false,
      message: 'Pflichtfeld fehlt',
    });
  });
});
