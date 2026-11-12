import { describe, it, expect } from 'vitest';
import { groupIndicator, groupSize } from './statsGroups';
import type { EventGroupStat, StatsThresholdMode } from '@/types';

function g(
  yes: number,
  counted: number,
  thresholdMode: StatsThresholdMode,
  thresholdValue: number,
  pending = 0,
): EventGroupStat {
  return { roleId: 'r1', name: 'Spieler', color: '#000', yes, counted, pending, thresholdMode, thresholdValue };
}

describe('groupSize', () => {
  it('is everyone expected: counted responses plus missing responses', () => {
    expect(groupSize({ counted: 3, pending: 12 })).toBe(15);
    expect(groupSize({ counted: 0, pending: 0 })).toBe(0);
  });
});

describe('groupIndicator', () => {
  it('is "none" when nobody of the group was expected -- never complete, never low', () => {
    expect(groupIndicator(g(0, 0, 'percent', 50))).toEqual({ kind: 'none' });
    expect(groupIndicator(g(0, 0, 'count', 8))).toEqual({ kind: 'none' });
  });

  it('is "complete" when every counted member attended', () => {
    expect(groupIndicator(g(5, 5, 'percent', 50))).toEqual({ kind: 'complete' });
    expect(groupIndicator(g(1, 1, 'percent', 100))).toEqual({ kind: 'complete' });
  });

  it('lets complete win over an absolute threshold the group cannot reach', () => {
    // Two trainers, both there, threshold "at least 3 attendees".
    expect(groupIndicator(g(2, 2, 'count', 3))).toEqual({ kind: 'complete' });
  });

  describe('percent mode', () => {
    it('is low strictly below the threshold', () => {
      expect(groupIndicator(g(2, 5, 'percent', 50))).toEqual({ kind: 'pct', pct: 40, low: true });
    });

    it('is not low exactly at the threshold', () => {
      expect(groupIndicator(g(1, 2, 'percent', 50))).toEqual({ kind: 'pct', pct: 50, low: false });
    });

    it('is not low above the threshold', () => {
      expect(groupIndicator(g(3, 5, 'percent', 50))).toEqual({ kind: 'pct', pct: 60, low: false });
    });

    it('compares the exact ratio, not the rounded label', () => {
      // 124/250 = 49.6% -> displays "50%" but is below a 50% threshold.
      expect(groupIndicator(g(124, 250, 'percent', 50))).toEqual({ kind: 'pct', pct: 50, low: true });
    });

    it('never marks anything low with a threshold of 0', () => {
      expect(groupIndicator(g(0, 5, 'percent', 0))).toEqual({ kind: 'pct', pct: 0, low: false });
    });

    it('marks every incomplete group low with a threshold of 100', () => {
      expect(groupIndicator(g(9, 10, 'percent', 100))).toEqual({ kind: 'pct', pct: 90, low: true });
    });
  });

  describe('count mode', () => {
    it('is low when fewer members attended than the threshold', () => {
      expect(groupIndicator(g(6, 10, 'count', 8))).toEqual({ kind: 'pct', pct: 60, low: true });
    });

    it('is not low when exactly the threshold attended', () => {
      expect(groupIndicator(g(8, 10, 'count', 8))).toEqual({ kind: 'pct', pct: 80, low: false });
    });

    it('is not low when more than the threshold attended', () => {
      expect(groupIndicator(g(9, 10, 'count', 8))).toEqual({ kind: 'pct', pct: 90, low: false });
    });

    it('ignores the percentage -- a low ratio with enough attendees is fine', () => {
      expect(groupIndicator(g(8, 40, 'count', 8))).toEqual({ kind: 'pct', pct: 20, low: false });
    });

    it('never marks anything low with a threshold of 0', () => {
      expect(groupIndicator(g(0, 5, 'count', 0))).toEqual({ kind: 'pct', pct: 0, low: false });
    });
  });

  // A member who gave no response counts as not present.
  describe('missing responses', () => {
    it('does not show a group as complete when some members never responded', () => {
      // Three "yes", twelve silent: 3 of 15, not "3 of 3".
      expect(groupIndicator(g(3, 3, 'percent', 50, 12))).toEqual({ kind: 'pct', pct: 20, low: true });
    });

    it('applies an absolute threshold against the attendees, however few responded', () => {
      expect(groupIndicator(g(3, 3, 'count', 8, 12))).toEqual({ kind: 'pct', pct: 20, low: true });
      // Enough attendees, even though some never answered.
      expect(groupIndicator(g(8, 8, 'count', 8, 2))).toEqual({ kind: 'pct', pct: 80, low: false });
    });

    it('measures the percentage against the whole group, not only the responders', () => {
      // 6 yes, 2 no, 2 silent -> 6 of 10.
      expect(groupIndicator(g(6, 8, 'percent', 70, 2))).toEqual({ kind: 'pct', pct: 60, low: true });
      expect(groupIndicator(g(6, 8, 'percent', 60, 2))).toEqual({ kind: 'pct', pct: 60, low: false });
    });

    it('shows a group where nobody responded as 0%, not as "no data"', () => {
      expect(groupIndicator(g(0, 0, 'percent', 50, 5))).toEqual({ kind: 'pct', pct: 0, low: true });
      // Threshold 0 = never red, also here.
      expect(groupIndicator(g(0, 0, 'percent', 0, 5))).toEqual({ kind: 'pct', pct: 0, low: false });
    });

    it('is complete only once nobody is missing', () => {
      expect(groupIndicator(g(5, 5, 'percent', 50, 1))).toEqual({ kind: 'pct', pct: 83, low: false });
      expect(groupIndicator(g(5, 5, 'percent', 50, 0))).toEqual({ kind: 'complete' });
    });
  });

  it('caps the label at 99% for an incomplete group that would round to 100%', () => {
    expect(groupIndicator(g(199, 200, 'percent', 50))).toEqual({ kind: 'pct', pct: 99, low: false });
  });
});
