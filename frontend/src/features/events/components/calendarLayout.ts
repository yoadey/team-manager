import { useEffect, useState, type RefObject } from 'react';

/** How many event / absence chips a day cell shows before "+n". */
export interface CellLimits {
  events: number;
  absences: number;
}

/**
 * The fixed limits the calendar has always used -- now the *minimum*: a cell
 * never shows fewer entries than this, and shows exactly this many wherever
 * the available height can't be measured.
 */
export const BASE_LIMITS: { desktop: CellLimits; mobile: CellLimits } = {
  desktop: { events: 3, absences: 2 },
  mobile: { events: 2, absences: 1 },
};

/** Minimum day-cell height -- unchanged from the pre-stretch layout. */
export const MIN_ROW_HEIGHT = { desktop: 76, mobile: 58 } as const;

// Vertical space one entry line takes inside a day cell, including the 2px
// gap between lines, and the space a cell needs before its first entry
// (2 x padding + 2 x border + the 20px day number + its gap). The line
// height is the tallest chip rounded up: on desktop an absence/birthday chip
// renders at 15.5px (an event chip at 15px), on mobile an event chip at 14px.
// Rounding up matters: a raised limit must never make a cell taller than
// its row (see weekRowTemplate), so the estimate has to err on the side of
// one entry too few.
const LINE_HEIGHT = { desktop: 18, mobile: 16 } as const;
const CELL_OVERHEAD = { desktop: 34, mobile: 30 } as const;

/**
 * `grid-template-rows` for the weekday header plus the six week rows.
 *
 * Each week row is at least `rowHeight` tall -- the measured share of the
 * free height (never below the fixed minimum) -- and otherwise as tall as
 * its own content (`auto`). So the rows are equal and fill the screen as
 * long as every day fits, and a day with more entries than fit grows *only
 * its own week*, exactly like before the calendar stretched.
 *
 * Deliberately not `1fr`: the grid's height is indefinite (the page is only
 * min-height-constrained, so it can grow and scroll), and `fr` rows sized
 * against an indefinite height all take the size of the tallest row -- one
 * busy day would inflate all six weeks and push the calendar off screen.
 */
export function weekRowTemplate(rowHeight: number | null, mobile: boolean): string {
  const min = mobile ? MIN_ROW_HEIGHT.mobile : MIN_ROW_HEIGHT.desktop;
  return `auto repeat(6, minmax(${Math.max(min, rowHeight ?? 0)}px, auto))`;
}

/**
 * How many entry lines fit into a day cell of the given row height, or null
 * when the height is unknown.
 */
export function linesThatFit(rowHeight: number | null, mobile: boolean): number | null {
  if (rowHeight === null) return null;
  const key = mobile ? 'mobile' : 'desktop';
  return Math.max(0, Math.floor((rowHeight - CELL_OVERHEAD[key]) / LINE_HEIGHT[key]));
}

/**
 * Per-cell entry limits: the base limits, raised as far as the cell's `lines`
 * have room for this particular day's entries. Events are served first, then
 * birthdays (always shown in full), then absences get what is left. A "+n"
 * overflow marker is itself a line, so a limit is only raised when the
 * entries it adds still fit *with* the marker a remaining overflow needs.
 */
export function cellLimits(
  lines: number | null,
  mobile: boolean,
  counts: { events: number; birthdays: number; absences: number },
): CellLimits {
  const base = mobile ? BASE_LIMITS.mobile : BASE_LIMITS.desktop;
  if (lines === null) return base;

  const fit = (wanted: number, room: number, min: number) => {
    // Everything fits -> no marker needed. Otherwise one line goes to "+n".
    if (wanted <= room) return Math.max(min, wanted);
    return Math.max(min, room - 1);
  };

  // Room for events once the day's birthdays and the absences' guaranteed
  // base share (plus their possible marker) are set aside.
  const baseAbsences = Math.min(counts.absences, base.absences);
  const absenceMarker = counts.absences > base.absences ? 1 : 0;
  const events = fit(counts.events, lines - counts.birthdays - baseAbsences - absenceMarker, base.events);
  const shownEvents = Math.min(counts.events, events);
  const eventMarker = counts.events > shownEvents ? 1 : 0;

  const absences = fit(counts.absences, lines - shownEvents - eventMarker - counts.birthdays, base.absences);
  return { events, absences };
}

const px = (v: string) => Number.parseFloat(v) || 0;

/**
 * The height each of the calendar's six week rows gets when the grid
 * stretches to the bottom of its scroll container -- or null while that
 * can't be determined (no ResizeObserver, no scrolling ancestor, not laid
 * out yet), in which case callers fall back to BASE_LIMITS.
 *
 * Computed from the scroll container's own height and the grid's *top edge*
 * only, never from the grid's or a cell's rendered height: those grow with
 * the entries shown, so deriving the limits from them would feed back into
 * itself (more room -> more entries -> taller cells -> "more room").
 * Everything this reads is independent of how many entries the cells show.
 */
export function useStretchedRowHeight(gridRef: RefObject<HTMLElement | null>, gap: number): number | null {
  const [rowHeight, setRowHeight] = useState<number | null>(null);

  useEffect(() => {
    const grid = gridRef.current;
    const scroller = grid?.closest('main');
    if (!grid || !scroller || typeof ResizeObserver === 'undefined') return;

    const measure = () => {
      const scrollerStyle = getComputedStyle(scroller);
      // The grid sits inside the calendar card: its bottom padding + border
      // stay below the grid when the card reaches the scroller's content edge.
      const card = grid.parentElement;
      const cardStyle = card ? getComputedStyle(card) : null;
      const below =
        px(scrollerStyle.paddingBottom) + (cardStyle ? px(cardStyle.paddingBottom) + px(cardStyle.borderBottomWidth) : 0);
      // Grid top relative to the scroller's (unscrolled) top edge.
      const top = grid.getBoundingClientRect().top - scroller.getBoundingClientRect().top + scroller.scrollTop;
      const header = (grid.firstElementChild as HTMLElement | null)?.getBoundingClientRect().height ?? 0;
      const available = scroller.clientHeight - top - below - header - gap * 6;
      const next = available > 0 ? Math.floor(available / 6) : null;
      setRowHeight((prev) => (prev === next ? prev : next));
    };

    measure();
    // The scroller resizes with the viewport; a width change can also wrap
    // the toolbar above the grid and so move the grid's top edge.
    const observer = new ResizeObserver(measure);
    observer.observe(scroller);
    return () => observer.disconnect();
  }, [gridRef, gap]);

  return rowHeight;
}
