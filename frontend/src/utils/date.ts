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

/** Offset in minutes of timeZone relative to UTC at the given instant (e.g. +120 for CEST). */
function tzOffsetMinutesAt(instant: Date, timeZone: string): number {
  const dtf = new Intl.DateTimeFormat('en-US', {
    timeZone,
    hourCycle: 'h23',
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  });
  const parts: Record<string, string> = {};
  for (const p of dtf.formatToParts(instant)) parts[p.type] = p.value;
  const asUtc = Date.UTC(
    Number(parts.year),
    Number(parts.month) - 1,
    Number(parts.day),
    Number(parts.hour) % 24,
    Number(parts.minute),
    Number(parts.second),
  );
  return (asUtc - instant.getTime()) / 60000;
}

/**
 * Combines a YYYY-MM-DD date and HH:mm wall-clock time understood to be in
 * timeZone into the correct UTC Date, regardless of the calling browser's
 * own timezone. Unlike combineDateAndTimeLocal (which reinterprets the same
 * strings in whatever timezone the browser happens to be running in), this
 * is for data -- like event date/time fields, documented as team-local
 * (Europe/Berlin) wall-clock strings -- that must resolve to the same
 * absolute instant no matter where the caller is physically located, such
 * as building an .ics export consumed by calendar apps in other timezones.
 * Accurate outside the ~1 hour DST-transition ambiguity window twice a year,
 * an accepted edge case shared by most lightweight tz-conversion helpers.
 */
export function zonedTimeToUtc(date: string, hhmm: string, timeZone: string): Date {
  const match = DATE_ONLY_RE.exec(date);
  if (!match) return parseDateOnlyLocal(date);
  const year = Number(match[1]);
  const month = Number(match[2]);
  const day = Number(match[3]);
  const time = HHMM_RE.exec(hhmm);
  const hour = time ? Number(time[1]) : 0;
  const minute = time ? Number(time[2]) : 0;
  const guessUtcMs = Date.UTC(year, month - 1, day, hour, minute, 0);
  const offsetMinutes = tzOffsetMinutesAt(new Date(guessUtcMs), timeZone);
  return new Date(guessUtcMs - offsetMinutes * 60000);
}
