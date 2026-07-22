// =============================================================================
// Design tokens — the single source of truth for colours, type/status metadata
// and small formatting helpers. Values are taken verbatim from the design
// handoff (section 8, Design-Tokens) so the MUI build matches pixel for pixel.
// =============================================================================

import type { AttendanceStatus, EventType } from '@/types';
import { parseDateOnlyLocal, todayLocalDate } from '@/utils/date';
import { getCurrency, getIntlLocale, t } from '@/i18n';

export interface ThemePreset {
  primary: string;
  onPrimary: string;
  primaryContainer: string;
  onPrimaryContainer: string;
  secondaryContainer: string;
  onSecondaryContainer: string;
}

// Bound directly to a real object literal (not looked up by indexing) so
// it's a ThemePreset, not ThemePreset | undefined -- buildTokens() falls
// back to this when `presetKey` doesn't match any known preset (e.g. a
// team's custom primaryColor), so it must be guaranteed defined independent
// of the THEME_PRESETS lookup that can legitimately miss.
const DEFAULT_THEME_PRESET: ThemePreset = {
  primary: '#1565C0',
  onPrimary: '#FFFFFF',
  primaryContainer: '#D7E3FF',
  onPrimaryContainer: '#001B3E',
  secondaryContainer: '#DCE3F2',
  onSecondaryContainer: '#101C2B',
};

export const THEME_PRESETS: Record<string, ThemePreset> = {
  '#1565C0': DEFAULT_THEME_PRESET,
  '#6750A4': {
    primary: '#6750A4',
    onPrimary: '#FFFFFF',
    primaryContainer: '#EADDFF',
    onPrimaryContainer: '#21005D',
    secondaryContainer: '#E8DEF8',
    onSecondaryContainer: '#1D192B',
  },
  '#00796B': {
    primary: '#00796B',
    onPrimary: '#FFFFFF',
    primaryContainer: '#9DF1E2',
    onPrimaryContainer: '#00201B',
    secondaryContainer: '#CCE8E2',
    onSecondaryContainer: '#0B1F1B',
  },
  '#B71C1C': {
    primary: '#B3261E',
    onPrimary: '#FFFFFF',
    primaryContainer: '#F9DEDC',
    onPrimaryContainer: '#410E0B',
    secondaryContainer: '#F4DDDA',
    onSecondaryContainer: '#2C1512',
  },
  '#33691E': {
    primary: '#386A20',
    onPrimary: '#FFFFFF',
    primaryContainer: '#B7F397',
    onPrimaryContainer: '#072100',
    secondaryContainer: '#D8E7CD',
    onSecondaryContainer: '#121F0C',
  },
};

export const DEFAULT_PRESET_KEY = '#1565C0';

/** Full token object combining a preset with the shared neutral/semantic colours. */
export interface AppTokens extends ThemePreset {
  surface: string;
  onSurface: string;
  onSurfaceVariant: string;
  outline: string;
  success: string;
  error: string;
  warn: string;
}

export function buildTokens(presetKey: string): AppTokens {
  const p = THEME_PRESETS[presetKey] || DEFAULT_THEME_PRESET;
  return {
    ...p,
    surface: '#FBFBFE',
    onSurface: '#1A1C20',
    onSurfaceVariant: '#44474E',
    outline: '#74777F',
    success: '#2E7D32',
    error: '#BA1A1A',
    warn: '#8A6100',
  };
}

// Neutral colour palettes — light and dark.
// All component code references NEUTRAL, which uses CSS custom properties so
// dark-mode switching happens automatically without touching component files.
// Exported (light-mode literal values only) for buildMuiTheme: MUI computes
// hover/disabled/Skeleton overlays by calling alpha()/darken()/lighten() on
// palette.text/background/divider internally, which requires an actual
// parseable color -- NEUTRAL's var(--tv-neutral-*) strings break that math
// (see theme.ts's doc comment). Dark mode itself is handled independently,
// by the CSS custom properties this constant seeds (see neutralCssVars) and
// components referencing NEUTRAL directly, not by MUI's theme.palette.mode
// (createTheme always sets mode: 'light').
export const NEUTRAL_LIGHT = {
  appBg: '#E4E5EC',
  surface: '#FBFBFE',
  card: '#FFFFFF',
  sidebar: '#F4F4FA',
  onSurface: '#1A1C20',
  onSurfaceVariant: '#44474E',
  secondary: '#6A6D76',
  faint: '#767676',
  line: '#E6E7EE',
  line2: '#ECEDF3',
  line3: '#E0E2EA',
  inputBorder: '#C8CAD2',
  error: '#BA1A1A',
  errorBg: '#FFDAD6',
  success: '#2E7D32',
  successBg: '#D7F0D8',
  warn: '#8A6100',
  warnBg: '#FFE7B0',
};

const NEUTRAL_DARK = {
  appBg: '#111318',
  surface: '#1A1C20',
  card: '#22242A',
  sidebar: '#1E2026',
  onSurface: '#E3E2E6',
  onSurfaceVariant: '#C4C6CF',
  secondary: '#8E9099',
  faint: '#8E8E8E',
  line: '#2E3038',
  line2: '#292A31',
  line3: '#2A2C33',
  inputBorder: '#44474F',
  error: '#FFB4AB',
  errorBg: '#93000A',
  success: '#7BDA7B',
  successBg: '#1A3C1A',
  warn: '#F2C06B',
  warnBg: '#3A2A00',
};

/** Build the CSS custom-property map for a given colour scheme. */
export function neutralCssVars(dark: boolean): Record<string, string> {
  const src = dark ? NEUTRAL_DARK : NEUTRAL_LIGHT;
  return Object.fromEntries(Object.entries(src).map(([k, v]) => [`--tv-neutral-${k}`, v]));
}

/**
 * Neutral / line colours used across cards & surfaces.
 * Values are CSS custom-property references so that toggling
 * `data-color-scheme="dark"` on `<html>` instantly switches the whole UI.
 */
export const NEUTRAL: Record<keyof typeof NEUTRAL_LIGHT, string> = Object.fromEntries(
  Object.keys(NEUTRAL_LIGHT).map((k) => [k, `var(--tv-neutral-${k})`]),
) as Record<keyof typeof NEUTRAL_LIGHT, string>;

export interface TypeMeta {
  label: string;
  icon: string;
  color: string;
  bg: string;
  on: string;
}
export function typeMeta(type: EventType | string): TypeMeta {
  // The 'event' entry is also the fallback for an unrecognized `type`, so
  // it's bound to its own real object literal (not looked up by indexing
  // `m.event`/`m['event']`, which would be TypeMeta | undefined for the same
  // reason as THEME_PRESETS above) and reused as both the map entry and the
  // guaranteed-defined fallback.
  const eventMeta: TypeMeta = { label: t('eventType.event'), icon: 'celebration', color: '#6A3EA1', bg: '#EADDFF', on: '#23005C' };
  const m: Record<string, TypeMeta> = {
    training: {
      label: t('eventType.training'),
      icon: 'fitness_center',
      color: '#1565C0',
      bg: '#D7E3FF',
      on: '#00315C',
    },
    auftritt: { label: t('eventType.auftritt'), icon: 'emoji_events', color: '#9A5B00', bg: '#FFDDB0', on: '#2E1500' },
    event: eventMeta,
  };
  return m[type] || eventMeta;
}

export interface StatusMeta {
  label: string;
  icon: string;
  color: string;
  bg: string;
}
export function statusMeta(s: AttendanceStatus | string): StatusMeta {
  // Same reasoning as typeMeta()'s eventMeta: 'pending' is also the fallback
  // for an unrecognized status, so it's bound to a real object literal
  // rather than looked up by indexing.
  const pendingMeta: StatusMeta = { label: t('attendance.pending'), icon: 'schedule', color: '#5A5D66', bg: '#E7E8EE' };
  const m: Record<string, StatusMeta> = {
    yes: { label: t('attendance.yes'), icon: 'check_circle', color: '#2E7D32', bg: '#D7F0D8' },
    maybe: { label: t('attendance.maybe'), icon: 'help', color: '#9A5B00', bg: '#FFE5B8' },
    no: { label: t('attendance.no'), icon: 'cancel', color: '#BA1A1A', bg: '#FFDAD6' },
    pending: pendingMeta,
    not_nominated: { label: t('attendance.not_nominated'), icon: 'block', color: '#9A9DA6', bg: '#F0F0F4' },
  };
  return m[s] || pendingMeta;
}

// ---- Formatting helpers -----------------------------------------------------
export const todayStr = todayLocalDate;
export function hhmm(isoStr: string | null): string {
  if (!isoStr) return '';
  if (/^\d{2}:\d{2}$/.test(isoStr)) return isoStr;
  const d = new Date(isoStr);
  return String(d.getHours()).padStart(2, '0') + ':' + String(d.getMinutes()).padStart(2, '0');
}

// Constructing an Intl.DateTimeFormat/NumberFormat is expensive relative to a
// cache lookup, and these helpers run per-row in list views (e.g. finances
// tables), so every unrelated app re-render was reconstructing one per row.
// Mirrors the pluralRulesCache pattern in src/i18n/index.ts. Keyed on locale +
// options since both getIntlLocale()/getCurrency() and the options literal
// can vary per call site.
const dateTimeFormatCache = new Map<string, Intl.DateTimeFormat>();
function getDateTimeFormat(locale: string, options: Intl.DateTimeFormatOptions): Intl.DateTimeFormat {
  const key = locale + JSON.stringify(options);
  let f = dateTimeFormatCache.get(key);
  if (!f) {
    f = new Intl.DateTimeFormat(locale, options);
    dateTimeFormatCache.set(key, f);
  }
  return f;
}
const numberFormatCache = new Map<string, Intl.NumberFormat>();
function getNumberFormat(locale: string, options: Intl.NumberFormatOptions): Intl.NumberFormat {
  const key = locale + JSON.stringify(options);
  let f = numberFormatCache.get(key);
  if (!f) {
    f = new Intl.NumberFormat(locale, options);
    numberFormatCache.set(key, f);
  }
  return f;
}

export const fmtDate = (ds: string) =>
  getDateTimeFormat(getIntlLocale(), { weekday: 'short', day: 'numeric', month: 'short' }).format(
    parseDateOnlyLocal(ds),
  );
export const fmtDateLong = (ds: string) =>
  getDateTimeFormat(getIntlLocale(), { weekday: 'long', day: 'numeric', month: 'long', year: 'numeric' }).format(
    parseDateOnlyLocal(ds),
  );
export function fmtRange(a: string, b: string) {
  const f = (x: string) =>
    getDateTimeFormat(getIntlLocale(), { day: 'numeric', month: 'short' }).format(parseDateOnlyLocal(x));
  return f(a) + ' – ' + f(b);
}
export const fmtMoney = (n: number) =>
  getNumberFormat(getIntlLocale(), { style: 'currency', currency: getCurrency() }).format(n);
export function fmtDateTime(isoStr: string) {
  return getDateTimeFormat(getIntlLocale(), {
    day: 'numeric',
    month: 'short',
    hour: '2-digit',
    minute: '2-digit',
  }).format(new Date(isoStr));
}
export function monthName(ym: string) {
  const [y, m] = String(ym || '').split('-');
  if (y === undefined || m === undefined) return ym || '';
  return getDateTimeFormat(getIntlLocale(), { month: 'long', year: 'numeric' }).format(new Date(+y, +m - 1, 1));
}
export function initials(name: string) {
  return (name || '')
    .split(' ')
    .filter(Boolean)
    .slice(0, 2)
    .map((s) => s[0])
    .join('')
    .toUpperCase();
}
export function relTime(isoStr: string) {
  const mins = Math.round((Date.now() - new Date(isoStr).getTime()) / 60000);
  if (mins < 1) return t('relTime.now');
  if (mins < 60) return t('relTime.minutes', { n: mins });
  const hrs = Math.round(mins / 60);
  if (hrs < 24) return t('relTime.hours', { n: hrs });
  const days = Math.round(hrs / 24);
  return days === 1 ? t('relTime.day') : t('relTime.days', { n: days });
}
