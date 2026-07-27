import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { PollVotersSheet } from './PollVotersSheet';
import type { Poll } from '../types';

vi.mock('@/context/AppContext', () => ({ useApp: vi.fn() }));

const app = { state: { primaryColor: '#4285F4' } } as never;

function voter(userId: string, name: string) {
  return { userId, membershipId: 'm_' + userId, name, color: '#888', photo: null };
}

function makePoll(over: Partial<Poll> = {}): Poll {
  return {
    id: 'p1',
    question: 'Best day?',
    multiple: true,
    anonymous: false,
    createdAt: '2026-01-01T00:00:00Z',
    totalVotes: 2,
    myVote: null,
    options: [
      { id: 'o1', text: 'Monday', count: 2, pct: 100, voters: [voter('u1', 'Alice'), voter('u2', 'Bob')] },
      { id: 'o2', text: 'Tuesday', count: 1, pct: 50, voters: [voter('u1', 'Alice')] },
    ],
    ...over,
  };
}

function sheet(poll: Poll | null) {
  return { type: 'pollVoters', poll } as never;
}

describe('PollVotersSheet', () => {
  beforeEach(() => vi.clearAllMocks());

  it('lists voters per option in the by-option view', () => {
    render(<PollVotersSheet app={app} sheet={sheet(makePoll())} />);
    expect(screen.getByText('Monday')).toBeTruthy();
    // Alice voted for both options -> appears twice in the by-option lists.
    expect(screen.getAllByText('Alice').length).toBe(2);
    expect(screen.getAllByText('Bob').length).toBe(1);
  });

  it('marks each user’s selected options in the matrix view', () => {
    render(<PollVotersSheet app={app} sheet={sheet(makePoll())} />);
    fireEvent.click(screen.getByRole('button', { name: /Matrix/i }));
    // One row per distinct user (Alice, Bob) -> each name appears once.
    expect(screen.getAllByText('Alice').length).toBe(1);
    expect(screen.getAllByText('Bob').length).toBe(1);
    // Legend maps numbered columns back to option text.
    expect(screen.getByText('Tuesday')).toBeTruthy();
  });

  it('reveals no identities for an anonymous poll', () => {
    render(<PollVotersSheet app={app} sheet={sheet(makePoll({ anonymous: true }))} />);
    expect(screen.queryByText('Alice')).toBeNull();
    expect(screen.queryByText('Bob')).toBeNull();
  });

  it('shows an empty state when there are no votes', () => {
    const empty = makePoll({
      totalVotes: 0,
      options: [
        { id: 'o1', text: 'Monday', count: 0, pct: 0, voters: [] },
        { id: 'o2', text: 'Tuesday', count: 0, pct: 0, voters: [] },
      ],
    });
    render(<PollVotersSheet app={app} sheet={sheet(empty)} />);
    expect(screen.queryByText('Alice')).toBeNull();
  });
});
