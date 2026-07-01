import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { EventCalendar } from './EventCalendar';
import type { TeamEvent } from '../types';

vi.mock('@/context/AppContext', () => ({
  useApp: vi.fn(),
}));

vi.mock('@/layouts/useCompact', () => ({
  useCompact: vi.fn(() => false),
}));

import { useApp } from '@/context/AppContext';
const mockUseApp = vi.mocked(useApp);

function makeEvent(overrides: Partial<TeamEvent> = {}): TeamEvent {
  return {
    id: 'ev1',
    teamId: 't1',
    type: 'training',
    title: 'Training',
    date: '2026-03-10',
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
    summary: { yes: 0, no: 0, maybe: 0, pending: 0, notNominated: 0, nominated: 0, total: 0 },
    myStatus: 'pending',
    myAuto: false,
    myReason: '',
    ...overrides,
  };
}

function makeApp(overrides: { events?: TeamEvent[]; calMonth?: Date | null; calShowAbsences?: boolean } = {}) {
  const openEventDetail = vi.fn();
  const setState = vi.fn();
  const toggleCalAbsences = vi.fn();
  const app = {
    state: {
      primaryColor: 'blue',
      calMonth: overrides.calMonth ?? new Date(2026, 2, 1), // March 2026
      calShowAbsences: overrides.calShowAbsences ?? false,
      events: overrides.events ?? [],
      absences: [],
    },
    openEventDetail,
    setState,
    toggleCalAbsences,
  };
  mockUseApp.mockReturnValue(app as unknown as ReturnType<typeof useApp>);
  return app;
}

describe('EventCalendar', () => {
  beforeEach(() => {
    vi.clearAllMocks();
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

  it('toggling the "show absences" checkbox calls app.toggleCalAbsences', () => {
    const app = makeApp();
    render(<EventCalendar />);
    const checkbox = screen.getByRole('checkbox');
    fireEvent.click(checkbox);
    expect(app.toggleCalAbsences).toHaveBeenCalledTimes(1);
  });
});
