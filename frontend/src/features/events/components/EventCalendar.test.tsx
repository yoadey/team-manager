import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, within } from '@testing-library/react';
import { EventCalendar } from './EventCalendar';
import type { TeamEvent } from '../types';

vi.mock('@/context/AppContext', () => ({
  useApp: vi.fn(),
}));

vi.mock('@/layouts/useCompact', () => ({
  useCompact: vi.fn(() => false),
}));

// The stretched row height comes from real layout (scroll container height,
// grid position) that jsdom doesn't compute -- stubbed here so a test can
// set it directly; everything else in calendarLayout (the limit maths) stays
// real. Defaults to null = "can't measure", i.e. the fixed base limits.
vi.mock('./calendarLayout', async (importOriginal) => {
  const mod = await importOriginal<typeof import('./calendarLayout')>();
  return { ...mod, useStretchedRowHeight: vi.fn(() => null) };
});

vi.mock('../hooks/useEventQueries', () => ({
  useEventsQuery: vi.fn(),
}));

vi.mock('../hooks/useAbsenceQueries', () => ({
  useAbsencesQuery: vi.fn(),
}));

vi.mock('@/features/members', () => ({
  useMembersQuery: vi.fn(),
}));

import { useApp } from '@/context/AppContext';
import { useCompact } from '@/layouts/useCompact';
import { useStretchedRowHeight } from './calendarLayout';
import { useEventsQuery } from '../hooks/useEventQueries';
import { useAbsencesQuery } from '../hooks/useAbsenceQueries';
import { useMembersQuery } from '@/features/members';
import type { Member } from '@/features/members';
const mockUseApp = vi.mocked(useApp);
const mockUseEventsQuery = vi.mocked(useEventsQuery);
const mockUseAbsencesQuery = vi.mocked(useAbsencesQuery);
const mockUseMembersQuery = vi.mocked(useMembersQuery);
const mockRowHeight = vi.mocked(useStretchedRowHeight);
const mockUseCompact = vi.mocked(useCompact);

function makeMember(overrides: Partial<Member> = {}): Member {
  return {
    membershipId: 'ms1',
    userId: 'u1',
    name: 'Alice Example',
    email: 'alice@example.com',
    phone: '',
    birthday: '1990-03-10',
    address: '',
    avatarColor: '#000000',
    photo: null,
    group: '',
    title: '',
    roles: [],
    joinedAt: '2020-01-01',
    primaryRole: null,
    perms: { events: 'read', members: 'read', finances: 'read', news: 'read', polls: 'read', settings: 'read', stats: 'read' },
    excludeFromStats: false,
    ...overrides,
  };
}

function makeEvent(overrides: Partial<TeamEvent> = {}): TeamEvent {
  return {
    id: 'ev1',
    teamId: 't1',
    type: 'training',
    title: 'Training',
    date: '2026-03-10',
    multiDayEndDate: null,
    location: '',
    note: '',
    meetTime: null,
    startTime: '18:00',
    endTime: null,
    meetTimeMandatory: false,
    responseMode: 'opt_in',
    recurring: false,
    seriesId: null,
    status: 'active',
    cancelLeadMinutes: null,
    excludeFromStats: false,
    summary: { yes: 0, no: 0, maybe: 0, pending: 0, notNominated: 0, nominated: 0, total: 0 },
    myStatus: 'pending',
    myAuto: false,
    myReason: '',
    ...overrides,
  };
}

function makeApp(
  overrides: {
    events?: TeamEvent[];
    calMonth?: Date | null;
    calShowAbsences?: boolean;
    absences?: { from: string; to: string; name: string; roleColor: string }[];
    members?: Member[];
    canSeeBirthdays?: boolean;
    canCreateEvent?: boolean;
  } = {},
) {
  const openEventDetail = vi.fn();
  const openEventForm = vi.fn();
  const setState = vi.fn();
  const toggleCalAbsences = vi.fn();
  const canSeeBirthdays = overrides.canSeeBirthdays ?? true;
  const canCreateEvent = overrides.canCreateEvent ?? true;
  const can = vi.fn((module: string, level?: string) => {
    if (module === 'members' && level === 'write') return canSeeBirthdays;
    if (module === 'events' && level === 'write') return canCreateEvent;
    return true;
  });
  const app = {
    api: {},
    state: {
      primaryColor: 'blue',
      activeTeamId: 't1',
      calMonth: overrides.calMonth ?? new Date(2026, 2, 1), // March 2026
      calShowAbsences: overrides.calShowAbsences ?? false,
    },
    can,
    openEventDetail,
    openEventForm,
    setState,
    toggleCalAbsences,
  };
  mockUseApp.mockReturnValue(app as unknown as ReturnType<typeof useApp>);
  mockUseEventsQuery.mockReturnValue({ data: overrides.events ?? [] } as never);
  mockUseAbsencesQuery.mockReturnValue({ data: overrides.absences ?? [] } as never);
  mockUseMembersQuery.mockReturnValue({ data: overrides.members ?? [] } as never);
  return app;
}

describe('EventCalendar', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockRowHeight.mockReturnValue(null);
    mockUseCompact.mockReturnValue(false);
  });

  describe('filling the available space', () => {
    const eventsOn = (date: string, n: number) =>
      Array.from({ length: n }, (_, i) =>
        makeEvent({ id: 'ev' + i, title: 'Termin ' + (i + 1), date, startTime: `1${i % 10}:00` }),
      );
    const chipsOnDay = () => screen.queryAllByText(/Termin \d+/);

    it('has no width cap and lays out as a growing flex column', () => {
      makeApp();
      render(<EventCalendar />);
      const root = screen.getByTestId('calendar-root');
      const style = getComputedStyle(root);
      expect(['', 'none']).toContain(style.maxWidth);
      expect(style.display).toBe('flex');
      expect(style.flexDirection).toBe('column');
      expect(style.flexGrow).toBe('1');
    });

    it('lets the card and the grid take the remaining height', () => {
      makeApp();
      render(<EventCalendar />);
      const grid = screen.getByTestId('calendar-grid');
      expect(getComputedStyle(grid).flexGrow).toBe('1');
      expect(getComputedStyle(grid.parentElement!).flexGrow).toBe('1');
      expect(getComputedStyle(grid.parentElement!).display).toBe('flex');
    });

    it('measures against the grid with the desktop gap, and with no gap on mobile', () => {
      makeApp();
      const { unmount } = render(<EventCalendar />);
      expect(mockRowHeight).toHaveBeenLastCalledWith(expect.objectContaining({ current: expect.anything() }), 6);
      unmount();

      mockUseCompact.mockReturnValue(true);
      render(<EventCalendar />);
      expect(mockRowHeight).toHaveBeenLastCalledWith(expect.anything(), 0);
    });

    it('keeps the fixed limit of 3 events + "+n" when the height cannot be measured', () => {
      makeApp({ events: eventsOn('2026-03-10', 5) });
      render(<EventCalendar />);
      expect(chipsOnDay()).toHaveLength(3);
      expect(screen.getByText('+2')).toBeTruthy();
    });

    it('keeps the fixed limit at the minimum row height', () => {
      mockRowHeight.mockReturnValue(76);
      makeApp({ events: eventsOn('2026-03-10', 5) });
      render(<EventCalendar />);
      expect(chipsOnDay()).toHaveLength(3);
      expect(screen.getByText('+2')).toBeTruthy();
    });

    it('shows every event of a day once the stretched row has room for them', () => {
      mockRowHeight.mockReturnValue(170); // 8 lines
      makeApp({ events: eventsOn('2026-03-10', 5) });
      render(<EventCalendar />);
      expect(chipsOnDay()).toHaveLength(5);
      expect(screen.queryByText(/^\+\d+$/)).toBeNull();
    });

    it('still summarises what does not fit a stretched row with "+n"', () => {
      mockRowHeight.mockReturnValue(170); // 8 lines -> 7 chips + marker
      makeApp({ events: eventsOn('2026-03-10', 12) });
      render(<EventCalendar />);
      expect(chipsOnDay()).toHaveLength(7);
      expect(screen.getByText('+5')).toBeTruthy();
    });

    it('raises the absence limit too when a stretched row has room', () => {
      mockRowHeight.mockReturnValue(170);
      makeApp({
        calShowAbsences: true,
        absences: ['Anna A', 'Bob B', 'Carla C', 'Dora D'].map((name) => ({
          from: '2026-03-10',
          to: '2026-03-10',
          name,
          roleColor: '#111111',
        })),
      });
      render(<EventCalendar />);
      for (const first of ['Anna', 'Bob', 'Carla', 'Dora']) expect(screen.getByText(first)).toBeTruthy();
      expect(screen.queryByText(/^\+\d+ /)).toBeNull();
    });

    it('uses the mobile base limit of 2 events when unmeasured on a compact screen', () => {
      mockUseCompact.mockReturnValue(true);
      makeApp({ events: eventsOn('2026-03-10', 5) });
      render(<EventCalendar />);
      expect(chipsOnDay()).toHaveLength(2);
      expect(screen.getByText('+3')).toBeTruthy();
    });
  });

  it('renders the month label for the active calMonth', () => {
    makeApp();
    render(<EventCalendar />);
    expect(screen.getByText(/März 2026/i)).toBeTruthy();
  });

  it('renders an event chip for a day with an event', () => {
    makeApp({ events: [makeEvent({ date: '2026-03-10', title: 'Training' })] });
    render(<EventCalendar />);
    expect(screen.getByText((content) => content.includes('Training'))).toBeTruthy();
  });

  it('clicking an event chip calls app.openEventDetail with the event id', () => {
    const app = makeApp({ events: [makeEvent({ id: 'ev42', date: '2026-03-10', title: 'Training' })] });
    render(<EventCalendar />);
    const chip = screen.getByText((content) => content.includes('Training')).closest('button');
    expect(chip).toBeTruthy();
    fireEvent.click(chip!);
    expect(app.openEventDetail).toHaveBeenCalledWith('ev42');
  });

  it('renders a multi-day event on every day it spans, each with a day-N-of-M indicator', () => {
    makeApp({
      events: [
        makeEvent({ id: 'camp1', date: '2026-03-10', multiDayEndDate: '2026-03-12', title: 'Trainingslager' }),
      ],
    });
    render(<EventCalendar />);
    expect(screen.getByText((c) => c.includes('Trainingslager') && c.includes('1/3'))).toBeTruthy();
    expect(screen.getByText((c) => c.includes('Trainingslager') && c.includes('2/3'))).toBeTruthy();
    expect(screen.getByText((c) => c.includes('Trainingslager') && c.includes('3/3'))).toBeTruthy();
  });

  it('does not show a day-N-of-M indicator for a single-day event', () => {
    makeApp({ events: [makeEvent({ date: '2026-03-10', title: 'Training' })] });
    render(<EventCalendar />);
    expect(screen.queryByText((c) => c.includes('1/1'))).toBeFalsy();
  });

  it('clicking the next-month button advances calMonth by one month', () => {
    const app = makeApp({ calMonth: new Date(2026, 2, 1) });
    render(<EventCalendar />);
    const next = screen.getByLabelText(/Nächster Monat|next month/i);
    fireEvent.click(next);
    expect(app.setState).toHaveBeenCalledWith({ calMonth: new Date(2026, 3, 1) });
  });

  it('clicking the previous-month button goes back one month', () => {
    const app = makeApp({ calMonth: new Date(2026, 2, 1) });
    render(<EventCalendar />);
    const prev = screen.getByLabelText(/Vorheriger Monat|previous month/i);
    fireEvent.click(prev);
    expect(app.setState).toHaveBeenCalledWith({ calMonth: new Date(2026, 1, 1) });
  });

  it('double-clicking an in-month day cell opens the create sheet pre-dated to that day', () => {
    const app = makeApp({ calMonth: new Date(2026, 2, 1) }); // March 2026
    render(<EventCalendar />);
    const cell = screen.getByText('15').parentElement!;
    fireEvent.doubleClick(cell);
    expect(app.openEventForm).toHaveBeenCalledWith(null, '2026-03-15');
  });

  it('double-clicking a day cell does nothing for a member without events:write', () => {
    const app = makeApp({ calMonth: new Date(2026, 2, 1), canCreateEvent: false });
    render(<EventCalendar />);
    const cell = screen.getByText('15').parentElement!;
    fireEvent.doubleClick(cell);
    expect(app.openEventForm).not.toHaveBeenCalled();
  });

  it('double-clicking an event chip opens only the event, not the day create sheet', () => {
    const app = makeApp({ events: [makeEvent({ id: 'ev42', date: '2026-03-10', title: 'Training' })] });
    render(<EventCalendar />);
    const chip = screen.getByText((content) => content.includes('Training')).closest('button')!;
    fireEvent.doubleClick(chip);
    expect(app.openEventForm).not.toHaveBeenCalled();
  });

  it('double-clicking an absence chip does not open the day create sheet', () => {
    const app = makeApp({
      calMonth: new Date(2026, 2, 1),
      calShowAbsences: true,
      absences: [{ from: '2026-03-10', to: '2026-03-10', name: 'Max Mustermann', roleColor: '#123456' }],
    });
    render(<EventCalendar />);
    fireEvent.doubleClick(screen.getByText('Max'));
    expect(app.openEventForm).not.toHaveBeenCalled();
  });

  it('double-clicking a birthday chip does not open the day create sheet', () => {
    const app = makeApp({
      calMonth: new Date(2026, 2, 1),
      canSeeBirthdays: true,
      members: [makeMember({ membershipId: 'ms1', name: 'Alice Example', birthday: '1990-03-10' })],
    });
    render(<EventCalendar />);
    fireEvent.doubleClick(screen.getByText('Alice Example'));
    expect(app.openEventForm).not.toHaveBeenCalled();
  });

  it('exposes a keyboard equivalent (Enter/Space) for the day create sheet, labelled with the day', () => {
    const app = makeApp({ calMonth: new Date(2026, 2, 1) });
    render(<EventCalendar />);
    const cell = screen.getByText('15').parentElement!;
    expect(cell).toHaveAttribute('role', 'button');
    expect(cell.getAttribute('aria-label')).toMatch(/15/);

    fireEvent.keyDown(cell, { key: 'Enter' });
    expect(app.openEventForm).toHaveBeenCalledWith(null, '2026-03-15');

    app.openEventForm.mockClear();
    fireEvent.keyDown(cell, { key: ' ' });
    expect(app.openEventForm).toHaveBeenCalledWith(null, '2026-03-15');
  });

  it('does not expose a keyboard/button role on a day cell without events:write', () => {
    makeApp({ calMonth: new Date(2026, 2, 1), canCreateEvent: false });
    render(<EventCalendar />);
    const cell = screen.getByText('15').parentElement!;
    expect(cell).not.toHaveAttribute('role', 'button');
    expect(cell).not.toHaveAttribute('tabindex');
  });

  it('toggling the "show absences" checkbox calls app.toggleCalAbsences', () => {
    const app = makeApp();
    render(<EventCalendar />);
    const checkbox = screen.getByRole('checkbox');
    fireEvent.click(checkbox);
    expect(app.toggleCalAbsences).toHaveBeenCalledTimes(1);
  });

  // Regression test: the day-by-day absence expansion used to add a fixed
  // 86400000ms instead of incrementing the calendar day, so on a DST "fall
  // back" transition it landed on the same date twice (pushing a duplicate
  // chip into that day's cell) and skipped the range's last day (leaving its
  // cell empty) -- even though the total chip count across the month stayed
  // the same either way. 2026-10-25 is a DST transition (Europe/Berlin).
  it('renders exactly one absence chip per day of a range spanning a DST transition', () => {
    vi.stubEnv('TZ', 'Europe/Berlin');
    try {
      makeApp({
        calMonth: new Date(2026, 9, 1), // October 2026
        calShowAbsences: true,
        absences: [{ from: '2026-10-24', to: '2026-10-27', name: 'Max Mustermann', roleColor: '#123456' }],
      });
      render(<EventCalendar />);
      for (const day of [24, 25, 26, 27]) {
        const cell = screen.getByText(String(day)).parentElement!;
        expect(within(cell).getAllByText('Max')).toHaveLength(1);
      }
    } finally {
      vi.unstubAllEnvs();
    }
  });

  // Regression test: the "+N absent" overflow badge used to derive its
  // abbreviation by slicing the first 3 characters off t('events.absent')
  // (the LONG-form translation), rather than using a dedicated short-form
  // translation key -- fragile for any locale whose translation doesn't
  // happen to front-load a recognizable 3-character stem, and liable to cut
  // a translated string mid-character. Use the dedicated
  // events.absentShort key instead. Mocking absentShort to a value whose
  // first 3 characters DIFFER from t('events.absent')'s first 3 characters
  // proves the badge renders the dedicated key's full value rather than
  // deriving it from the long form -- with the real (unmocked) strings the
  // two code paths happen to produce the same output, so a plain string
  // assertion wouldn't distinguish old from new behavior.
  it('labels the absence overflow badge with the dedicated short-form translation, not a slice of the long form', async () => {
    const i18n = await import('@/i18n');
    const realT = i18n.t;
    const spy = vi.spyOn(i18n, 't').mockImplementation((key: string, params?: Record<string, string | number>) => {
      if (key === 'events.absentShort') return 'XYZ';
      if (key === 'events.absent') return 'not-this-value';
      return realT(key, params);
    });
    try {
      makeApp({
        calMonth: new Date(2026, 9, 1),
        calShowAbsences: true,
        absences: [
          { from: '2026-10-05', to: '2026-10-05', name: 'Anna Müller', roleColor: '#111111' },
          { from: '2026-10-05', to: '2026-10-05', name: 'Bob Schmidt', roleColor: '#222222' },
          { from: '2026-10-05', to: '2026-10-05', name: 'Carla Weiß', roleColor: '#333333' },
        ],
      });
      render(<EventCalendar />);
      expect(screen.getByText('+1 XYZ')).toBeTruthy();
      expect(screen.queryByText(/not/)).toBeNull();
    } finally {
      spy.mockRestore();
    }
  });

  describe('birthdays', () => {
    it('renders a birthday entry on the correct day for a user with permission', () => {
      makeApp({
        calMonth: new Date(2026, 2, 1), // March 2026
        canSeeBirthdays: true,
        members: [makeMember({ membershipId: 'ms1', name: 'Alice Example', birthday: '1990-03-10' })],
      });
      render(<EventCalendar />);
      const cell = screen.getByText('10').parentElement!;
      expect(within(cell).getByText('Alice Example')).toBeTruthy();
    });

    it('renders no birthday entry for a user without permission to see it', () => {
      makeApp({
        calMonth: new Date(2026, 2, 1),
        canSeeBirthdays: false,
        members: [makeMember({ membershipId: 'ms1', name: 'Alice Example', birthday: '1990-03-10' })],
      });
      render(<EventCalendar />);
      expect(screen.queryByText('Alice Example')).toBeNull();
      // Also verifies the query is skipped entirely rather than fetched-then-filtered.
      expect(mockUseMembersQuery).toHaveBeenCalledWith(expect.anything(), null);
    });

    // The birthday recurs every year without a stored per-year event row --
    // verify the same member's birthday synthesizes into a visible entry
    // when the calendar is navigated to a different year.
    it('recurs every year: the same birthday appears when viewing a later year', () => {
      makeApp({
        calMonth: new Date(2030, 2, 1), // March 2030
        canSeeBirthdays: true,
        members: [makeMember({ membershipId: 'ms1', name: 'Alice Example', birthday: '1990-03-10' })],
      });
      render(<EventCalendar />);
      const cell = screen.getByText('10').parentElement!;
      expect(within(cell).getByText('Alice Example')).toBeTruthy();
    });

    it('does not render a birthday entry on a day it does not fall on', () => {
      makeApp({
        calMonth: new Date(2026, 2, 1),
        canSeeBirthdays: true,
        members: [makeMember({ membershipId: 'ms1', name: 'Alice Example', birthday: '1990-03-10' })],
      });
      render(<EventCalendar />);
      const otherCell = screen.getByText('11').parentElement!;
      expect(within(otherCell).queryByText('Alice Example')).toBeNull();
    });
  });
});
