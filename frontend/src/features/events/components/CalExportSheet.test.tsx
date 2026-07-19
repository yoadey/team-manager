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

import { useApp } from '@/context/AppContext';
import { useEventsQuery } from '../hooks/useEventQueries';
const mockUseApp = vi.mocked(useApp);
const mockUseEventsQuery = vi.mocked(useEventsQuery);

function makeApp(eventsOverrides: unknown[] = []) {
  mockUseEventsQuery.mockReturnValue({ data: eventsOverrides } as never);
  return {
    api: {},
    state: {
      primaryColor: '#4285F4',
      activeTeamId: 'team1',
    },
    activeTeam: vi.fn().mockReturnValue({ id: 'team1', name: 'SG Muster' }),
    downloadIcs: vi.fn(),
    copyCalUrl: vi.fn(),
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

  it('shows calendar URL with team id', () => {
    mockUseApp.mockReturnValue(makeApp() as never);
    const app = mockUseApp();
    render(<CalExportSheet app={app as never} sheet={sheet} />);
    expect(screen.getByText(/team1\.ics/)).toBeTruthy();
  });

  it('calls copyCalUrl when copy button clicked', () => {
    const app = makeApp();
    mockUseApp.mockReturnValue(app as never);
    render(<CalExportSheet app={app as never} sheet={sheet} />);
    fireEvent.click(screen.getByText(/Kopieren/i));
    expect(app.copyCalUrl).toHaveBeenCalled();
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

  // Regression test: the subscribe-link description used to assert the
  // calendar "bleibt automatisch aktuell" (stays up to date automatically)
  // unconditionally, directly contradicting the prototype note further down
  // the same sheet, which says the subscription link isn't functional yet.
  // A user reading only the description (and the "copied" success toast)
  // had no reason to doubt the link would actually work.
  it('subscribe description does not contradict the prototype note about the link not being active yet', () => {
    mockUseApp.mockReturnValue(makeApp() as never);
    const app = mockUseApp();
    render(<CalExportSheet app={app as never} sheet={sheet} />);
    expect(screen.getByText(/Dieser Link soll/i).textContent).toMatch(/noch nicht aktiv/i);
  });
});
