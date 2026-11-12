import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Stats } from './Stats';

const mockSetStatsRange = vi.fn();

// `threshold` is [mode, value, pending]; pending (members of the group who
// gave no response) defaults to 0.
function group(
  roleId: string,
  name: string,
  yes: number,
  counted: number,
  [thresholdMode, thresholdValue, pending = 0]: ['percent' | 'count', number, number?],
) {
  return { roleId, name, color: '#1565C0', yes, counted, pending, thresholdMode, thresholdValue };
}

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
      {
        id: 'ev1',
        title: 'Training',
        date: '2026-03-01',
        type: 'training',
        yes: 8,
        nominated: 10,
        pending: 0,
        enough: true,
        groups: [
          // complete -> check mark
          group('r_trainer', 'Trainer', 2, 2, ['count', 1]),
          // 6/8 = 75% >= 50% -> plain percentage
          group('r_member', 'Mitglied', 6, 8, ['percent', 50]),
        ],
      },
      {
        id: 'ev2',
        title: 'Spiel',
        date: '2026-03-05',
        type: 'match',
        yes: 5,
        nominated: 10,
        pending: 0,
        enough: false,
        groups: [
          // nobody responded -> no data
          group('r_trainer', 'Trainer', 0, 0, ['count', 1]),
          // 2/5 = 40% < 50% -> red
          group('r_member', 'Mitglied', 2, 5, ['percent', 50]),
          // 6/10 = 60%, but fewer than 8 attendees -> red (count mode)
          group('r_player', 'Spieler', 6, 10, ['count', 8]),
        ],
      },
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
  canDefinePresets = true,
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
    askConfirm: vi.fn(),
    toastMsg: vi.fn(),
    can: vi.fn().mockReturnValue(canDefinePresets),
  };
}

// Desktop by default; the matrix pins its count column only there.
vi.mock('@/layouts/useCompact', () => ({
  useCompact: vi.fn(() => false),
}));

vi.mock('@/context/AppContext', () => ({
  useApp: vi.fn(),
  useAppActions: vi.fn().mockReturnValue({ openEventDetail: vi.fn() }),
}));

// Mocked directly on the hooks module (not just a barrel re-export) --
// Stats.tsx imports `useStatsQuery`/`useAttendanceMatrixQuery` via this exact
// relative path, so this must match it (see the identical comment/pattern in
// PollsPage.test.tsx/NewsPage.test.tsx).
vi.mock('./hooks/useStatsQueries', () => ({
  useStatsQuery: vi.fn(),
  useAttendanceMatrixQuery: vi.fn(),
}));

// Also mocked directly (not just a barrel re-export), same reasoning as
// useStatsQueries above -- and it must be, since the real hook calls
// useQuery/useMutation and this test file renders <Stats /> without a
// QueryClientProvider.
vi.mock('./hooks/useStatsPreferencesActions', () => ({
  useStatsPreferencesActions: vi.fn(),
}));

import { useApp } from '@/context/AppContext';
import { useCompact } from '@/layouts/useCompact';
import { useAttendanceMatrixQuery, useStatsQuery } from './hooks/useStatsQueries';
import { useStatsPreferencesActions } from './hooks/useStatsPreferencesActions';
const mockUseApp = useApp as ReturnType<typeof vi.fn>;
const mockUseStatsQuery = useStatsQuery as ReturnType<typeof vi.fn>;
const mockUseMatrixQuery = useAttendanceMatrixQuery as ReturnType<typeof vi.fn>;
const mockUseStatsPreferencesActions = useStatsPreferencesActions as ReturnType<typeof vi.fn>;
const mockUseCompact = useCompact as ReturnType<typeof vi.fn>;

describe('Stats', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockUseCompact.mockReturnValue(false);
    mockUseStatsPreferencesActions.mockReturnValue({
      preferences: undefined,
      preferencesLoaded: false,
      presets: [],
      saveSelection: vi.fn(),
      createPreset: vi.fn(),
      creatingPreset: false,
      renamePreset: vi.fn(),
      deletePreset: vi.fn(),
    });
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

  it('no longer lists events below the quotas on the quota tab', () => {
    mockUseApp.mockReturnValue(makeApp(makeStats()));
    render(<Stats />);
    expect(screen.getByText('Quote pro Person')).toBeTruthy();
    expect(screen.queryByText('Training')).toBeNull();
    expect(screen.queryByText('Spiel')).toBeNull();
    expect(screen.queryAllByTestId('stats-event-row')).toHaveLength(0);
  });

  describe('events tab ("Termine")', () => {
    async function openEventsTab(stats: unknown = makeStats()) {
      mockUseApp.mockReturnValue(makeApp(stats));
      render(<Stats />);
      await userEvent.click(screen.getByRole('tab', { name: 'Termine' }));
    }

    it('lists the events of the range with their turnout, without the quota sections', async () => {
      await openEventsTab();
      expect(screen.getByRole('tab', { name: 'Termine' }).getAttribute('aria-selected')).toBe('true');
      const rows = screen.getAllByTestId('stats-event-row');
      expect(rows).toHaveLength(2);
      expect(within(rows[0]!).getByText('Training')).toBeTruthy();
      expect(within(rows[0]!).getByText(/8\/10 anwesend/)).toBeTruthy();
      expect(within(rows[1]!).getByText('Spiel')).toBeTruthy();
      expect(screen.queryByText('Quote pro Person')).toBeNull();
    });

    it('does not request the matrix for the events tab', async () => {
      await openEventsTab();
      const calls = mockUseMatrixQuery.mock.calls;
      expect(calls[calls.length - 1]![3]).toBe(false);
    });

    it('shows a check mark instead of a percentage for a complete group', async () => {
      await openEventsTab();
      const row = screen.getAllByTestId('stats-event-row')[0]!;
      const pill = within(row).getByLabelText('Trainer: vollständig anwesend (2 von 2)');
      expect(pill.getAttribute('data-state')).toBe('complete');
      expect(pill.textContent).not.toMatch(/%/);
      expect(pill.querySelector('svg')).toBeTruthy();
    });

    it('shows the percentage, not red, for an incomplete group at or above its threshold', async () => {
      await openEventsTab();
      const row = screen.getAllByTestId('stats-event-row')[0]!;
      const pill = within(row).getByLabelText('Mitglied: 75% anwesend (6 von 8)');
      expect(pill.getAttribute('data-state')).toBe('ok');
      expect(pill.textContent).toContain('75%');
      expect(pill.querySelector('svg')).toBeNull();
    });

    it('marks an incomplete group below its percentage threshold red', async () => {
      await openEventsTab();
      const row = screen.getAllByTestId('stats-event-row')[1]!;
      const pill = within(row).getByLabelText('Mitglied: 40% anwesend (2 von 5), unter dem Schwellenwert');
      expect(pill.getAttribute('data-state')).toBe('low');
      // Not colour alone: a low group also carries a warning icon.
      expect(pill.querySelector('svg')).toBeTruthy();
      expect(pill.textContent).toContain('40%');
    });

    it('marks an incomplete group below its absolute threshold red, still showing a percentage', async () => {
      await openEventsTab();
      const row = screen.getAllByTestId('stats-event-row')[1]!;
      const pill = within(row).getByLabelText('Spieler: 60% anwesend (6 von 10), unter dem Schwellenwert');
      expect(pill.getAttribute('data-state')).toBe('low');
      // Not colour alone: a low group also carries a warning icon.
      expect(pill.querySelector('svg')).toBeTruthy();
      expect(pill.textContent).toContain('60%');
    });

    it('shows a dash for a group without counted responses', async () => {
      await openEventsTab();
      const row = screen.getAllByTestId('stats-event-row')[1]!;
      const pill = within(row).getByLabelText('Trainer: keine Daten');
      expect(pill.getAttribute('data-state')).toBe('none');
      expect(pill.textContent).toContain('–');
    });

    it('renders one indicator per group, in the order the API returned them', async () => {
      await openEventsTab();
      const row = screen.getAllByTestId('stats-event-row')[1]!;
      const list = within(row).getByRole('list', { name: 'Anwesenheit je Gruppe für Spiel' });
      const names = within(list)
        .getAllByRole('listitem')
        .map((li) => li.getAttribute('aria-label')!.split(':')[0]);
      expect(names).toEqual(['Trainer', 'Mitglied', 'Spieler']);
    });

    it('renders an event without groups without a group list', async () => {
      const stats = makeStats();
      stats.events[0]!.groups = [];
      await openEventsTab(stats);
      const row = screen.getAllByTestId('stats-event-row')[0]!;
      expect(within(row).queryByRole('list')).toBeNull();
    });

    // A missing response counts as not present: the event line and the group
    // indicators are both measured against everyone expected.
    it('counts members without a response as not present, on the event line and in the groups', async () => {
      const stats = makeStats();
      stats.events[0] = {
        ...stats.events[0]!,
        yes: 3,
        nominated: 3,
        pending: 12,
        groups: [
          // both trainers answered yes -> complete
          group('r_trainer', 'Trainer', 2, 2, ['count', 1, 0]),
          // 3 yes, 12 silent -> 3 of 15, below "at least 8"
          group('r_player', 'Spieler', 3, 3, ['count', 8, 12]),
          // nobody answered -> 0 of 4, red; not "no data"
          group('r_helper', 'Betreuer', 0, 0, ['percent', 50, 4]),
        ],
      };
      await openEventsTab(stats);
      const row = screen.getAllByTestId('stats-event-row')[0]!;
      expect(within(row).getByText(/3\/15 anwesend/)).toBeTruthy();
      expect(within(row).queryByText(/3\/3 anwesend/)).toBeNull();

      expect(within(row).getByLabelText('Trainer: vollständig anwesend (2 von 2)').getAttribute('data-state')).toBe('complete');
      const players = within(row).getByLabelText('Spieler: 20% anwesend (3 von 15), unter dem Schwellenwert');
      expect(players.getAttribute('data-state')).toBe('low');
      expect(players.textContent).toContain('20%');
      const helpers = within(row).getByLabelText('Betreuer: 0% anwesend (0 von 4), unter dem Schwellenwert');
      expect(helpers.getAttribute('data-state')).toBe('low');
    });

    it('still shows the group indicators when nobody responded but members were expected', async () => {
      const stats = makeStats();
      stats.events[0] = {
        ...stats.events[0]!,
        yes: 0,
        nominated: 0,
        pending: 6,
        groups: [group('r_member', 'Mitglied', 0, 0, ['percent', 50, 6])],
      };
      await openEventsTab(stats);
      const row = screen.getAllByTestId('stats-event-row')[0]!;
      expect(within(row).getByText(/0\/6 anwesend/)).toBeTruthy();
      expect(within(row).getByRole('list')).toBeTruthy();
    });

    it('shows no group indicators for an event nobody was expected at', async () => {
      const stats = makeStats();
      stats.events[0] = {
        ...stats.events[0]!,
        yes: 0,
        nominated: 0,
        pending: 0,
        groups: [group('r_trainer', 'Trainer', 0, 0, ['count', 1]), group('r_member', 'Mitglied', 0, 0, ['percent', 50])],
      };
      await openEventsTab(stats);
      const row = screen.getAllByTestId('stats-event-row')[0]!;
      expect(within(row).getByText(/0\/0 anwesend/)).toBeTruthy();
      expect(within(row).queryByRole('list')).toBeNull();
      // The other event still shows its indicators.
      expect(within(screen.getAllByTestId('stats-event-row')[1]!).getByRole('list')).toBeTruthy();
    });

    it('shows the empty state when the range has no events', async () => {
      await openEventsTab({ ...makeStats(), events: [] });
      expect(screen.getByText('Noch keine vergangenen Termine')).toBeTruthy();
    });

    it('shows a spinner while the overview is loading', async () => {
      mockUseApp.mockReturnValue(makeApp(null));
      render(<Stats />);
      await userEvent.click(screen.getByRole('tab', { name: 'Termine' }));
      expect(screen.getByRole('status')).toBeTruthy();
    });
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

  it('renders the Quote/Matrix/Termine tabs in that order', () => {
    mockUseApp.mockReturnValue(makeApp(makeStats()));
    render(<Stats />);
    expect(screen.getAllByRole('tab').map((el) => el.textContent)).toEqual(['Quote', 'Matrix', 'Termine']);
  });

  it('puts the attendance count directly after the member name in the matrix', async () => {
    mockUseApp.mockReturnValue(makeApp(makeStats(), null, makeMatrix()));
    render(<Stats />);
    await userEvent.click(screen.getByRole('tab', { name: 'Matrix' }));

    // Header: Mitglied | Ja | 1.1. | 6.1. | 7.1.
    const headers = screen.getAllByRole('columnheader').map((th) => th.textContent);
    expect(headers).toEqual(['Mitglied', 'Ja', '1.1.', '6.1.', '7.1.']);

    // Every body row: name cell, then the count, then one cell per event.
    const bodyRows = screen.getAllByRole('row').slice(1);
    expect(bodyRows).toHaveLength(2);
    const peter = bodyRows[0]!;
    expect(within(peter).getByRole('rowheader').textContent).toContain('Peter');
    const peterCells = within(peter).getAllByRole('cell');
    expect(peterCells).toHaveLength(4);
    expect(peterCells[0]!.getAttribute('data-testid')).toBe('matrix-total');
    expect(peterCells[0]!.textContent).toBe('2');
    expect(within(bodyRows[1]!).getAllByRole('cell')[0]!.textContent).toBe('1');
    // The count is no longer the last cell.
    expect(peterCells[peterCells.length - 1]!.getAttribute('data-testid')).toBeNull();
  });

  it('pins the count column next to the pinned name column on desktop', async () => {
    mockUseApp.mockReturnValue(makeApp(makeStats(), null, makeMatrix()));
    render(<Stats />);
    await userEvent.click(screen.getByRole('tab', { name: 'Matrix' }));

    const nameCell = screen.getAllByRole('rowheader')[0]!;
    const nameStyle = getComputedStyle(nameCell);
    expect(nameStyle.position).toBe('sticky');
    expect(nameStyle.left).toBe('0px');
    expect(nameStyle.width).toBe('190px');

    // The count sits exactly at the right edge of the fixed-width name column.
    for (const el of [screen.getByTestId('matrix-total-header'), ...screen.getAllByTestId('matrix-total')]) {
      const style = getComputedStyle(el);
      expect(style.position).toBe('sticky');
      expect(style.left).toBe('190px');
    }
    // Event cells scroll.
    const eventCell = within(screen.getAllByRole('row')[1]!).getAllByRole('cell')[1]!;
    expect(getComputedStyle(eventCell).position).not.toBe('sticky');
  });

  it('pins only the name column on a phone', async () => {
    mockUseCompact.mockReturnValue(true);
    mockUseApp.mockReturnValue(makeApp(makeStats(), null, makeMatrix()));
    render(<Stats />);
    await userEvent.click(screen.getByRole('tab', { name: 'Matrix' }));

    expect(getComputedStyle(screen.getAllByRole('rowheader')[0]!).position).toBe('sticky');
    for (const el of [screen.getByTestId('matrix-total-header'), ...screen.getAllByTestId('matrix-total')]) {
      expect(getComputedStyle(el).position).not.toBe('sticky');
    }
    // Still the second column.
    expect(screen.getAllByRole('columnheader').map((th) => th.textContent)).toEqual(['Mitglied', 'Ja', '1.1.', '6.1.', '7.1.']);
  });

  it('switches to the matrix tab and renders the member × event grid', async () => {
    mockUseApp.mockReturnValue(makeApp(makeStats(), null, makeMatrix()));
    render(<Stats />);
    await userEvent.click(screen.getByRole('tab', { name: 'Matrix' }));

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
    await userEvent.click(screen.getByRole('tab', { name: 'Matrix' }));
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
    await userEvent.click(screen.getByRole('tab', { name: 'Matrix' }));
    expect(screen.getByRole('status')).toBeTruthy();
  });

  it('stays on the quota tab view by default', () => {
    mockUseApp.mockReturnValue(makeApp(makeStats()));
    render(<Stats />);
    expect(screen.getByText('Quote pro Person')).toBeTruthy();
  });
});

describe('Stats saved presets', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders a saved preset as a chip alongside the fixed presets', () => {
    mockUseApp.mockReturnValue(makeApp(null));
    mockUseStatsPreferencesActions.mockReturnValue({
      preferences: undefined,
      preferencesLoaded: false,
      presets: [{ id: 'p1', name: 'Saison 2026/27', from: '2026-08-01', to: '2027-05-31' }],
      saveSelection: vi.fn(),
      createPreset: vi.fn(),
      creatingPreset: false,
      renamePreset: vi.fn(),
      deletePreset: vi.fn(),
    });
    render(<Stats />);
    expect(screen.getByText('Saison 2026/27')).toBeTruthy();
  });

  it('selecting a saved preset applies its range and persists it as the new selection', async () => {
    const saveSelection = vi.fn();
    mockUseApp.mockReturnValue(makeApp(null));
    mockUseStatsPreferencesActions.mockReturnValue({
      preferences: undefined,
      preferencesLoaded: false,
      presets: [{ id: 'p1', name: 'Saison 2026/27', from: '2026-08-01', to: '2027-05-31' }],
      saveSelection,
      createPreset: vi.fn(),
      creatingPreset: false,
      renamePreset: vi.fn(),
      deletePreset: vi.fn(),
    });
    render(<Stats />);
    await userEvent.click(screen.getByText('Saison 2026/27'));
    expect(mockSetStatsRange).toHaveBeenCalledWith({ from: '2026-08-01', to: '2027-05-31' });
    expect(saveSelection).toHaveBeenCalledWith({ from: '2026-08-01', to: '2027-05-31' }, 'p1');
  });

  it('clicking "Neu" then saving creates a preset from the currently selected range', async () => {
    const createPreset = vi.fn().mockResolvedValue({ id: 'p2', name: 'Winterpause', from: '2026-01-01', to: '2026-03-01' });
    mockUseApp.mockReturnValue(makeApp(null, { from: '2026-01-01', to: '2026-03-01' }));
    mockUseStatsPreferencesActions.mockReturnValue({
      preferences: undefined,
      preferencesLoaded: false,
      presets: [],
      saveSelection: vi.fn(),
      createPreset,
      creatingPreset: false,
      renamePreset: vi.fn(),
      deletePreset: vi.fn(),
    });
    render(<Stats />);
    await userEvent.click(screen.getByText('Neu'));
    await userEvent.type(screen.getByPlaceholderText('z. B. Saison 2026/27'), 'Winterpause');
    await userEvent.click(screen.getByText('Speichern'));
    expect(createPreset).toHaveBeenCalledWith('Winterpause', { from: '2026-01-01', to: '2026-03-01' });
  });

  it('deleting a saved preset asks for confirmation before removing it', async () => {
    const deletePreset = vi.fn();
    const askConfirm = vi.fn(({ onConfirm }: { onConfirm: () => void }) => onConfirm());
    mockUseApp.mockReturnValue({ ...makeApp(null), askConfirm });
    mockUseStatsPreferencesActions.mockReturnValue({
      preferences: undefined,
      preferencesLoaded: false,
      presets: [{ id: 'p1', name: 'Saison 2026/27', from: '2026-08-01', to: '2027-05-31' }],
      saveSelection: vi.fn(),
      createPreset: vi.fn(),
      creatingPreset: false,
      renamePreset: vi.fn(),
      deletePreset,
    });
    render(<Stats />);
    await userEvent.click(screen.getByLabelText('„Saison 2026/27" löschen'));
    expect(askConfirm).toHaveBeenCalled();
    expect(deletePreset).toHaveBeenCalledWith('p1');
  });

  it('hides the "Neu" preset button and rename/delete affordances for a stats:read-only caller', () => {
    mockUseApp.mockReturnValue(makeApp(null, { from: '2026-01-01', to: '2026-03-01' }, null, false));
    mockUseStatsPreferencesActions.mockReturnValue({
      preferences: undefined,
      preferencesLoaded: false,
      presets: [{ id: 'p1', name: 'Saison 2026/27', from: '2026-08-01', to: '2027-05-31' }],
      saveSelection: vi.fn(),
      createPreset: vi.fn(),
      creatingPreset: false,
      renamePreset: vi.fn(),
      deletePreset: vi.fn(),
    });
    render(<Stats />);
    expect(screen.queryByText('Neu')).toBeNull();
    expect(screen.queryByLabelText('„Saison 2026/27" umbenennen')).toBeNull();
    expect(screen.queryByLabelText('„Saison 2026/27" löschen')).toBeNull();
    // The preset itself is still visible and selectable -- read access alone
    // must not hide a preset the caller already has.
    expect(screen.getByText('Saison 2026/27')).toBeTruthy();
  });

  it('selecting a saved preset still works for a stats:read-only caller', async () => {
    const saveSelection = vi.fn();
    mockUseApp.mockReturnValue(makeApp(null, null, null, false));
    mockUseStatsPreferencesActions.mockReturnValue({
      preferences: undefined,
      preferencesLoaded: false,
      presets: [{ id: 'p1', name: 'Saison 2026/27', from: '2026-08-01', to: '2027-05-31' }],
      saveSelection,
      createPreset: vi.fn(),
      creatingPreset: false,
      renamePreset: vi.fn(),
      deletePreset: vi.fn(),
    });
    render(<Stats />);
    await userEvent.click(screen.getByText('Saison 2026/27'));
    expect(mockSetStatsRange).toHaveBeenCalledWith({ from: '2026-08-01', to: '2027-05-31' });
    expect(saveSelection).toHaveBeenCalledWith({ from: '2026-08-01', to: '2027-05-31' }, 'p1');
  });

  it('hydrates the range from the caller last-saved preferences on first load', () => {
    mockUseApp.mockReturnValue(makeApp(null, null));
    mockUseStatsPreferencesActions.mockReturnValue({
      preferences: { range: { from: '2026-02-01', to: '2026-04-01' }, presetId: null },
      preferencesLoaded: true,
      presets: [],
      saveSelection: vi.fn(),
      createPreset: vi.fn(),
      creatingPreset: false,
      renamePreset: vi.fn(),
      deletePreset: vi.fn(),
    });
    render(<Stats />);
    expect(mockSetStatsRange).toHaveBeenCalledWith({ from: '2026-02-01', to: '2026-04-01' });
  });

  it('does not overwrite an already-selected range with the saved preferences', () => {
    mockUseApp.mockReturnValue(makeApp(null, { from: '2026-05-01', to: '2026-06-01' }));
    mockUseStatsPreferencesActions.mockReturnValue({
      preferences: { range: { from: '2026-02-01', to: '2026-04-01' }, presetId: null },
      preferencesLoaded: true,
      presets: [],
      saveSelection: vi.fn(),
      createPreset: vi.fn(),
      creatingPreset: false,
      renamePreset: vi.fn(),
      deletePreset: vi.fn(),
    });
    render(<Stats />);
    expect(mockSetStatsRange).not.toHaveBeenCalled();
  });
});
