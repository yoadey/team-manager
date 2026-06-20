// =============================================================================
// Lightweight i18n layer (no external dependency).
//
// Goals addressed here:
//  - A single source of truth for the active locale (instead of `de-DE` hard
//    coded in every Intl call across the codebase).
//  - A `t()` lookup with parameter interpolation backed by message catalogs.
//  - Locale-aware number/currency formatting through `getIntlLocale()` /
//    `getCurrency()`, consumed by the formatting helpers in styles/tokens.ts.
//
// The catalog currently ships German (the product's primary language) plus an
// English skeleton so additional languages can be added without touching call
// sites. UI strings can be migrated onto `t()` incrementally.
// =============================================================================

import { de } from './de';
import { en } from './en';

export type Locale = 'de' | 'en';
export type Messages = typeof de;

interface LocaleConfig {
  /** BCP-47 tag handed to the `Intl.*` constructors. */
  intl: string;
  /** ISO 4217 currency used by money formatting. */
  currency: string;
  messages: Messages;
}

const LOCALES: Record<Locale, LocaleConfig> = {
  de: { intl: 'de-DE', currency: 'EUR', messages: de },
  en: { intl: 'en-US', currency: 'EUR', messages: en },
};

export const SUPPORTED_LOCALES: Locale[] = ['de', 'en'];
export const DEFAULT_LOCALE: Locale = 'de';

let activeLocale: Locale = DEFAULT_LOCALE;
const listeners = new Set<() => void>();

export function getLocale(): Locale {
  return activeLocale;
}

export function getIntlLocale(): string {
  return LOCALES[activeLocale].intl;
}

export function getCurrency(): string {
  return LOCALES[activeLocale].currency;
}

export function setLocale(locale: Locale): void {
  if (locale === activeLocale || !LOCALES[locale]) return;
  activeLocale = locale;
  listeners.forEach((fn) => fn());
}

/** Subscribe to locale changes (used by the React provider to re-render). */
export function subscribeLocale(fn: () => void): () => void {
  listeners.add(fn);
  return () => {
    listeners.delete(fn);
  };
}

type Params = Record<string, string | number>;

function interpolate(template: string, params?: Params): string {
  if (!params) return template;
  return template.replace(/\{(\w+)\}/g, (_, key) => (key in params ? String(params[key]) : `{${key}}`));
}

function lookupKey(messages: Messages, key: string): string | undefined {
  return key
    .split('.')
    .reduce<unknown>(
      (acc, part) => (acc && typeof acc === 'object' ? (acc as Record<string, unknown>)[part] : undefined),
      messages,
    ) as string | undefined;
}

const pluralRulesCache: Partial<Record<string, Intl.PluralRules>> = {};
function selectPluralForm(count: number, intlLocale: string): Intl.LDMLPluralRule {
  pluralRulesCache[intlLocale] ??= new Intl.PluralRules(intlLocale);
  return pluralRulesCache[intlLocale]!.select(count);
}

/**
 * Translate a dotted key, falling back to the default locale and finally to the
 * key itself so a missing string is visible rather than throwing.
 *
 * Pass `count` in params to activate plural form selection. The function looks
 * up `<key>_<form>` (e.g. `members.count_one`, `members.count_other`) before
 * falling back to the base key. Add plural variants to the message catalog with
 * the `_one` / `_other` (etc.) suffix using standard CLDR plural categories.
 */
export function t(key: string, params?: Params): string {
  const count = params?.count;

  let value: string | undefined;

  if (count !== undefined) {
    const form = selectPluralForm(Number(count), LOCALES[activeLocale].intl);
    const pluralKey = `${key}_${form}`;
    value =
      lookupKey(LOCALES[activeLocale].messages, pluralKey) ?? lookupKey(LOCALES[DEFAULT_LOCALE].messages, pluralKey);
  }

  if (value === undefined) {
    value = lookupKey(LOCALES[activeLocale].messages, key) ?? lookupKey(LOCALES[DEFAULT_LOCALE].messages, key);
  }

  return typeof value === 'string' ? interpolate(value, params) : key;
}
