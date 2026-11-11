import type { EventGroupStat } from '@/types';

/**
 * What a group's (role's) indicator shows for one event:
 * - `none`: nobody in the group has a counted response -- no data, never red.
 * - `complete`: every counted member attended -- the check mark. Always wins
 *   over the threshold (a fully present two-person group with a threshold of
 *   three attendees is still complete, not "too low").
 * - `pct`: the attendance percentage, `low` when it is below the group's
 *   threshold -- below `thresholdValue` percent in 'percent' mode, fewer than
 *   `thresholdValue` attendees in 'count' mode.
 */
export type GroupIndicator = { kind: 'none' } | { kind: 'complete' } | { kind: 'pct'; pct: number; low: boolean };

export function groupIndicator(g: EventGroupStat): GroupIndicator {
  if (g.counted <= 0) return { kind: 'none' };
  if (g.yes >= g.counted) return { kind: 'complete' };
  // The threshold is compared against the exact ratio, not the rounded label
  // -- 49.6% is below a 50% threshold even though it displays as "50%".
  const ratio = (g.yes / g.counted) * 100;
  const low = g.thresholdMode === 'count' ? g.yes < g.thresholdValue : ratio < g.thresholdValue;
  // Capped at 99 so an incomplete group (e.g. 199/200) never reads "100%"
  // without the check mark that "complete" alone gets.
  return { kind: 'pct', pct: Math.min(99, Math.round(ratio)), low };
}
