import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { EventsPage } from './EventsPage';

vi.mock('@/context/AppContext', () => ({
  useApp: vi.fn(),
  useAppActions: vi.fn().mockReturnValue({ openEventDetail: vi.fn() }),
}));

vi.mock('@/features/events', async (importOriginal) => {
  const mod = await importOriginal<typeof import('@/features/events')>();
  return {
    ...mod,
    EventCalendar: () => <div data-testid="calendar">Calendar</div>,
    EventAbsences: () => <div data-testid="absences">Absences</div>,
  };
});

import { useApp } from '@/context/AppContext';
const mockUseApp = useApp as ReturnType<typeof vi.fn>;

function makeApp(overrides: Record<string, unknown> = {}) {
  return {
    state: {
      primaryColor: '#4285F4',
      events: [],
      eventsView: 'list',
      eventScope: 'upcoming',
      eventsOnlyPending: false,
      calShowAbsences: false,
      calMonth: null,
      absences: null,
      user: { id: 'u1' },
      ...overrides,
    },
    can: vi.fn().mockReturnValue(false),
    go: vi.fn(),
    setEventsView: vi.fn(),
    goEventsPending: vi.fn(),
    toggleCalAbsences: vi.fn(),
    openEventForm: vi.fn(),
    openAbsenceForm: vi.fn(),
    openCalExport: vi.fn(),
    setState: vi.fn(),
  };
}

describe('EventsPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders empty state when no upcoming events', () => {
    mockUseApp.mockReturnValue(makeApp());
    render(<EventsPage />);
    expect(screen.getByText('Keine anstehenden Termine')).toBeTruthy();
  });

  it('renders events when they exist', () => {
    mockUseApp.mockReturnValue(
      makeApp({
        events: [
          {
            id: 'ev1',
            title: 'Saisonauftakt',
            date: '2099-06-15',
            type: 'training',
            status: 'active',
            myStatus: 'yes',
            summary: { yes: 8, no: 2, maybe: 0 },
            location: 'Halle',
            note: '',
            startTime: '19:30',
            endTime: '21:00',
            meetTime: null,
          },
        ],
      }),
    );
    render(<EventsPage />);
    expect(screen.getByText('Saisonauftakt')).toBeTruthy();
  });

  it('renders past events empty state when eventScope is past', () => {
    mockUseApp.mockReturnValue(makeApp({ eventScope: 'past', events: [] }));
    render(<EventsPage />);
    expect(screen.getByText('Kein Termin im Archiv')).toBeTruthy();
  });

  it('renders view toggle buttons', () => {
    mockUseApp.mockReturnValue(makeApp());
    render(<EventsPage />);
    expect(screen.getByText('Liste')).toBeTruthy();
    expect(screen.getByText('Kalender')).toBeTruthy();
    expect(screen.getByText('Abwesend')).toBeTruthy();
  });

  it('renders Exportieren button', () => {
    mockUseApp.mockReturnValue(makeApp());
    render(<EventsPage />);
    expect(screen.getByText('Exportieren')).toBeTruthy();
  });

  it('calls setEventsView when clicking view buttons', async () => {
    const app = makeApp();
    mockUseApp.mockReturnValue(app);
    render(<EventsPage />);
    await userEvent.click(screen.getByText('Kalender'));
    expect(app.setEventsView).toHaveBeenCalledWith('calendar');
  });

  it('calls setState when clicking scope buttons', async () => {
    const app = makeApp();
    mockUseApp.mockReturnValue(app);
    render(<EventsPage />);
    await userEvent.click(screen.getByText('Archiv'));
    expect(app.setState).toHaveBeenCalledWith({ eventScope: 'past' });
  });

  it('calls openCalExport when clicking Exportieren', async () => {
    const app = makeApp();
    mockUseApp.mockReturnValue(app);
    render(<EventsPage />);
    await userEvent.click(screen.getByText('Exportieren'));
    expect(app.openCalExport).toHaveBeenCalled();
  });

  it('renders calendar view when eventsView is calendar', () => {
    mockUseApp.mockReturnValue(makeApp({ eventsView: 'calendar' }));
    render(<EventsPage />);
    expect(screen.getByTestId('calendar')).toBeTruthy();
  });

  it('renders absences view when eventsView is absences', () => {
    mockUseApp.mockReturnValue(makeApp({ eventsView: 'absences' }));
    render(<EventsPage />);
    expect(screen.getByTestId('absences')).toBeTruthy();
  });

  it('renders pending filter chip and clears it', async () => {
    const app = makeApp({ eventsOnlyPending: true, eventScope: 'upcoming' });
    mockUseApp.mockReturnValue(app);
    render(<EventsPage />);
    expect(screen.getByText('Nur offene Rückmeldungen')).toBeTruthy();
    await userEvent.click(screen.getByText('Nur offene Rückmeldungen').closest('button')!);
    expect(app.setState).toHaveBeenCalledWith({ eventsOnlyPending: false });
  });
});
