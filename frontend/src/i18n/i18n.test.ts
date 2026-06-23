import { describe, it, expect, afterEach } from 'vitest';
import { getCurrency, getIntlLocale, getLocale, setLocale, t } from './index';

afterEach(() => setLocale('de'));

describe('i18n locale', () => {
  it('defaults to German', () => {
    expect(getLocale()).toBe('de');
    expect(getIntlLocale()).toBe('de-DE');
    expect(getCurrency()).toBe('EUR');
  });

  it('switches locale and intl tag together', () => {
    setLocale('en');
    expect(getLocale()).toBe('en');
    expect(getIntlLocale()).toBe('en-US');
  });

  it('ignores unsupported locales', () => {
    // @ts-expect-error invalid locale guarded at runtime
    setLocale('fr');
    expect(getLocale()).toBe('de');
  });

  it('persists the chosen locale to localStorage', () => {
    setLocale('en');
    expect(localStorage.getItem('tv_locale')).toBe('en');
    setLocale('de');
    expect(localStorage.getItem('tv_locale')).toBe('de');
  });
});

describe('t()', () => {
  it('resolves dotted keys', () => {
    expect(t('attendance.yes')).toBe('Zugesagt');
    setLocale('en');
    expect(t('attendance.yes')).toBe('Attending');
  });

  it('interpolates params', () => {
    expect(t('relTime.minutes', { n: 5 })).toBe('vor 5 Min');
  });

  it('returns the key when missing', () => {
    expect(t('does.not.exist')).toBe('does.not.exist');
  });
});
