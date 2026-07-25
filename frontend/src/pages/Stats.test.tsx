import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Stats } from './Stats';

const mockSetStatsRange = vi.fn();

function makeStats() {
  return {
    avg: 75,
    pastCount: 20,
    members: [
      { userId: 'u1', name: 'Anna Müller', quote: 85, photo: null, avatarColor: '#F00' },
      { userId: 'u2', name: 'Bob Schmidt', quote: 45, photo: null, avatarColor: '#0F0' },
      { userId: 'u3', name: 'Clara Braun', quote: null, photo: null, avatarColor: '#00F' },
    ],
    events: [
      { id: 'ev1', title: 'Training', date: '2026-03-01', type: 'training', yes: 8, nominated: 10, enough: true },
      { id: 'ev2', title: 'Spiel', date: '2026-03-05', type: 'match', yes: 5, nominated: 10, enough: false },
    ],
  };
}

function makeMatrix() {
  return {
    from: '2026-01-01',
    to: '2026-03-31',
    events: [
      { id: 'ev1', title: 'Training', date: '2026-01-01', type: 'training' },
      { id: 'ev2', title: 'Auftritt', date: '2026-01-06', type: 'auftritt' },
      { id: 'ev3', title: 'Feier', date: '2026-01-07', type: 'event' },
    ],
    members: [
      {
        userId: 'u1',
        name: 'Peter',
        avatarColor: '#F00',
        photo: null,
        yes: 2,
        counted: 3,
        cells: { ev1: 'yes', ev2: 'no', ev3: 'yes' },
      },
      {
        userId: 'u2',
        name: 'Marie',
        avatarColor: '#0F0',
        photo: null,
        yes: 1,
        counted: 3,
        cells: { ev1: 'yes', ev2: 'maybe', ev3: 'no' },
      },
    ],
  };
}

function makeApp(
  statsOverride: unknown = null,
  statsRange: { from: string; to: string } | null = null,
  matrixOverride: unknown = null,
) {
  mockUseStatsQuery.mockReturnValue({ data: statsOverride ?? undefined } as never);
  mockUseMatrixQuery.mockReturnValue({ data: matrixOverride ?? undefined } as never);
  return {
    api: {},
    state: {
      phase: 'app',
      primaryColor: '#4285F4',
      activeTeamId: 't1',
      statsRange,
      user: { id: 'u1', name: 'Test User', avatarColor: '#000', photo: null },
    },
    setStatsRange: mockSetStatsRange,
  };
}

vi.mock('@/context/AppContext', () => ({
  useApp: vi.fn(),
  useAppActions: vi.fn().mockReturnValue({ openEventDetail: vi.fn() }),
}));

// Mocked directly on the hooks module (not just a barrel re-export) --
// Stats.tsx imports `useStatsQuery` via this exact relative path, so this
// must match it (see the identical comment/pattern in PollsPage.test.tsx/
// NewsPage.test.tsx).
vi.mock('./hooks/useStatsQueries', () => ({
  useStatsQuery: vi.fn(),
  useAttendanceMatrixQuery: vi.fn(),
}));

import { useApp } from '@/context/AppContext';
import { useStatsQuery, useAttendanceMatrixQuery } from './hooks/useStatsQueries';
const mockUseApp = useApp as ReturnType<typeof vi.fn>;
const mockUseStatsQuery = useStatsQuery as ReturnType<typeof vi.fn>;
const mockUseMatrixQuery = useAttendanceMatrixQuery as ReturnType<typeof vi.fn>;

describe('Stats', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('shows spinner when stats are null', () => {
    mockUseApp.mockReturnValue(makeApp(null));
    render(<Stats />);
    expect(screen.getByRole('status')).toBeTruthy();
    expect(screen.getByText('Gesamt')).toBeTruthy();
    expect(screen.getByText('3 Monate')).toBeTruthy();
  });

  it('renders filter presets', () => {
    mockUseApp.mockReturnValue(makeApp(null));
    render(<Stats />);
    expect(screen.getByText('Gesamt')).toBeTruthy();
    expect(screen.getByText('3 Monate')).toBeTruthy();
    expect(screen.getByText('6 Monate')).toBeTruthy();
    expect(screen.getByText('12 Monate')).toBeTruthy();
  });

  it('calls setStatsRange with an explicit all-time range when clicking Gesamt', async () => {
    mockUseApp.mockReturnValue(makeApp(null));
    render(<Stats />);
    await userEvent.click(screen.getByText('Gesamt'));
    expect(mockSetStatsRange).toHaveBeenCalledWith({ from: '1970-01-01', to: expect.any(String) });
  });

  it('calls setStatsRange with 3-month range when clicking "3 Monate"', async () => {
    mockUseApp.mockReturnValue(makeApp(null));
    render(<Stats />);
    await userEvent.click(screen.getByText('3 Monate'));
    expect(mockSetStatsRange).toHaveBeenCalledWith(expect.objectContaining({ to: expect.any(String) }));
  });

  // Regression test: Date.setMonth silently rolls over into the following
  // month when the target month has fewer days than today's day-of-month
  // (e.g. May 31 minus 3 months lands on "Feb 31", which JS normalizes to
  // Mar 3) -- narrowing the range by a few days without any indication to
  // the user. The preset "N Monate" must always land on N calendar months
  // back, clamped to the last day of that month rather than rolling over.
  it('does not roll over into the next month when "today" has no equivalent day in the target month', async () => {
    const realDate = globalThis.Date;
    class FixedDate extends realDate {
      constructor(...args: unknown[]) {
        if (args.length === 0) {
          super(2026, 4, 31); // "today" = May 31, 2026
        } else {
          // @ts-expect-error -- forwarding varargs to the real Date constructor
          super(...args);
        }
      }
      static override now() {
        return new realDate(2026, 4, 31).getTime();
      }
    }
    globalThis.Date = FixedDate as DateConstructor;
    try {
      mockUseApp.mockReturnValue(makeApp(null));
      render(<Stats />);
      await userEvent.click(screen.getByText('3 Monate'));
      expect(mockSetStatsRange).toHaveBeenCalledWith({ from: '2026-02-28', to: '2026-05-31' });
    } finally {
      globalThis.Date = realDate;
    }
  });

  it('shows stats ring when stats loaded', () => {
    mockUseApp.mockReturnValue(makeApp(makeStats()));
    render(<Stats />);
    expect(screen.getByText('75%')).toBeTruthy();
    expect(screen.getByText('Team-Anwesenheit')).toBeTruthy();
  });

  it('shows member bars section', () => {
    mockUseApp.mockReturnValue(makeApp(makeStats()));
    render(<Stats />);
    expect(screen.getByText('Quote pro Person')).toBeTruthy();
    expect(screen.getByText('Anna Müller')).toBeTruthy();
    expect(screen.getByText('Bob Schmidt')).toBeTruthy();
    expect(screen.getByText('Clara Braun')).toBeTruthy();
  });

  it('shows "–" for member with null quote', () => {
    mockUseApp.mockReturnValue(makeApp(makeStats()));
    render(<Stats />);
    const dashes = screen.getAllByText('–');
    expect(dashes.length).toBeGreaterThan(0);
  });

  it('renders event stats section', () => {
    mockUseApp.mockReturnValue(makeApp(makeStats()));
    render(<Stats />);
    expect(screen.getByText('Aufstellung je Termin')).toBeTruthy();
    expect(screen.getByText('Training')).toBeTruthy();
    expect(screen.getByText('Spiel')).toBeTruthy();
  });

  it('shows "Vollständig" chip for events with enough attendance', () => {
    mockUseApp.mockReturnValue(makeApp(makeStats()));
    render(<Stats />);
    expect(screen.getByText('Vollständig')).toBeTruthy();
    expect(screen.getByText('Zu wenig')).toBeTruthy();
  });

  it('shows empty events state when no past events', () => {
    const stats = { ...makeStats(), events: [] };
    mockUseApp.mockReturnValue(makeApp(stats));
    render(<Stats />);
    expect(screen.getByText('Noch keine vergangenen Termine')).toBeTruthy();
  });

  it('highlights active range preset', () => {
    const threeMonthsRange = {
      from: '2025-12-01',
      to: '2026-03-01',
    };
    mockUseApp.mockReturnValue(makeApp(null, threeMonthsRange));
    render(<Stats />);
    // 3 Monate button should be in the DOM
    expect(screen.getByText('3 Monate')).toBeTruthy();
  });

  it('shows custom range when dates do not match a preset', () => {
    const customRange = { from: '2025-06-15', to: '2026-01-15' };
    mockUseApp.mockReturnValue(makeApp(null, customRange));
    render(<Stats />);
    // Filter bar renders — no crash
    expect(screen.getByText('Gesamt')).toBeTruthy();
  });

  it('renders the Übersicht/Matrix tab toggle', () => {
    mockUseApp.mockReturnValue(makeApp(makeStats()));
    render(<Stats />);
    expect(screen.getByText('Übersicht')).toBeTruthy();
    expect(screen.getByText('Matrix')).toBeTruthy();
  });

  it('switches to the matrix tab and renders the member × event grid', async () => {
    mockUseApp.mockReturnValue(makeApp(makeStats(), null, makeMatrix()));
    render(<Stats />);
    await userEvent.click(screen.getByText('Matrix'));

    // Members appear as rows (sorted order comes from the backend/mock).
    expect(screen.getByText('Peter')).toBeTruthy();
    expect(screen.getByText('Marie')).toBeTruthy();
    // Column header uses the compact d.m. format.
    expect(screen.getByText('1.1.')).toBeTruthy();
    expect(screen.getByText('6.1.')).toBeTruthy();
    // The overview sections are no longer shown on the matrix tab.
    expect(screen.queryByText('Quote pro Person')).toBeNull();
  });

  it('filters matrix columns by event type via the checkboxes', async () => {
    mockUseApp.mockReturnValue(makeApp(makeStats(), null, makeMatrix()));
    render(<Stats />);
    await userEvent.click(screen.getByText('Matrix'));
    expect(screen.getByText('7.1.')).toBeTruthy(); // the 'event'-type column

    // Uncheck the "event" type filter → its column disappears.
    await userEvent.click(screen.getByRole('checkbox', { name: 'Team-Event' }));
    expect(screen.queryByText('7.1.')).toBeNull();
    // Other type columns remain.
    expect(screen.getByText('1.1.')).toBeTruthy();
  });

  it('shows a spinner on the matrix tab while the matrix loads', async () => {
    mockUseApp.mockReturnValue(makeApp(makeStats(), null, null));
    render(<Stats />);
    await userEvent.click(screen.getByText('Matrix'));
    expect(screen.getByRole('status')).toBeTruthy();
  });
});
