import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { PollsPage } from './PollsPage';

vi.mock('@/context/AppContext', () => ({
  useApp: vi.fn(),
  useAppActions: vi.fn().mockReturnValue({}),
}));

// Mocked directly on the hooks module (not just a `@/features/polls` barrel
// re-export) -- PollsPage.tsx imports `usePollsQuery` via a relative import,
// so this must match that module path (see the identical comment/pattern in
// EventsPage.test.tsx/MembersPage.test.tsx).
vi.mock('./hooks/usePollQueries', () => ({
  usePollsQuery: vi.fn(),
}));

import { useApp } from '@/context/AppContext';
import { usePollsQuery } from './hooks/usePollQueries';
const mockUseApp = useApp as ReturnType<typeof vi.fn>;
const mockUsePollsQuery = usePollsQuery as ReturnType<typeof vi.fn>;

function makeApp(overrides: Record<string, unknown> = {}) {
  const { polls, ...stateOverrides } = overrides;
  mockUsePollsQuery.mockReturnValue({ data: 'polls' in overrides ? polls : [] });
  return {
    api: {},
    state: {
      primaryColor: '#4285F4',
      activeTeamId: 't1',
      user: { id: 'u1' },
      ...stateOverrides,
    },
    can: vi.fn().mockReturnValue(false),
    openPollForm: vi.fn(),
    removePoll: vi.fn(),
    togglePollOption: vi.fn(),
  };
}

function makeOption(text: string, pct = 50) {
  return {
    id: text,
    text,
    count: 3,
    pct,
    voters: [],
  };
}

function makePoll(overrides: Record<string, unknown> = {}) {
  return {
    id: 'poll1',
    question: 'Welcher Termin passt?',
    options: [makeOption('Montag'), makeOption('Dienstag', 30)],
    multiple: false,
    anonymous: false,
    createdAt: '2026-01-15T10:00:00Z',
    totalVotes: 5,
    myVote: null,
    ...overrides,
  };
}

describe('PollsPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('shows spinner when polls is null', () => {
    mockUseApp.mockReturnValue(makeApp({ polls: null }));
    render(<PollsPage />);
    expect(screen.getByRole('status')).toBeTruthy();
  });

  it('shows empty state when no polls', () => {
    mockUseApp.mockReturnValue(makeApp({ polls: [] }));
    render(<PollsPage />);
    expect(screen.getByText('Noch keine Umfragen')).toBeTruthy();
  });

  it('renders polls when they exist', () => {
    mockUseApp.mockReturnValue(makeApp({ polls: [makePoll()] }));
    render(<PollsPage />);
    expect(screen.getByText('Welcher Termin passt?')).toBeTruthy();
    expect(screen.getByText('Montag')).toBeTruthy();
    expect(screen.getByText('Dienstag')).toBeTruthy();
  });

  it('renders poll with user vote', () => {
    mockUseApp.mockReturnValue(makeApp({ polls: [makePoll({ myVote: ['Montag'] })] }));
    render(<PollsPage />);
    expect(screen.getByText('Welcher Termin passt?')).toBeTruthy();
  });

  it('renders multiple choice poll', () => {
    mockUseApp.mockReturnValue(makeApp({ polls: [makePoll({ multiple: true })] }));
    render(<PollsPage />);
    expect(screen.getByText('Welcher Termin passt?')).toBeTruthy();
  });

  it('shows delete button for admin users', () => {
    const app = makeApp({ polls: [makePoll()] });
    app.can = vi.fn().mockReturnValue(true);
    mockUseApp.mockReturnValue(app);
    render(<PollsPage />);
    expect(screen.getByText('Welcher Termin passt?')).toBeTruthy();
  });

  it('calls togglePollOption when clicking an option', async () => {
    const poll = makePoll();
    const app = makeApp({ polls: [poll] });
    mockUseApp.mockReturnValue(app);
    render(<PollsPage />);
    await userEvent.click(screen.getByText('Montag').closest('button')!);
    expect(app.togglePollOption).toHaveBeenCalledWith(poll, 'Montag');
  });

  it('calls removePoll when clicking delete (admin)', async () => {
    const poll = makePoll();
    const app = makeApp({ polls: [poll] });
    app.can = vi.fn().mockReturnValue(true);
    mockUseApp.mockReturnValue(app);
    render(<PollsPage />);
    await userEvent.click(screen.getByLabelText('Umfrage löschen'));
    expect(app.removePoll).toHaveBeenCalledWith('poll1');
  });

  it('renders anonymous poll with Anonym chip', () => {
    mockUseApp.mockReturnValue(makeApp({ polls: [makePoll({ anonymous: true })] }));
    render(<PollsPage />);
    expect(screen.getByText('Anonym')).toBeTruthy();
  });
});
