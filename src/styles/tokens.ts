// =============================================================================
// Design tokens — the single source of truth for colours, type/status metadata
// and small formatting helpers. Values are taken verbatim from the design
// handoff (section 8, Design-Tokens) so the MUI build matches pixel for pixel.
// =============================================================================

import type { AttendanceStatus, EventType } from '../types';
import { parseDateOnlyLocal, todayLocalDate } from '../utils/date';

export interface ThemePreset {
  primary: string;
  onPrimary: string;
  primaryContainer: string;
  onPrimaryContainer: string;
  secondaryContainer: string;
  onSecondaryContainer: string;
}

export const THEME_PRESETS: Record<string, ThemePreset> = {
  '#1565C0': { primary: '#1565C0', onPrimary: '#FFFFFF', primaryContainer: '#D7E3FF', onPrimaryContainer: '#001B3E', secondaryContainer: '#DCE3F2', onSecondaryContainer: '#101C2B' },
  '#6750A4': { primary: '#6750A4', onPrimary: '#FFFFFF', primaryContainer: '#EADDFF', onPrimaryContainer: '#21005D', secondaryContainer: '#E8DEF8', onSecondaryContainer: '#1D192B' },
  '#00796B': { primary: '#00796B', onPrimary: '#FFFFFF', primaryContainer: '#9DF1E2', onPrimaryContainer: '#00201B', secondaryContainer: '#CCE8E2', onSecondaryContainer: '#0B1F1B' },
  '#B71C1C': { primary: '#B3261E', onPrimary: '#FFFFFF', primaryContainer: '#F9DEDC', onPrimaryContainer: '#410E0B', secondaryContainer: '#F4DDDA', onSecondaryContainer: '#2C1512' },
  '#33691E': { primary: '#386A20', onPrimary: '#FFFFFF', primaryContainer: '#B7F397', onPrimaryContainer: '#072100', secondaryContainer: '#D8E7CD', onSecondaryContainer: '#121F0C' },
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
  const p = THEME_PRESETS[presetKey] || THEME_PRESETS[DEFAULT_PRESET_KEY];
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

// Neutral / line colours used across cards & surfaces.
export const NEUTRAL = {
  appBg: '#E4E5EC',
  surface: '#FBFBFE',
  card: '#FFFFFF',
  sidebar: '#F4F4FA',
  onSurface: '#1A1C20',
  onSurfaceVariant: '#44474E',
  secondary: '#6A6D76',
  faint: '#9A9DA6',
  line: '#E6E7EE',
  line2: '#ECEDF3',
  line3: '#E0E2EA',
  inputBorder: '#C8CAD2',
};

export interface TypeMeta { label: string; icon: string; color: string; bg: string; on: string; }
export function typeMeta(type: EventType | string): TypeMeta {
  const m: Record<string, TypeMeta> = {
    training: { label: 'Training', icon: 'fitness_center', color: '#1565C0', bg: '#D7E3FF', on: '#00315C' },
    auftritt: { label: 'Auftritt / Turnier', icon: 'emoji_events', color: '#9A5B00', bg: '#FFDDB0', on: '#2E1500' },
    event: { label: 'Team-Event', icon: 'celebration', color: '#6A3EA1', bg: '#EADDFF', on: '#23005C' },
  };
  return m[type] || m.event;
}

export interface StatusMeta { label: string; icon: string; color: string; bg: string; }
export function statusMeta(s: AttendanceStatus | string): StatusMeta {
  const m: Record<string, StatusMeta> = {
    yes: { label: 'Zugesagt', icon: 'check_circle', color: '#2E7D32', bg: '#D7F0D8' },
    maybe: { label: 'Unsicher', icon: 'help', color: '#9A5B00', bg: '#FFE5B8' },
    no: { label: 'Abgesagt', icon: 'cancel', color: '#BA1A1A', bg: '#FFDAD6' },
    pending: { label: 'Offen', icon: 'schedule', color: '#5A5D66', bg: '#E7E8EE' },
    not_nominated: { label: 'Nicht nominiert', icon: 'block', color: '#9A9DA6', bg: '#F0F0F4' },
  };
  return m[s] || m.pending;
}

// ---- Formatting helpers -----------------------------------------------------
export const todayStr = todayLocalDate;
export function hhmm(isoStr: string | null): string {
  if (!isoStr) return '';
  if (/^\d{2}:\d{2}$/.test(isoStr)) return isoStr;
  const d = new Date(isoStr);
  return String(d.getHours()).padStart(2, '0') + ':' + String(d.getMinutes()).padStart(2, '0');
}
export const fmtDate = (ds: string) =>
  new Intl.DateTimeFormat('de-DE', { weekday: 'short', day: 'numeric', month: 'short' }).format(parseDateOnlyLocal(ds));
export const fmtDateLong = (ds: string) =>
  new Intl.DateTimeFormat('de-DE', { weekday: 'long', day: 'numeric', month: 'long', year: 'numeric' }).format(parseDateOnlyLocal(ds));
export function fmtRange(a: string, b: string) {
  const f = (x: string) => new Intl.DateTimeFormat('de-DE', { day: 'numeric', month: 'short' }).format(parseDateOnlyLocal(x));
  return f(a) + ' – ' + f(b);
}
export const fmtMoney = (n: number) => new Intl.NumberFormat('de-DE', { style: 'currency', currency: 'EUR' }).format(n);
export function fmtDateTime(isoStr: string) {
  return new Intl.DateTimeFormat('de-DE', { day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit' }).format(new Date(isoStr));
}
export function monthName(ym: string) {
  const p = String(ym || '').split('-');
  if (p.length < 2) return ym || '';
  return new Intl.DateTimeFormat('de-DE', { month: 'long', year: 'numeric' }).format(new Date(+p[0], +p[1] - 1, 1));
}
export function initials(name: string) {
  return (name || '').split(' ').filter(Boolean).slice(0, 2).map((s) => s[0]).join('').toUpperCase();
}
export function relTime(isoStr: string) {
  const mins = Math.round((Date.now() - new Date(isoStr).getTime()) / 60000);
  if (mins < 1) return 'gerade eben';
  if (mins < 60) return 'vor ' + mins + ' Min';
  const hrs = Math.round(mins / 60);
  if (hrs < 24) return 'vor ' + hrs + ' Std';
  const days = Math.round(hrs / 24);
  return 'vor ' + days + ' ' + (days === 1 ? 'Tag' : 'Tagen');
}
