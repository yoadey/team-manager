import { describe, it, expect } from 'vitest';
import { eventFormSchema } from './eventFormSchema';

function baseValues(overrides: Record<string, unknown> = {}) {
  return {
    type: 'training',
    title: 'Training',
    date: '2026-08-14',
    multiDayEndDate: '',
    meetT: '',
    startT: '',
    endT: '',
    location: '',
    note: '',
    meetTimeMandatory: false,
    responseMode: 'opt_out',
    nominatedRoleIds: [],
    recurring: false,
    repeatWeeks: 8,
    repeatMode: 'weeks',
    repeatEndDate: '',
    cancelLeadHours: 0,
    cancelLeadMinutes: 0,
    seriesId: null,
    ...overrides,
  };
}

describe('eventFormSchema multiDayEndDate validation', () => {
  it('accepts an empty multiDayEndDate', () => {
    expect(eventFormSchema.safeParse(baseValues()).success).toBe(true);
  });

  it('accepts a valid multi-day span', () => {
    const result = eventFormSchema.safeParse(baseValues({ multiDayEndDate: '2026-08-16' }));
    expect(result.success).toBe(true);
  });

  it('rejects a malformed multiDayEndDate', () => {
    const result = eventFormSchema.safeParse(baseValues({ multiDayEndDate: 'not-a-date' }));
    expect(result.success).toBe(false);
  });

  it('rejects multiDayEndDate before date', () => {
    const result = eventFormSchema.safeParse(baseValues({ multiDayEndDate: '2026-08-10' }));
    expect(result.success).toBe(false);
  });

  it('rejects multiDayEndDate combined with recurring', () => {
    const result = eventFormSchema.safeParse(
      baseValues({ multiDayEndDate: '2026-08-16', recurring: true, repeatWeeks: 8 }),
    );
    expect(result.success).toBe(false);
  });

  // Regression test: the client-side span validation used to stop at
  // "before start", letting a typo'd year (e.g. 2029 instead of 2026)
  // round-trip to the server and come back as a raw, untranslated 400
  // instead of an inline field error.
  it('rejects a span longer than 1095 days', () => {
    const result = eventFormSchema.safeParse(baseValues({ multiDayEndDate: '2030-08-14' }));
    expect(result.success).toBe(false);
  });

  it('accepts a span of exactly 1095 days', () => {
    const result = eventFormSchema.safeParse(baseValues({ multiDayEndDate: '2029-08-13' }));
    expect(result.success).toBe(true);
  });
});
