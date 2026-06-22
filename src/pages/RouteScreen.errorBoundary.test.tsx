import { describe, it, expect, vi } from 'vitest';
import { render, screen, act } from '@testing-library/react';
import { RouteScreen } from './index';

// The per-route ErrorBoundary must catch a crashing page so the app shell and
// navigation survive instead of the whole tree unmounting.

vi.mock('@/context/AppContext', () => ({
  useApp: vi.fn(),
}));

vi.mock('./Home', () => ({
  Home: () => {
    throw new Error('boom: home page crashed');
  },
}));

vi.mock('@/features/events', () => ({ EventsPage: () => <div>Events</div> }));
vi.mock('@/features/members', () => ({ MembersPage: () => <div>Members</div> }));
vi.mock('@/features/finances', () => ({ FinancesPage: () => <div>Finances</div> }));
vi.mock('@/features/news', () => ({ NewsPage: () => <div>News</div> }));
vi.mock('@/features/polls', () => ({ PollsPage: () => <div>Polls</div> }));
vi.mock('@/features/team', () => ({ TeamPage: () => <div>Team</div> }));
vi.mock('./Stats', () => ({ Stats: () => <div>Stats</div> }));
vi.mock('@/components/ui', () => ({ SpinnerBox: () => <div role="status">Loading</div> }));

const captureError = vi.fn();
vi.mock('@/monitoring', () => ({ captureError: (...args: unknown[]) => captureError(...args) }));

import { useApp } from '@/context/AppContext';
const mockUseApp = useApp as ReturnType<typeof vi.fn>;

describe('RouteScreen — per-route error boundary', () => {
  it('renders the contained fallback when the active page throws', async () => {
    // Silence React's expected error logging for the thrown render.
    const spy = vi.spyOn(console, 'error').mockImplementation(() => {});
    mockUseApp.mockReturnValue({ state: { route: 'home' }, can: vi.fn().mockReturnValue(true) });

    await act(async () => {
      render(<RouteScreen />);
    });

    // Default (component-level) fallback shows the retry control, not a blank tree.
    expect(screen.getByText('Neu versuchen')).toBeTruthy();
    expect(captureError).toHaveBeenCalled();
    spy.mockRestore();
  });
});
