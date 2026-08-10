import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { CalendarSharesSheet, SharedCalendarsSheet } from './CalendarShareSheets';

vi.mock('@/context/AppContext', () => ({
  useApp: vi.fn(),
  useAppActions: vi.fn().mockReturnValue({}),
}));

vi.mock('../hooks/useCalendarShareQueries', () => ({
  useCalendarSharesQuery: vi.fn(),
  useSharedCalendarSourcesQuery: vi.fn(),
  useSharedCalendarEventsQuery: vi.fn(),
}));

import { useApp } from '@/context/AppContext';
import {
  useCalendarSharesQuery,
  useSharedCalendarSourcesQuery,
  useSharedCalendarEventsQuery,
} from '../hooks/useCalendarShareQueries';

const mockUseApp = vi.mocked(useApp);
const mockUseCalendarSharesQuery = vi.mocked(useCalendarSharesQuery);
const mockUseSharedCalendarSourcesQuery = vi.mocked(useSharedCalendarSourcesQuery);
const mockUseSharedCalendarEventsQuery = vi.mocked(useSharedCalendarEventsQuery);

function makeApp(overrides: Record<string, unknown> = {}) {
  const app = {
    api: {},
    state: { primaryColor: '#4285F4' },
    activeTeam: vi.fn().mockReturnValue({ id: 'owner-team-1', name: 'Meine Mannschaft' }),
    can: vi.fn().mockReturnValue(true),
    grantCalendarShare: vi.fn().mockResolvedValue(undefined),
    revokeCalendarShare: vi.fn(),
    ...overrides,
  };
  mockUseApp.mockReturnValue(app as unknown as ReturnType<typeof useApp>);
  return app;
}

const SHEET = {} as never;
const VALID_UUID = '11111111-1111-1111-1111-111111111111';

describe('CalendarSharesSheet', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockUseCalendarSharesQuery.mockReturnValue({ data: [], isLoading: false } as never);
  });

  it('shows the team\'s own id for sharing with another team\'s admin', () => {
    const app = makeApp();
    render(<CalendarSharesSheet app={app as never} sheet={SHEET} />);
    expect(screen.getByText('owner-team-1')).toBeTruthy();
  });

  it('shows an empty state when no shares exist', () => {
    const app = makeApp();
    render(<CalendarSharesSheet app={app as never} sheet={SHEET} />);
    expect(screen.getByText(/Noch keine Freigaben erteilt/i)).toBeTruthy();
  });

  it('renders granted shares with a revoke button when the caller can manage settings', () => {
    mockUseCalendarSharesQuery.mockReturnValue({
      data: [{ viewerTeamId: 'v1', viewerTeamName: 'Nachbarteam', createdAt: '2026-01-15T00:00:00Z' }],
      isLoading: false,
    } as never);
    const app = makeApp();
    render(<CalendarSharesSheet app={app as never} sheet={SHEET} />);
    expect(screen.getByText('Nachbarteam')).toBeTruthy();
    fireEvent.click(screen.getByLabelText(/Freigabe widerrufen/i));
    expect(app.revokeCalendarShare).toHaveBeenCalledWith('v1', 'Nachbarteam');
  });

  it('does not render revoke controls without settings:write', () => {
    mockUseCalendarSharesQuery.mockReturnValue({
      data: [{ viewerTeamId: 'v1', viewerTeamName: 'Nachbarteam', createdAt: '2026-01-15T00:00:00Z' }],
      isLoading: false,
    } as never);
    const app = makeApp({ can: vi.fn().mockReturnValue(false) });
    render(<CalendarSharesSheet app={app as never} sheet={SHEET} />);
    expect(screen.queryByLabelText(/Freigabe widerrufen/i)).toBeNull();
  });

  it('disables the grant button until a validly-shaped team id is entered', () => {
    const app = makeApp();
    render(<CalendarSharesSheet app={app as never} sheet={SHEET} />);
    const submit = screen.getByText(/Zugriff gewähren/i).closest('button')!;
    expect(submit).toBeDisabled();

    fireEvent.change(screen.getByPlaceholderText('00000000-0000-0000-0000-000000000000'), {
      target: { value: 'not-a-uuid' },
    });
    expect(submit).toBeDisabled();

    fireEvent.change(screen.getByPlaceholderText('00000000-0000-0000-0000-000000000000'), {
      target: { value: VALID_UUID },
    });
    expect(submit).not.toBeDisabled();
  });

  it('calls grantCalendarShare with the entered team id on submit', async () => {
    const app = makeApp();
    render(<CalendarSharesSheet app={app as never} sheet={SHEET} />);
    fireEvent.change(screen.getByPlaceholderText('00000000-0000-0000-0000-000000000000'), {
      target: { value: VALID_UUID },
    });
    fireEvent.click(screen.getByText(/Zugriff gewähren/i));
    expect(app.grantCalendarShare).toHaveBeenCalledWith(VALID_UUID);
  });
});

describe('SharedCalendarsSheet', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('shows an empty state when no team has shared its calendar', () => {
    mockUseSharedCalendarSourcesQuery.mockReturnValue({ data: [], isLoading: false } as never);
    const app = makeApp();
    render(<SharedCalendarsSheet app={app as never} sheet={SHEET} />);
    expect(screen.getByText(/Kein Team hat dir bisher seinen Kalender freigegeben/i)).toBeTruthy();
  });

  it('lists sources and expands to show that team\'s redacted events on click', () => {
    mockUseSharedCalendarSourcesQuery.mockReturnValue({
      data: [{ ownerTeamId: 'o1', ownerTeamName: 'Nachbarteam' }],
      isLoading: false,
    } as never);
    mockUseSharedCalendarEventsQuery.mockReturnValue({
      data: [{ id: 'e1', type: 'training', title: 'Training', date: '2026-02-01', startTime: '18:00', endTime: null, location: 'Halle 1' }],
      isLoading: false,
    } as never);
    const app = makeApp();
    render(<SharedCalendarsSheet app={app as never} sheet={SHEET} />);

    expect(screen.getByText('Nachbarteam')).toBeTruthy();
    expect(screen.queryByText('Training')).toBeNull();

    fireEvent.click(screen.getByText('Nachbarteam'));
    expect(screen.getByText('Training')).toBeTruthy();
    expect(screen.getByText('Halle 1')).toBeTruthy();
    // Redacted projection: nothing attendance/comment/note-shaped ever renders.
    expect(screen.queryByText(/Teilnahme|Kommentar|Notiz/i)).toBeNull();
  });

  it('renders a date range for a multi-day shared event', () => {
    mockUseSharedCalendarSourcesQuery.mockReturnValue({
      data: [{ ownerTeamId: 'o1', ownerTeamName: 'Nachbarteam' }],
      isLoading: false,
    } as never);
    mockUseSharedCalendarEventsQuery.mockReturnValue({
      data: [
        {
          id: 'e1',
          type: 'training',
          title: 'Trainingslager',
          date: '2026-06-10',
          multiDayEndDate: '2026-06-12',
          startTime: null,
          endTime: null,
          location: null,
        },
      ],
      isLoading: false,
    } as never);
    const app = makeApp();
    render(<SharedCalendarsSheet app={app as never} sheet={SHEET} />);
    fireEvent.click(screen.getByText('Nachbarteam'));

    expect(screen.getByText(/–/)).toBeTruthy();
  });
});
