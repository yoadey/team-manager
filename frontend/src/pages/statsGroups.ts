import type { EventGroupStat } from '@/types';

/**
 * How many of a group's members were expected at the event: everyone with a
 * counted response plus everyone who gave no response at all. A missing
 * response counts as not present, so this -- not `counted` alone -- is what
 * "fully present" and the attendance percentage are measured against.
 */
export function groupSize(g: Pick<EventGroupStat, 'counted' | 'pending'>): number {
  return g.counted + g.pending;
}

/**
 * What a group's (role's) indicator shows for one event:
 * - `none`: nobody in the group was expected at the event -- no data, never
 *   red.
 * - `complete`: every expected member attended -- the check mark. A member
 *   who gave no response is not present, so three "yes" and twelve silent
 *   members are *not* complete. Complete always wins over the threshold (a
 *   fully present two-person group with a threshold of three attendees is
 *   still complete, not "too low").
 * - `pct`: the share of the group that attended, `low` when it is below the
 *   group's threshold -- below `thresholdValue` percent in 'percent' mode,
 *   fewer than `thresholdValue` attendees in 'count' mode.
 */
export type GroupIndicator = { kind: 'none' } | { kind: 'complete' } | { kind: 'pct'; pct: number; low: boolean };

export function groupIndicator(g: EventGroupStat): GroupIndicator {
  const size = groupSize(g);
  if (size <= 0) return { kind: 'none' };
  if (g.yes >= size) return { kind: 'complete' };
  // The threshold is compared against the exact ratio, not the rounded label
  // -- 49.6% is below a 50% threshold even though it displays as "50%".
  const ratio = (g.yes / size) * 100;
  const low = g.thresholdMode === 'count' ? g.yes < g.thresholdValue : ratio < g.thresholdValue;
  // Capped at 99 so an incomplete group (e.g. 199/200) never reads "100%"
  // without the check mark that "complete" alone gets.
  return { kind: 'pct', pct: Math.min(99, Math.round(ratio)), low };
}
