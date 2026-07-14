import { describe, expect, it } from 'vitest';
import {
  combineDateAndTimeLocal,
  formatDateOnly,
  parseDateOnlyLocal,
  todayLocalDate,
  zonedTimeToUtc,
} from './date';

describe('formatDateOnly', () => {
  it('formats using local calendar fields with zero-padding', () => {
    // Month is zero-based in Date, so January === 0 must surface as "01".
    expect(formatDateOnly(new Date(2024, 0, 5))).toBe('2024-01-05');
    expect(formatDateOnly(new Date(2024, 11, 31))).toBe('2024-12-31');
  });

  it('does not shift the day across timezones (uses local, not UTC)', () => {
    // A local-midnight date must never roll back to the previous day.
    const localMidnight = new Date(2024, 5, 15, 0, 0, 0, 0);
    expect(formatDateOnly(localMidnight)).toBe('2024-06-15');
  });
});

describe('parseDateOnlyLocal', () => {
  it('parses YYYY-MM-DD to local midnight', () => {
    const parsed = parseDateOnlyLocal('2024-06-15');
    expect(parsed.getFullYear()).toBe(2024);
    expect(parsed.getMonth()).toBe(5);
    expect(parsed.getDate()).toBe(15);
    expect(parsed.getHours()).toBe(0);
    expect(parsed.getMinutes()).toBe(0);
  });

  it('round-trips with formatDateOnly without drift', () => {
    const date = '2023-02-28';
    expect(formatDateOnly(parseDateOnlyLocal(date))).toBe(date);
  });

  it('falls back to the native Date parser for non YYYY-MM-DD input', () => {
    // ISO timestamps are not date-only, so the regex must miss and delegate.
    const parsed = parseDateOnlyLocal('2024-06-15T10:30:00Z');
    expect(Number.isNaN(parsed.getTime())).toBe(false);
  });

  it('produces an Invalid Date for unparseable input', () => {
    expect(Number.isNaN(parseDateOnlyLocal('not-a-date').getTime())).toBe(true);
  });
});

describe('combineDateAndTimeLocal', () => {
  it('combines a date and HH:mm into a local Date', () => {
    const combined = combineDateAndTimeLocal('2024-06-15', '19:30');
    expect(combined.getFullYear()).toBe(2024);
    expect(combined.getMonth()).toBe(5);
    expect(combined.getDate()).toBe(15);
    expect(combined.getHours()).toBe(19);
    expect(combined.getMinutes()).toBe(30);
  });

  it('returns local midnight when the time component is malformed', () => {
    const combined = combineDateAndTimeLocal('2024-06-15', 'invalid');
    expect(combined.getHours()).toBe(0);
    expect(combined.getMinutes()).toBe(0);
    expect(combined.getDate()).toBe(15);
  });
});

describe('zonedTimeToUtc', () => {
  // Assertions are on absolute UTC instants, so these pass regardless of the
  // test runner's own local timezone -- unlike combineDateAndTimeLocal,
  // zonedTimeToUtc's whole point is to be independent of it.
  it('converts a summer (CEST, UTC+2) Berlin wall-clock time to the correct UTC instant', () => {
    const utc = zonedTimeToUtc('2024-06-15', '19:30', 'Europe/Berlin');
    expect(utc.toISOString()).toBe('2024-06-15T17:30:00.000Z');
  });

  it('converts a winter (CET, UTC+1) Berlin wall-clock time to the correct UTC instant', () => {
    const utc = zonedTimeToUtc('2024-01-15', '19:30', 'Europe/Berlin');
    expect(utc.toISOString()).toBe('2024-01-15T18:30:00.000Z');
  });

  it('defaults to midnight when the time component is malformed', () => {
    const utc = zonedTimeToUtc('2024-06-15', 'invalid', 'Europe/Berlin');
    expect(utc.toISOString()).toBe('2024-06-14T22:00:00.000Z');
  });

  it('supports a non-Berlin timeZone argument (New York, EDT UTC-4)', () => {
    const utc = zonedTimeToUtc('2024-06-15', '19:30', 'America/New_York');
    expect(utc.toISOString()).toBe('2024-06-15T23:30:00.000Z');
  });
});

describe('todayLocalDate', () => {
  it('matches formatDateOnly of the current date', () => {
    expect(todayLocalDate()).toBe(formatDateOnly(new Date()));
  });

  it('returns a valid YYYY-MM-DD string', () => {
    expect(todayLocalDate()).toMatch(/^\d{4}-\d{2}-\d{2}$/);
  });
});
