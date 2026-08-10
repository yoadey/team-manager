import { describe, it, expect } from 'vitest';
import { effectiveRsvpCutoff, isRsvpCutoffPassed, eventEffectiveEndDate, isEventPast } from './rsvpCutoff';

function baseEvent(overrides: Partial<Parameters<typeof effectiveRsvpCutoff>[0]> = {}) {
  return {
    date: '2026-07-15',
    startTime: null,
    meetTime: null,
    cancelLeadMinutes: null,
    ...overrides,
  };
}

describe('effectiveRsvpCutoff', () => {
  it('returns null when no cutoff is set', () => {
    expect(effectiveRsvpCutoff(baseEvent())).toBeNull();
  });

  it('derives the cutoff from cancelLeadMinutes and startTime (Europe/Berlin, CEST = UTC+2)', () => {
    const cutoff = effectiveRsvpCutoff(baseEvent({ startTime: '12:00', cancelLeadMinutes: 60 }));
    // Start: 2026-07-15T10:00:00Z (12:00 CEST). Cutoff: minus 60 minutes.
    expect(cutoff?.toISOString()).toBe('2026-07-15T09:00:00.000Z');
  });

  it('falls back to meetTime when startTime is unset', () => {
    const cutoff = effectiveRsvpCutoff(baseEvent({ meetTime: '11:00', cancelLeadMinutes: 30 }));
    expect(cutoff?.toISOString()).toBe('2026-07-15T08:30:00.000Z');
  });

  it('falls back to 18:00 when neither startTime nor meetTime is set', () => {
    const cutoff = effectiveRsvpCutoff(baseEvent({ cancelLeadMinutes: 0 }));
    expect(cutoff?.toISOString()).toBe('2026-07-15T16:00:00.000Z');
  });

  it('handles winter dates correctly (CET = UTC+1)', () => {
    const cutoff = effectiveRsvpCutoff(baseEvent({ date: '2026-01-15', startTime: '12:00', cancelLeadMinutes: 0 }));
    expect(cutoff?.toISOString()).toBe('2026-01-15T11:00:00.000Z');
  });
});

describe('isRsvpCutoffPassed', () => {
  it('is false when no cutoff is configured', () => {
    expect(isRsvpCutoffPassed(baseEvent())).toBe(false);
  });

  it('is true once now is past the effective cutoff', () => {
    const event = baseEvent({ startTime: '12:00', cancelLeadMinutes: 60 });
    const cutoffMs = new Date('2026-07-15T09:00:00.000Z').getTime();
    expect(isRsvpCutoffPassed(event, cutoffMs - 1000)).toBe(false);
    expect(isRsvpCutoffPassed(event, cutoffMs)).toBe(true);
    expect(isRsvpCutoffPassed(event, cutoffMs + 1000)).toBe(true);
  });
});

describe('eventEffectiveEndDate', () => {
  it('returns date for a single-day event', () => {
    expect(eventEffectiveEndDate({ date: '2026-07-15', multiDayEndDate: null })).toBe('2026-07-15');
  });

  it('returns multiDayEndDate for a multi-day event', () => {
    expect(eventEffectiveEndDate({ date: '2026-07-15', multiDayEndDate: '2026-07-17' })).toBe('2026-07-17');
  });
});

describe('isEventPast', () => {
  it('is true for a single-day event dated before today', () => {
    expect(isEventPast({ date: '2026-07-14', multiDayEndDate: null }, '2026-07-15')).toBe(true);
  });

  it('is false for a single-day event dated today or later', () => {
    expect(isEventPast({ date: '2026-07-15', multiDayEndDate: null }, '2026-07-15')).toBe(false);
  });

  // Regression test: every "is this event past" check across the app used
  // to compare `date` (the start day) alone against today, so a multi-day
  // event that had started but not finished was misclassified as already
  // over -- hiding RSVP controls, dropping it from "upcoming" lists, and
  // dimming its card -- the moment its first day passed, contradicting the
  // backend's own COALESCE(end_date, date) scope semantics.
  it('is false for a multi-day event that has started but not finished', () => {
    expect(isEventPast({ date: '2026-07-14', multiDayEndDate: '2026-07-16' }, '2026-07-15')).toBe(false);
  });

  it('is true for a multi-day event that has already finished', () => {
    expect(isEventPast({ date: '2026-07-10', multiDayEndDate: '2026-07-12' }, '2026-07-15')).toBe(true);
  });
});
