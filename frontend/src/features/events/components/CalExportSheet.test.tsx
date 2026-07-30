import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { CalExportSheet } from './CalExportSheet';

vi.mock('@/context/AppContext', () => ({
  useApp: vi.fn(),
  useAppActions: vi.fn().mockReturnValue({}),
}));

vi.mock('../hooks/useEventQueries', () => ({
  useEventsQuery: vi.fn(),
}));

vi.mock('../hooks/useCalExportActions', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../hooks/useCalExportActions')>();
  return { ...actual, useCalendarFeedUrlQuery: vi.fn(), useCalendarFeedSettingsQuery: vi.fn() };
});

import { useApp } from '@/context/AppContext';
import { useEventsQuery } from '../hooks/useEventQueries';
import { useCalendarFeedUrlQuery, useCalendarFeedSettingsQuery } from '../hooks/useCalExportActions';
const mockUseApp = vi.mocked(useApp);
const mockUseEventsQuery = vi.mocked(useEventsQuery);
const mockUseCalendarFeedUrlQuery = vi.mocked(useCalendarFeedUrlQuery);
const mockUseCalendarFeedSettingsQuery = vi.mocked(useCalendarFeedSettingsQuery);

const TEST_FEED_URL = 'https://app.example.com/api/v1/calendar-feed/abc123.ics';
const TEST_FEED_SETTINGS = { types: ['training', 'auftritt', 'event'], includeBirthdays: true };

function makeApp(eventsOverrides: unknown[] = []) {
  mockUseEventsQuery.mockReturnValue({ data: eventsOverrides } as never);
  mockUseCalendarFeedUrlQuery.mockReturnValue({ data: TEST_FEED_URL, isLoading: false, isError: false } as never);
  mockUseCalendarFeedSettingsQuery.mockReturnValue({ data: TEST_FEED_SETTINGS } as never);
  return {
    api: {},
    state: {
      primaryColor: '#4285F4',
      activeTeamId: 'team1',
    },
    activeTeam: vi.fn().mockReturnValue({ id: 'team1', name: 'SG Muster' }),
    downloadIcs: vi.fn(),
    copyCalUrl: vi.fn(),
    regenerateCalUrl: vi.fn(),
    toggleCalFeedType: vi.fn(),
    toggleCalFeedBirthdays: vi.fn(),
  };
}

describe('CalExportSheet', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  const sheet = {} as never;

  it('renders download button', () => {
    mockUseApp.mockReturnValue(makeApp() as never);
    const app = mockUseApp();
    render(<CalExportSheet app={app as never} sheet={sheet} />);
    expect(screen.getByText(/Kalenderdatei/i)).toBeTruthy();
  });

  it('calls downloadIcs on download button click', () => {
    const app = makeApp();
    mockUseApp.mockReturnValue(app as never);
    render(<CalExportSheet app={app as never} sheet={sheet} />);
    fireEvent.click(screen.getByText(/Kalenderdatei/i).closest('button')!);
    expect(app.downloadIcs).toHaveBeenCalled();
  });

  it('shows the fetched calendar feed URL', () => {
    mockUseApp.mockReturnValue(makeApp() as never);
    const app = mockUseApp();
    render(<CalExportSheet app={app as never} sheet={sheet} />);
    expect(screen.getByText(TEST_FEED_URL)).toBeTruthy();
  });

  it('shows a loading placeholder while the URL is being issued', () => {
    mockUseApp.mockReturnValue(makeApp() as never);
    mockUseCalendarFeedUrlQuery.mockReturnValue({ data: undefined, isLoading: true, isError: false } as never);
    const app = mockUseApp();
    render(<CalExportSheet app={app as never} sheet={sheet} />);
    expect(screen.queryByText(TEST_FEED_URL)).toBeNull();
  });

  it('calls copyCalUrl with the fetched URL when the copy button is clicked', () => {
    const app = makeApp();
    mockUseApp.mockReturnValue(app as never);
    render(<CalExportSheet app={app as never} sheet={sheet} />);
    fireEvent.click(screen.getByText(/Kopieren/i));
    expect(app.copyCalUrl).toHaveBeenCalledWith(TEST_FEED_URL);
  });

  it('calls regenerateCalUrl when the renew link is clicked', () => {
    const app = makeApp();
    mockUseApp.mockReturnValue(app as never);
    render(<CalExportSheet app={app as never} sheet={sheet} />);
    fireEvent.click(screen.getByText(/erneuern/i));
    expect(app.regenerateCalUrl).toHaveBeenCalled();
  });

  it('shows "Kopiert" text when sheet.copied is true', () => {
    mockUseApp.mockReturnValue(makeApp() as never);
    const app = mockUseApp();
    render(<CalExportSheet app={app as never} sheet={{ copied: true } as never} />);
    expect(screen.getByText('Kopiert')).toBeTruthy();
  });

  it('shows active event count in hero text', () => {
    const events = [{ status: 'active' }, { status: 'active' }, { status: 'cancelled' }];
    mockUseApp.mockReturnValue(makeApp(events) as never);
    const app = mockUseApp();
    render(<CalExportSheet app={app as never} sheet={sheet} />);
    // Shows count of active events (2)
    expect(screen.getByText(/2 aktiven Termine/i)).toBeTruthy();
  });

  it('renders Google Calendar hint section', () => {
    mockUseApp.mockReturnValue(makeApp() as never);
    const app = mockUseApp();
    render(<CalExportSheet app={app as never} sheet={sheet} />);
    expect(screen.getByText('Google Kalender')).toBeTruthy();
  });

  it('renders Apple hint section', () => {
    mockUseApp.mockReturnValue(makeApp() as never);
    const app = mockUseApp();
    render(<CalExportSheet app={app as never} sheet={sheet} />);
    expect(screen.getByText('Apple / iOS')).toBeTruthy();
  });

  it('renders a checked toggle for each selected content type and birthdays', () => {
    mockUseApp.mockReturnValue(makeApp() as never);
    const app = mockUseApp();
    render(<CalExportSheet app={app as never} sheet={sheet} />);
    expect(screen.getByText('Training')).toBeTruthy();
    expect(screen.getByText('Wettkampf / Auftritt')).toBeTruthy();
    expect(screen.getByText('Team-Event')).toBeTruthy();
    expect(screen.getByText('Geburtstage')).toBeTruthy();
  });

  it('calls toggleCalFeedType with the clicked type and current settings', () => {
    const app = makeApp();
    mockUseApp.mockReturnValue(app as never);
    render(<CalExportSheet app={app as never} sheet={sheet} />);
    fireEvent.click(screen.getByText('Training'));
    expect(app.toggleCalFeedType).toHaveBeenCalledWith('training', TEST_FEED_SETTINGS);
  });

  it('calls toggleCalFeedBirthdays with the current settings', () => {
    const app = makeApp();
    mockUseApp.mockReturnValue(app as never);
    render(<CalExportSheet app={app as never} sheet={sheet} />);
    fireEvent.click(screen.getByText('Geburtstage'));
    expect(app.toggleCalFeedBirthdays).toHaveBeenCalledWith(TEST_FEED_SETTINGS);
  });

  it('does not render the content-selection card while settings are still loading', () => {
    mockUseApp.mockReturnValue(makeApp() as never);
    mockUseCalendarFeedSettingsQuery.mockReturnValue({ data: undefined } as never);
    const app = mockUseApp();
    render(<CalExportSheet app={app as never} sheet={sheet} />);
    expect(screen.queryByText('Geburtstage')).toBeNull();
  });
});
