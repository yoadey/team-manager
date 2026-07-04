const DATE_ONLY_RE = /^(\d{4})-(\d{2})-(\d{2})$/;
const HHMM_RE = /^(\d{2}):(\d{2})$/;

const pad = (n: number) => String(n).padStart(2, '0');

/** Returns today's calendar date in the user's local timezone as YYYY-MM-DD. */
export function todayLocalDate(): string {
  return formatDateOnly(new Date());
}

/**
 * Lower bound passed as an explicit `from` for a genuinely all-time stats
 * query. The stats API has no dedicated "unbounded" flag -- an omitted
 * `from`/`to` instead makes both service layers apply their 3-month default
 * range (see serviceLayer.ts/serviceLayerReal.ts's `teamOverview` and the
 * backend's `stats.Service.defaultDateRange`). Passing this fixed, far-past
 * date explicitly bypasses that default and returns every event on record,
 * since no real club's data predates it.
 */
export const ALL_TIME_FROM_DATE = '1970-01-01';

/** Parses a YYYY-MM-DD calendar date as local midnight, never as UTC. */
export function parseDateOnlyLocal(date: string): Date {
  const match = DATE_ONLY_RE.exec(date);
  if (!match) return new Date(date);
  return new Date(Number(match[1]), Number(match[2]) - 1, Number(match[3]), 0, 0, 0, 0);
}

/** Formats a Date as YYYY-MM-DD using local calendar fields. */
export function formatDateOnly(date: Date): string {
  return date.getFullYear() + '-' + pad(date.getMonth() + 1) + '-' + pad(date.getDate());
}

/** Combines a YYYY-MM-DD date and HH:mm local time into a local Date. */
export function combineDateAndTimeLocal(date: string, hhmm: string): Date {
  const day = parseDateOnlyLocal(date);
  const match = HHMM_RE.exec(hhmm);
  if (!match) return day;
  day.setHours(Number(match[1]), Number(match[2]), 0, 0);
  return day;
}
