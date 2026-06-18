import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Home } from './Home';

const mockGo = vi.fn();
const mockGoEventsPending = vi.fn();
const mockOpenEventForm = vi.fn();

function makeApp(overrides: Record<string, unknown> = {}) {
  return {
    state: {
      phase: 'app',
      primaryColor: '#4285F4',
      events: [],
      news: [],
      user: { id: 'u1', name: 'Max Mustermann', avatarColor: '#000', photo: null },
      ...overrides,
    },
    activeTeam: () => ({
      id: 'team1',
      name: 'FC Test',
      photo: null,
      memberCount: 10,
    }),
    go: mockGo,
    goEventsPending: mockGoEventsPending,
    openEventForm: mockOpenEventForm,
  };
}

vi.mock('@/context/AppContext', () => ({
  useApp: vi.fn(),
  useAppActions: vi.fn().mockReturnValue({ openEventDetail: vi.fn() }),
}));

import { useApp } from '@/context/AppContext';
const mockUseApp = useApp as ReturnType<typeof vi.fn>;

describe('Home', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockUseApp.mockReturnValue(makeApp());
  });

  it('renders team name', () => {
    render(<Home />);
    expect(screen.getByText('FC Test')).toBeTruthy();
  });

  it('shows greeting with user first name', () => {
    render(<Home />);
    expect(screen.getByText(/Hallo Max/)).toBeTruthy();
  });

  it('shows pending response message when there are pending events', () => {
    mockUseApp.mockReturnValue(
      makeApp({
        events: [
          {
            id: 'ev1',
            title: 'Training',
            date: '2099-01-01',
            type: 'training',
            status: 'active',
            myStatus: 'pending',
            summary: { yes: 5, no: 2, maybe: 1 },
            location: null,
            note: null,
            startTime: null,
            endTime: null,
            meetTime: null,
          },
        ],
      }),
    );
    render(<Home />);
    expect(screen.getByText(/brauchen deine Rückmeldung/)).toBeTruthy();
  });

  it('shows "Alles beantwortet" when no pending events', () => {
    render(<Home />);
    expect(screen.getByText(/Alles beantwortet/)).toBeTruthy();
  });

  it('shows empty state when no upcoming events', () => {
    render(<Home />);
    expect(screen.getByText('Keine anstehenden Termine')).toBeTruthy();
  });

  it('shows empty state when no news', () => {
    render(<Home />);
    expect(screen.getByText('Noch keine News')).toBeTruthy();
  });

  it('shows Mitglieder stat with member count', () => {
    render(<Home />);
    expect(screen.getByText('Mitglieder')).toBeTruthy();
    expect(screen.getByText('10')).toBeTruthy();
  });

  it('navigates to events when clicking "Anstehende Termine" stat', async () => {
    render(<Home />);
    const stat = screen.getByText('Anstehende Termine').closest('button')!;
    await userEvent.click(stat);
    expect(mockGo).toHaveBeenCalledWith('events');
  });

  it('navigates to members when clicking Mitglieder stat', async () => {
    render(<Home />);
    const stat = screen.getByText('Mitglieder').closest('button')!;
    await userEvent.click(stat);
    expect(mockGo).toHaveBeenCalledWith('members');
  });

  it('calls goEventsPending when clicking pending stat', async () => {
    render(<Home />);
    const stat = screen.getByText('Offene Rückmeldungen').closest('button')!;
    await userEvent.click(stat);
    expect(mockGoEventsPending).toHaveBeenCalled();
  });

  it('navigates to events when clicking "Alle ansehen" events link', async () => {
    render(<Home />);
    const links = screen.getAllByText('Alle ansehen');
    await userEvent.click(links[0]);
    expect(mockGo).toHaveBeenCalledWith('events');
  });

  it('navigates to news when clicking "Alle ansehen" news link', async () => {
    render(<Home />);
    const links = screen.getAllByText('Alle ansehen');
    await userEvent.click(links[1]);
    expect(mockGo).toHaveBeenCalledWith('news');
  });

  it('renders with team photo as background', () => {
    mockUseApp.mockReturnValue(
      makeApp({
        teams: [],
        activeTeam: () => ({
          id: 'team1',
          name: 'Photo Team',
          photo: 'data:image/png;base64,abc',
          memberCount: 5,
        }),
      }),
    );
    mockUseApp.mockReturnValue({
      ...makeApp(),
      activeTeam: () => ({
        id: 'team1',
        name: 'Photo Team',
        photo: 'data:image/png;base64,abc',
        memberCount: 5,
      }),
    });
    render(<Home />);
    expect(screen.getByText('Photo Team')).toBeTruthy();
  });
});
