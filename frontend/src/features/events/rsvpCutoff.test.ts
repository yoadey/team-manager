import { describe, it, expect } from 'vitest';
import { effectiveRsvpCutoff, isRsvpCutoffPassed } from './rsvpCutoff';

function baseEvent(overrides: Partial<Parameters<typeof effectiveRsvpCutoff>[0]> = {}) {
  return {
    date: '2026-07-15',
    startTime: null,
    meetTime: null,
    rsvpDeadline: null,
    cancelLeadMinutes: null,
    ...overrides,
  };
}

describe('effectiveRsvpCutoff', () => {
  it('returns null when neither cutoff is set', () => {
    expect(effectiveRsvpCutoff(baseEvent())).toBeNull();
  });

  it('uses rsvpDeadline when only that is set', () => {
    const cutoff = effectiveRsvpCutoff(baseEvent({ rsvpDeadline: '2026-07-14T10:00:00.000Z' }));
    expect(cutoff?.toISOString()).toBe('2026-07-14T10:00:00.000Z');
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

  it('picks the earlier of rsvpDeadline and the cancelLeadMinutes-derived cutoff', () => {
    // cancelLeadMinutes cutoff: 2026-07-15T09:00:00Z; rsvpDeadline is earlier.
    const earlierDeadline = effectiveRsvpCutoff(
      baseEvent({ startTime: '12:00', cancelLeadMinutes: 60, rsvpDeadline: '2026-07-14T00:00:00.000Z' }),
    );
    expect(earlierDeadline?.toISOString()).toBe('2026-07-14T00:00:00.000Z');

    // rsvpDeadline is later than the cancelLeadMinutes cutoff -- the earlier one wins.
    const earlierLeadTime = effectiveRsvpCutoff(
      baseEvent({ startTime: '12:00', cancelLeadMinutes: 60, rsvpDeadline: '2026-07-15T23:00:00.000Z' }),
    );
    expect(earlierLeadTime?.toISOString()).toBe('2026-07-15T09:00:00.000Z');
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
