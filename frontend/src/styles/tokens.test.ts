import { describe, it, expect, vi, afterEach } from 'vitest';
import {
  buildTokens,
  neutralCssVars,
  typeMeta,
  statusMeta,
  hhmm,
  fmtRange,
  fmtDate,
  fmtMoney,
  monthName,
  initials,
  relTime,
  fmtDateTime,
  NEUTRAL,
  DEFAULT_PRESET_KEY,
} from './tokens';

describe('buildTokens', () => {
  it('returns token object with primary color', () => {
    const tokens = buildTokens('#4285F4');
    expect(tokens.primary).toBeTruthy();
  });

  it('uses default preset when key not found', () => {
    const tokens = buildTokens('invalid-key');
    const defaultTokens = buildTokens(DEFAULT_PRESET_KEY);
    expect(tokens.primary).toBe(defaultTokens.primary);
  });

  it('returns different tokens for different presets', () => {
    const blue = buildTokens('#1565C0');
    const green = buildTokens('#33691E');
    expect(blue.primary).not.toBe(green.primary);
  });
});

describe('neutralCssVars', () => {
  it('returns light CSS vars for dark=false', () => {
    const vars = neutralCssVars(false);
    expect(vars['--tv-neutral-appBg']).toBe('#E4E5EC');
  });

  it('returns dark CSS vars for dark=true', () => {
    const vars = neutralCssVars(true);
    expect(vars['--tv-neutral-appBg']).toBe('#111318');
  });

  it('has all expected neutral keys', () => {
    const vars = neutralCssVars(false);
    expect(vars['--tv-neutral-surface']).toBeTruthy();
    expect(vars['--tv-neutral-card']).toBeTruthy();
    expect(vars['--tv-neutral-error']).toBeTruthy();
  });
});

describe('NEUTRAL', () => {
  it('values are CSS custom property references', () => {
    expect(NEUTRAL.surface).toBe('var(--tv-neutral-surface)');
    expect(NEUTRAL.error).toBe('var(--tv-neutral-error)');
    expect(NEUTRAL.line).toBe('var(--tv-neutral-line)');
  });
});

describe('typeMeta', () => {
  it('returns training metadata', () => {
    const meta = typeMeta('training');
    expect(meta.icon).toBe('fitness_center');
    expect(meta.label).toBeTruthy();
  });

  it('returns auftritt metadata', () => {
    const meta = typeMeta('auftritt');
    expect(meta.icon).toBe('emoji_events');
  });

  it('returns event metadata for "event"', () => {
    const meta = typeMeta('event');
    expect(meta.icon).toBe('celebration');
  });

  it('falls back to event for unknown type', () => {
    const meta = typeMeta('unknown_type');
    expect(meta.icon).toBe('celebration');
  });
});

describe('statusMeta', () => {
  it('returns yes status metadata', () => {
    const meta = statusMeta('yes');
    expect(meta.icon).toBe('check_circle');
  });

  it('returns no status metadata', () => {
    const meta = statusMeta('no');
    expect(meta.icon).toBe('cancel');
  });

  it('returns maybe status metadata', () => {
    const meta = statusMeta('maybe');
    expect(meta.icon).toBe('help');
  });

  it('returns pending metadata for unknown', () => {
    const meta = statusMeta('unknown');
    expect(meta.icon).toBe('schedule');
  });

  it('returns not_nominated metadata', () => {
    const meta = statusMeta('not_nominated');
    expect(meta.icon).toBe('block');
  });
});

describe('hhmm', () => {
  it('returns empty string for null', () => {
    expect(hhmm(null)).toBe('');
  });

  it('returns time string as-is if already HH:MM format', () => {
    expect(hhmm('14:30')).toBe('14:30');
  });

  it('formats ISO timestamp to HH:MM', () => {
    // Create a date with known hours/minutes
    const result = hhmm('2025-06-15T09:05:00.000Z');
    expect(result).toMatch(/^\d{2}:\d{2}$/);
  });
});

describe('monthName', () => {
  it('returns year-month formatted string', () => {
    const result = monthName('2025-06');
    expect(result).toContain('2025');
  });

  it('returns empty string for empty input', () => {
    expect(monthName('')).toBe('');
  });

  it('returns input as-is if not valid year-month', () => {
    expect(monthName('notadate')).toBe('notadate');
  });
});

describe('initials', () => {
  it('returns first letters of first two words', () => {
    expect(initials('Anna Müller')).toBe('AM');
  });

  it('returns single letter for single word', () => {
    expect(initials('Anna')).toBe('A');
  });

  it('returns empty string for empty input', () => {
    expect(initials('')).toBe('');
  });

  it('is uppercase', () => {
    expect(initials('anna müller')).toBe('AM');
  });

  it('handles multiple spaces', () => {
    expect(initials('  Anna   Müller  ')).toBe('AM');
  });
});

describe('fmtRange', () => {
  it('returns a formatted date range string', () => {
    const result = fmtRange('2025-06-01', '2025-06-15');
    expect(result).toContain('–');
    expect(result).toContain('Juni');
  });
});

describe('fmtDateTime', () => {
  it('returns a formatted datetime string', () => {
    const result = fmtDateTime('2025-06-15T14:30:00.000Z');
    expect(typeof result).toBe('string');
    expect(result.length).toBeGreaterThan(0);
  });
});

describe('relTime', () => {
  it('returns "gerade eben" for very recent time', () => {
    const result = relTime(new Date().toISOString());
    expect(result).toBeTruthy();
  });

  it('returns minutes-ago text for recent time', () => {
    const fiveMinAgo = new Date(Date.now() - 5 * 60 * 1000).toISOString();
    const result = relTime(fiveMinAgo);
    expect(result).toContain('Min');
  });

  it('returns hours-ago text for hours-old time', () => {
    const twoHoursAgo = new Date(Date.now() - 2.5 * 60 * 60 * 1000).toISOString();
    const result = relTime(twoHoursAgo);
    expect(result).toBeTruthy();
  });

  it('returns days-ago text for old time', () => {
    const twoDaysAgo = new Date(Date.now() - 2 * 24 * 60 * 60 * 1000).toISOString();
    const result = relTime(twoDaysAgo);
    expect(result).toBeTruthy();
  });
});

describe('formatter caching', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('reuses a single Intl.DateTimeFormat instance across repeated fmtDate calls with the same options', () => {
    const OriginalDateTimeFormat = Intl.DateTimeFormat;
    const spy = vi.spyOn(Intl, 'DateTimeFormat').mockImplementation(function (
      ...args: ConstructorParameters<typeof Intl.DateTimeFormat>
    ) {
      return new OriginalDateTimeFormat(...args);
    });
    fmtDate('2025-06-01');
    fmtDate('2025-06-02');
    fmtDate('2025-06-03');
    expect(spy).toHaveBeenCalledTimes(1);
  });

  it('reuses a single Intl.NumberFormat instance across repeated fmtMoney calls', () => {
    const OriginalNumberFormat = Intl.NumberFormat;
    const spy = vi.spyOn(Intl, 'NumberFormat').mockImplementation(function (
      ...args: ConstructorParameters<typeof Intl.NumberFormat>
    ) {
      return new OriginalNumberFormat(...args);
    });
    fmtMoney(1);
    fmtMoney(2);
    fmtMoney(3);
    expect(spy).toHaveBeenCalledTimes(1);
  });
});
