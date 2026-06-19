import { describe, it, expect, vi } from 'vitest';
import { render, screen, act } from '@testing-library/react';
import { RouteScreen } from './index';

vi.mock('@/context/AppContext', () => ({
  useApp: vi.fn(),
}));

vi.mock('./Home', () => ({
  Home: () => <div data-testid="home">Home</div>,
}));

vi.mock('@/features/events', () => ({
  EventsPage: () => <div data-testid="events">Events</div>,
}));

vi.mock('@/features/members', () => ({
  MembersPage: () => <div data-testid="members">Members</div>,
}));

vi.mock('@/components/ui', () => ({
  SpinnerBox: () => <div role="status">Loading</div>,
}));

vi.mock('./Stats', () => ({
  Stats: () => <div data-testid="stats">Stats</div>,
}));

vi.mock('@/features/finances', () => ({
  FinancesPage: () => <div data-testid="finances">Finances</div>,
}));

vi.mock('@/features/news', () => ({
  NewsPage: () => <div data-testid="news">News</div>,
}));

vi.mock('@/features/polls', () => ({
  PollsPage: () => <div data-testid="polls">Polls</div>,
}));

vi.mock('@/features/team', () => ({
  TeamPage: () => <div data-testid="team">Team</div>,
}));

import { useApp } from '@/context/AppContext';
const mockUseApp = useApp as ReturnType<typeof vi.fn>;

function makeApp(route: string, hasPermission = true) {
  return {
    state: { route },
    can: vi.fn().mockReturnValue(hasPermission),
  };
}

describe('RouteScreen', () => {
  it('renders Home for home route', async () => {
    mockUseApp.mockReturnValue(makeApp('home'));
    await act(async () => {
      render(<RouteScreen />);
    });
    expect(screen.getByTestId('home')).toBeTruthy();
  });

  it('renders EventsPage for events route', async () => {
    mockUseApp.mockReturnValue(makeApp('events'));
    await act(async () => {
      render(<RouteScreen />);
    });
    expect(screen.getByTestId('events')).toBeTruthy();
  });

  it('renders MembersPage for members route', async () => {
    mockUseApp.mockReturnValue(makeApp('members'));
    await act(async () => {
      render(<RouteScreen />);
    });
    expect(screen.getByTestId('members')).toBeTruthy();
  });

  it('renders Home for finances route when no permission', async () => {
    mockUseApp.mockReturnValue(makeApp('finances', false));
    await act(async () => {
      render(<RouteScreen />);
    });
    expect(screen.getByTestId('home')).toBeTruthy();
  });

  it('renders FinancesPage for finances route with permission', async () => {
    mockUseApp.mockReturnValue(makeApp('finances'));
    await act(async () => {
      render(<RouteScreen />);
    });
    expect(screen.getByTestId('finances')).toBeTruthy();
  });

  it('renders Home for unknown route', async () => {
    mockUseApp.mockReturnValue(makeApp('unknown'));
    await act(async () => {
      render(<RouteScreen />);
    });
    expect(screen.getByTestId('home')).toBeTruthy();
  });

  it('renders NewsPage for news route', async () => {
    mockUseApp.mockReturnValue(makeApp('news'));
    await act(async () => {
      render(<RouteScreen />);
    });
    expect(screen.getByTestId('news')).toBeTruthy();
  });

  it('renders PollsPage for polls route', async () => {
    mockUseApp.mockReturnValue(makeApp('polls'));
    await act(async () => {
      render(<RouteScreen />);
    });
    expect(screen.getByTestId('polls')).toBeTruthy();
  });

  it('renders TeamPage for team route', async () => {
    mockUseApp.mockReturnValue(makeApp('team'));
    await act(async () => {
      render(<RouteScreen />);
    });
    expect(screen.getByTestId('team')).toBeTruthy();
  });

  it('renders Stats for stats route', async () => {
    mockUseApp.mockReturnValue(makeApp('stats'));
    await act(async () => {
      render(<RouteScreen />);
    });
    expect(screen.getByTestId('stats')).toBeTruthy();
  });
});
