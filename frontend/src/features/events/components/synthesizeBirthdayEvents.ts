import { formatDateOnly, parseDateOnlyLocal } from '@/utils/date';
import type { Member } from '@/features/members';

/** A yearly-recurring birthday pseudo-event synthesized for one calendar occurrence. */
export interface BirthdayEntry {
  membershipId: string;
  name: string;
  /** The occurrence's calendar date (YYYY-MM-DD), always within the requested range. */
  date: string;
}

/**
 * Synthesizes yearly-recurring birthday pseudo-events from each member's
 * `birthday` field for every occurrence -- one per calendar year the range
 * spans -- that falls within [rangeStart, rangeEnd] (inclusive). Nothing is
 * stored per year; this recomputes from the already-loaded member list on
 * every call, so a member's birthday keeps recurring on the calendar without
 * a dedicated event row.
 *
 * Callers are responsible for gating: pass an empty/undefined `members` list
 * (or skip calling this) when the viewer lacks permission to see birthdays.
 */
export function synthesizeBirthdayEvents(
  members: Pick<Member, 'membershipId' | 'name' | 'birthday'>[] | undefined,
  rangeStart: Date,
  rangeEnd: Date,
): BirthdayEntry[] {
  if (!members?.length) return [];
  const entries: BirthdayEntry[] = [];
  for (const m of members) {
    if (!m.birthday) continue;
    const bd = parseDateOnlyLocal(m.birthday);
    for (let year = rangeStart.getFullYear(); year <= rangeEnd.getFullYear(); year++) {
      const occurrence = new Date(year, bd.getMonth(), bd.getDate());
      if (occurrence >= rangeStart && occurrence <= rangeEnd) {
        entries.push({ membershipId: m.membershipId, name: m.name, date: formatDateOnly(occurrence) });
      }
    }
  }
  return entries;
}

/** Groups birthday entries by their occurrence date (YYYY-MM-DD), same shape as the events/absences grouping helpers. */
export function groupBirthdaysByDate(entries: BirthdayEntry[]): Record<string, BirthdayEntry[]> {
  const byDate: Record<string, BirthdayEntry[]> = {};
  entries.forEach((e) => {
    (byDate[e.date] = byDate[e.date] || []).push(e);
  });
  return byDate;
}
