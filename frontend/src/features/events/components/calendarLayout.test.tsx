import { afterEach, describe, expect, it, vi } from 'vitest';
import { act, render } from '@testing-library/react';
import { useRef } from 'react';
import { BASE_LIMITS, cellLimits, linesThatFit, useStretchedRowHeight } from './calendarLayout';

describe('linesThatFit', () => {
  it('is null while the row height is unknown', () => {
    expect(linesThatFit(null, false)).toBeNull();
    expect(linesThatFit(null, true)).toBeNull();
  });

  it('counts whole 17px lines after the 34px cell overhead on desktop', () => {
    expect(linesThatFit(76, false)).toBe(2); // (76 - 34) / 17 = 2.47
    expect(linesThatFit(85, false)).toBe(3); // exactly 3 lines
    expect(linesThatFit(84, false)).toBe(2); // one pixel short of 3
    expect(linesThatFit(170, false)).toBe(8);
  });

  it('counts whole 16px lines after the 30px cell overhead on mobile', () => {
    expect(linesThatFit(58, true)).toBe(1); // (58 - 30) / 16 = 1.75
    expect(linesThatFit(62, true)).toBe(2);
    expect(linesThatFit(110, true)).toBe(5);
  });

  it('never goes negative for a row smaller than the overhead', () => {
    expect(linesThatFit(10, false)).toBe(0);
  });
});

describe('cellLimits', () => {
  const none = { events: 0, birthdays: 0, absences: 0 };

  it('returns the fixed base limits when the height is unmeasured', () => {
    expect(cellLimits(null, false, { events: 9, birthdays: 1, absences: 9 })).toEqual(BASE_LIMITS.desktop);
    expect(cellLimits(null, true, { events: 9, birthdays: 1, absences: 9 })).toEqual(BASE_LIMITS.mobile);
  });

  it('never drops below the base limits, however little room there is', () => {
    expect(cellLimits(0, false, { events: 9, birthdays: 2, absences: 9 })).toEqual({ events: 3, absences: 2 });
    expect(cellLimits(2, false, { events: 9, birthdays: 0, absences: 9 })).toEqual({ events: 3, absences: 2 });
    expect(cellLimits(1, true, { events: 9, birthdays: 0, absences: 9 })).toEqual({ events: 2, absences: 1 });
  });

  it('keeps the base limits for an empty day', () => {
    expect(cellLimits(8, false, none)).toEqual({ events: 3, absences: 2 });
  });

  it('shows every event when they all fit', () => {
    // 5 events in 8 lines -> all 5, no marker needed.
    expect(cellLimits(8, false, { ...none, events: 5 })).toEqual({ events: 5, absences: 2 });
  });

  it('reserves one line for the "+n" marker when not every event fits', () => {
    // 12 events, 8 lines: 7 chips + the "+5" marker.
    expect(cellLimits(8, false, { ...none, events: 12 }).events).toBe(7);
  });

  it('fills exactly: as many events as lines need no marker line', () => {
    expect(cellLimits(8, false, { ...none, events: 8 }).events).toBe(8);
    // One more than fits -> a marker is needed, so one chip fewer.
    expect(cellLimits(8, false, { ...none, events: 9 }).events).toBe(7);
  });

  it('sets room aside for birthdays and the absences base share before raising the event limit', () => {
    // 8 lines - 1 birthday - 2 base absences - 1 absence marker = 4 for events;
    // 6 events don't fit in 4 -> 3 chips + marker (= the base limit).
    const l = cellLimits(8, false, { events: 6, birthdays: 1, absences: 5 });
    expect(l.events).toBe(3);
    // 3 events + marker + 1 birthday = 5 lines used -> 3 left for absences:
    // 5 absences don't fit -> 2 chips + marker.
    expect(l.absences).toBe(2);
  });

  it('gives absences the room events leave unused', () => {
    // 1 event, 6 absences, 8 lines: 7 lines left -> all 6 fit.
    expect(cellLimits(8, false, { events: 1, birthdays: 0, absences: 6 })).toEqual({ events: 3, absences: 6 });
  });

  it('never lets the rendered lines exceed the cell when limits are raised above base', () => {
    for (let lines = 0; lines <= 14; lines++) {
      for (let events = 0; events <= 12; events++) {
        for (let absences = 0; absences <= 12; absences++) {
          for (let birthdays = 0; birthdays <= 2; birthdays++) {
            const l = cellLimits(lines, false, { events, birthdays, absences });
            const shownEvents = Math.min(events, l.events);
            const shownAbsences = Math.min(absences, l.absences);
            const used =
              shownEvents + (events > shownEvents ? 1 : 0) + birthdays + shownAbsences + (absences > shownAbsences ? 1 : 0);
            // What the old fixed limits would have rendered for this day.
            const baseEvents = Math.min(events, 3);
            const baseAbsences = Math.min(absences, 2);
            const baseUsed =
              baseEvents + (events > baseEvents ? 1 : 0) + birthdays + baseAbsences + (absences > baseAbsences ? 1 : 0);
            const where = `lines=${lines} e=${events} a=${absences} b=${birthdays}`;
            expect(l.events, where).toBeGreaterThanOrEqual(3);
            expect(l.absences, where).toBeGreaterThanOrEqual(2);
            // Either it fits, or it is no taller than the previous layout.
            expect(used <= lines || used <= baseUsed, where).toBe(true);
          }
        }
      }
    }
  });
});

describe('useStretchedRowHeight', () => {
  const realRO = globalThis.ResizeObserver;
  afterEach(() => {
    globalThis.ResizeObserver = realRO;
    vi.restoreAllMocks();
  });

  function Probe({ gap, onValue }: { gap: number; onValue: (v: number | null) => void }) {
    const ref = useRef<HTMLDivElement>(null);
    onValue(useStretchedRowHeight(ref, gap));
    return (
      <div data-testid="card" style={{ paddingBottom: '16px', borderBottom: '1px solid #000' }}>
        <div ref={ref} data-testid="grid">
          <div data-testid="weekday-header" />
        </div>
      </div>
    );
  }

  // jsdom lays nothing out: fake the three measurements the hook reads.
  function fakeLayout(opts: { mainHeight: number; gridTop: number; headerHeight: number; scrollTop?: number }) {
    const rect = (top: number, height: number) => ({ top, height, bottom: top + height }) as DOMRect;
    vi.spyOn(HTMLElement.prototype, 'getBoundingClientRect').mockImplementation(function (this: HTMLElement) {
      if (this.tagName === 'MAIN') return rect(100, 0);
      if (this.dataset.testid === 'grid') return rect(100 + opts.gridTop - (opts.scrollTop ?? 0), 0);
      if (this.dataset.testid === 'weekday-header') return rect(0, opts.headerHeight);
      return rect(0, 0);
    });
    vi.spyOn(HTMLElement.prototype, 'clientHeight', 'get').mockImplementation(function (this: HTMLElement) {
      return this.tagName === 'MAIN' ? opts.mainHeight : 0;
    });
    vi.spyOn(HTMLElement.prototype, 'scrollTop', 'get').mockImplementation(function (this: HTMLElement) {
      return this.tagName === 'MAIN' ? (opts.scrollTop ?? 0) : 0;
    });
  }

  function stubResizeObserver() {
    const instances: { cb: ResizeObserverCallback; observed: Element[]; disconnected: boolean }[] = [];
    globalThis.ResizeObserver = class {
      private rec: (typeof instances)[number];
      constructor(cb: ResizeObserverCallback) {
        this.rec = { cb, observed: [], disconnected: false };
        instances.push(this.rec);
      }
      observe(el: Element) {
        this.rec.observed.push(el);
      }
      unobserve() {}
      disconnect() {
        this.rec.disconnected = true;
      }
    } as unknown as typeof ResizeObserver;
    return instances;
  }

  it('is null without ResizeObserver (the base limits then apply)', () => {
    // @ts-expect-error -- simulating a browser/jsdom without the API
    delete globalThis.ResizeObserver;
    const values: (number | null)[] = [];
    render(
      <main>
        <Probe gap={6} onValue={(v) => values.push(v)} />
      </main>,
    );
    expect(values[values.length - 1]).toBeNull();
  });

  it('is null when the grid has no scrolling <main> ancestor', () => {
    stubResizeObserver();
    const values: (number | null)[] = [];
    render(<Probe gap={6} onValue={(v) => values.push(v)} />);
    expect(values[values.length - 1]).toBeNull();
  });

  it('derives the row height from the scroller height and the grid top edge', () => {
    const observers = stubResizeObserver();
    // main: 900px tall with 56px bottom padding; grid starts 200px below
    // main's top; 24px weekday header; card adds 16px padding + 1px border.
    fakeLayout({ mainHeight: 900, gridTop: 200, headerHeight: 24 });
    const values: (number | null)[] = [];
    const { container } = render(
      <main style={{ paddingBottom: '56px' }}>
        <Probe gap={6} onValue={(v) => values.push(v)} />
      </main>,
    );
    // 900 - 200 (top) - 56 - 16 - 1 (below) - 24 (header) - 6 * 6 (gaps) = 567 -> 94 per row.
    expect(values[values.length - 1]).toBe(94);
    expect(observers).toHaveLength(1);
    expect(observers[0]!.observed).toEqual([container.querySelector('main')]);
  });

  it('is independent of the scroll position', () => {
    stubResizeObserver();
    fakeLayout({ mainHeight: 900, gridTop: 200, headerHeight: 24, scrollTop: 150 });
    const values: (number | null)[] = [];
    render(
      <main style={{ paddingBottom: '56px' }}>
        <Probe gap={6} onValue={(v) => values.push(v)} />
      </main>,
    );
    expect(values[values.length - 1]).toBe(94);
  });

  it('re-measures when the scroller resizes', () => {
    const observers = stubResizeObserver();
    fakeLayout({ mainHeight: 900, gridTop: 200, headerHeight: 24 });
    const values: (number | null)[] = [];
    render(
      <main style={{ paddingBottom: '56px' }}>
        <Probe gap={6} onValue={(v) => values.push(v)} />
      </main>,
    );
    expect(values[values.length - 1]).toBe(94);

    vi.restoreAllMocks();
    fakeLayout({ mainHeight: 1300, gridTop: 200, headerHeight: 24 });
    act(() => observers[0]!.cb([], {} as ResizeObserver));
    // 1300 - 200 - 73 - 24 - 36 = 967 -> 161 per row.
    expect(values[values.length - 1]).toBe(161);
  });

  it('is null when there is no room left below the grid top', () => {
    stubResizeObserver();
    fakeLayout({ mainHeight: 300, gridTop: 280, headerHeight: 24 });
    const values: (number | null)[] = [];
    render(
      <main style={{ paddingBottom: '56px' }}>
        <Probe gap={6} onValue={(v) => values.push(v)} />
      </main>,
    );
    expect(values[values.length - 1]).toBeNull();
  });

  it('disconnects its observer on unmount', () => {
    const observers = stubResizeObserver();
    fakeLayout({ mainHeight: 900, gridTop: 200, headerHeight: 24 });
    const { unmount } = render(
      <main>
        <Probe gap={6} onValue={() => {}} />
      </main>,
    );
    unmount();
    expect(observers[0]!.disconnected).toBe(true);
  });
});
