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
};
