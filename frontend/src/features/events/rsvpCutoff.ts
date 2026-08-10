import { zonedTimeToUtc } from '@/utils/date';
import type { TeamEvent } from './types';

// Every event's wall-clock meet/start/end time is interpreted in this fixed
// timezone -- neither this app nor its backend counterpart (events.
// ZonedTimeToUTC) have a per-team timezone concept yet.
const TEAM_TIMEZONE = 'Europe/Berlin';

type CutoffEventFields = Pick<TeamEvent, 'date' | 'startTime' | 'meetTime' | 'cancelLeadMinutes'>;

/**
 * The absolute instant an event starts: startTime, falling back to
 * meetTime, falling back to 18:00, interpreted as Europe/Berlin wall-clock.
 * Mirrors the backend's events.EventStartInstant so both sides derive the
 * same "start" a cancelLeadMinutes cutoff counts back from.
 */
function eventStartInstant(event: Pick<TeamEvent, 'date' | 'startTime' | 'meetTime'>): Date {
  const hhmm = event.startTime || event.meetTime || '18:00';
  return zonedTimeToUtc(event.date, hhmm, TEAM_TIMEZONE);
}

/**
 * The effective RSVP/cancellation cutoff for an event: event start minus
 * cancelLeadMinutes, if set -- once passed, it blocks a self-service
 * attendance change (see the backend's identical events.Service.
 * SetAttendance gating). Returns null when no cutoff is configured.
 */
export function effectiveRsvpCutoff(event: CutoffEventFields): Date | null {
  if (event.cancelLeadMinutes == null) return null;
  return new Date(eventStartInstant(event).getTime() - event.cancelLeadMinutes * 60_000);
}

/** Whether event's effective RSVP cutoff (if any) has already passed as of `now`. */
export function isRsvpCutoffPassed(event: CutoffEventFields, now: number = Date.now()): boolean {
  const cutoff = effectiveRsvpCutoff(event);
  return cutoff !== null && now >= cutoff.getTime();
}

/**
 * The last calendar day (YYYY-MM-DD) an event occurs on: multiDayEndDate
 * when set, otherwise date -- mirrors backend/internal/events/repository.go's
 * ListEvents `COALESCE(end_date, date)`. Every UI spot that classifies an
 * event as past/upcoming (event lists, cards, RSVP controls) must derive
 * "past" from this, not from `date` alone, or an ongoing multi-day event
 * reads as already over the moment its first day passes.
 */
export function eventEffectiveEndDate(event: Pick<TeamEvent, 'date' | 'multiDayEndDate'>): string {
  return event.multiDayEndDate || event.date;
}

/** Whether an event's last occurring day is strictly before `today` (YYYY-MM-DD). */
export function isEventPast(event: Pick<TeamEvent, 'date' | 'multiDayEndDate'>, today: string): boolean {
  return eventEffectiveEndDate(event) < today;
}
