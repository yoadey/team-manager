import type { DateRange } from '@/types';

/**
 * Team-scoped query key factory. Every server-state key is prefixed with the
 * owning team id so that switching the active team changes the key -- React
 * Query then treats the previous team's in-flight/cached queries as a
 * different cache entry instead of letting a late response for team A
 * overwrite team B's screen.
 */
export const queryKeys = {
  events: (teamId: string) => ['teams', teamId, 'events'] as const,
  eventDetail: (teamId: string, eventId: string) => ['teams', teamId, 'events', eventId] as const,
  members: (teamId: string) => ['teams', teamId, 'members'] as const,
  finances: (teamId: string) => ['teams', teamId, 'finances'] as const,
  polls: (teamId: string) => ['teams', teamId, 'polls'] as const,
  news: (teamId: string) => ['teams', teamId, 'news'] as const,
  absences: (teamId: string) => ['teams', teamId, 'absences'] as const,
  myAbsences: (teamId: string) => ['teams', teamId, 'myAbsences'] as const,
  notifications: (teamId: string) => ['teams', teamId, 'notifications'] as const,
  // staleTime: Infinity at the call site -- issuing a token rotates it
  // server-side, so this must only ever be (re)fetched on an explicit user
  // action (open the sheet once, or hit "renew"), never as a background
  // refetch that would silently invalidate a link the user already copied
  // or added to their calendar app.
  calendarFeedUrl: (teamId: string) => ['teams', teamId, 'calendarFeedUrl'] as const,
  // Also varies by date range (unlike every other key here): a range change
  // must swap to a different cache entry the same way a team switch does,
  // rather than reusing/overwriting the previous range's cached data.
  stats: (teamId: string, range: DateRange | null) =>
    ['teams', teamId, 'stats', range?.from ?? null, range?.to ?? null] as const,
};
