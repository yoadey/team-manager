import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { EventCard } from './cards';
import { NewsCard } from './cards';

const { mockOpenEventDetail } = vi.hoisted(() => ({ mockOpenEventDetail: vi.fn() }));

vi.mock('@/context/AppContext', () => ({
  useApp: vi.fn().mockReturnValue({
    state: { primaryColor: '#4285F4' },
  }),
  useAppActions: vi.fn().mockReturnValue({ openEventDetail: mockOpenEventDetail }),
}));

function makeEvent(overrides: Record<string, unknown> = {}) {
  return {
    id: 'ev1',
    title: 'Jahresabschluss',
    date: '2099-06-15',
    type: 'training',
    status: 'active',
    myStatus: 'yes',
    summary: { yes: 8, no: 2, maybe: 0 },
    location: 'Halle',
    note: '',
    startTime: '19:30',
    endTime: '21:00',
    meetTime: '19:15',
    meetTimeMandatory: true,
    responseMode: 'opt_out',
    nominatedRoleIds: [],
    ...overrides,
  } as never;
}

function makeNews() {
  return {
    id: 'n1',
    title: 'Neuigkeit',
    body: 'Wichtige Information',
    authorName: 'Coach',
    authorPhoto: null,
    authorColor: '#4285F4',
    createdAt: '2026-01-15T10:00:00Z',
    pinned: false,
  } as never;
}

describe('EventCard', () => {
  beforeEach(() => {
    mockOpenEventDetail.mockClear();
  });

  it('renders event title', () => {
    render(<EventCard e={makeEvent()} />);
    expect(screen.getByText('Jahresabschluss')).toBeTruthy();
  });

  it('renders attendance counts', () => {
    render(<EventCard e={makeEvent()} />);
    expect(screen.getByText(/8✓/)).toBeTruthy();
    expect(screen.getByText(/2✕/)).toBeTruthy();
  });

  it('renders maybe count when non-zero', () => {
    render(<EventCard e={makeEvent({ summary: { yes: 5, no: 1, maybe: 3 } })} />);
    expect(screen.getByText(/3\?/)).toBeTruthy();
  });

  it('does not render maybe when zero', () => {
    render(<EventCard e={makeEvent({ summary: { yes: 5, no: 1, maybe: 0 } })} />);
    expect(screen.queryByText(/\?/)).toBeNull();
  });

  it('renders cancelled event without status chip', () => {
    render(<EventCard e={makeEvent({ status: 'cancelled' })} />);
    expect(screen.getByText('Jahresabschluss')).toBeTruthy();
  });
});

describe('NewsCard', () => {
  it('renders news title and body', () => {
    render(<NewsCard n={makeNews()} />);
    expect(screen.getByText('Neuigkeit')).toBeTruthy();
    expect(screen.getByText('Wichtige Information')).toBeTruthy();
  });

  it('renders author name', () => {
    render(<NewsCard n={makeNews()} />);
    expect(screen.getByText('Coach')).toBeTruthy();
  });

  it('renders pin icon when pinned', () => {
    render(<NewsCard n={{ ...makeNews(), pinned: true }} />);
    expect(screen.getByText('push_pin')).toBeTruthy();
  });

  it('does not render pin icon when not pinned', () => {
    render(<NewsCard n={makeNews()} />);
    expect(screen.queryByText('push_pin')).toBeNull();
  });
});
