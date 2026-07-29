import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, act, fireEvent } from '@testing-library/react';
import { EventCard } from './cards';
import { NewsCard } from './cards';
import { setLocale } from '@/i18n';

const { mockOpenEventDetail, mockSetMyStatus, mockCan } = vi.hoisted(() => ({
  mockOpenEventDetail: vi.fn(),
  mockSetMyStatus: vi.fn(),
  mockCan: vi.fn().mockReturnValue(true),
}));

vi.mock('@/context/AppContext', () => ({
  useApp: vi.fn().mockReturnValue({
    state: { primaryColor: '#4285F4' },
  }),
  useAppActions: vi.fn().mockReturnValue({
    openEventDetail: mockOpenEventDetail,
    setMyStatus: mockSetMyStatus,
    can: mockCan,
  }),
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

function makeNews(overrides: Record<string, unknown> = {}) {
  return {
    id: 'n1',
    title: 'Neuigkeit',
    body: 'Wichtige Information',
    authorName: 'Coach',
    authorPhoto: null,
    authorColor: '#4285F4',
    createdAt: '2026-01-15T10:00:00Z',
    pinned: false,
    ...overrides,
  } as never;
}

describe('EventCard', () => {
  beforeEach(() => {
    mockOpenEventDetail.mockClear();
    mockSetMyStatus.mockClear();
    mockCan.mockClear();
    mockCan.mockReturnValue(true);
  });

  it('renders event title', () => {
    render(<EventCard e={makeEvent()} />);
    expect(screen.getByText('Jahresabschluss')).toBeTruthy();
  });

  it('sets attendance from an inline RSVP icon on an upcoming event', () => {
    render(<EventCard e={makeEvent({ myStatus: 'maybe' })} />);
    fireEvent.click(screen.getByRole('button', { name: 'Zusagen' }));
    expect(mockSetMyStatus).toHaveBeenCalledWith('ev1', 'yes', undefined);
  });

  it('does not show inline RSVP controls on a past event', () => {
    render(<EventCard e={makeEvent({ date: '2000-01-01' })} />);
    expect(screen.queryByRole('button', { name: 'Zusagen' })).toBeNull();
  });

  it('does not show inline RSVP controls once the cancellation lead time has passed', () => {
    // Event itself is still upcoming (default date 2099-06-15), but an
    // enormous lead time pushes the effective cutoff (start minus
    // cancelLeadMinutes) well into the past.
    mockCan.mockReturnValueOnce(false);
    render(<EventCard e={makeEvent({ cancelLeadMinutes: 100_000_000, myStatus: 'maybe' })} />);
    expect(screen.queryByRole('button', { name: 'Zusagen' })).toBeNull();
  });

  it('still shows inline RSVP controls past the cutoff for a caller holding events:write', () => {
    mockCan.mockReturnValueOnce(true);
    render(<EventCard e={makeEvent({ cancelLeadMinutes: 100_000_000, myStatus: 'maybe' })} />);
    expect(screen.getByRole('button', { name: 'Zusagen' })).toBeTruthy();
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

  // Regression: EventCard is memo()-wrapped on the `e` prop alone, but its
  // translated labels come from t()/getIntlLocale(), which read module-level
  // i18n state rather than a prop. Without subscribing to locale changes, an
  // already-mounted card kept showing the old language until `e` happened to
  // change for an unrelated reason.
  it('re-renders with the new language when the locale changes, without a prop change', async () => {
    render(<EventCard e={makeEvent({ status: 'cancelled' })} />);
    expect(screen.getByText('Abgesagt')).toBeTruthy();

    await act(async () => {
      setLocale('en');
    });

    expect(screen.queryByText('Abgesagt')).toBeNull();
    expect(screen.getByText('Cancelled')).toBeTruthy();

    await act(async () => {
      setLocale('de');
    });
  });
});

describe('NewsCard', () => {
  it('renders news title and body', () => {
    render(<NewsCard n={makeNews()} primaryColor="#4285F4" />);
    expect(screen.getByText('Neuigkeit')).toBeTruthy();
    expect(screen.getByText('Wichtige Information')).toBeTruthy();
  });

  it('renders author name', () => {
    render(<NewsCard n={makeNews()} primaryColor="#4285F4" />);
    expect(screen.getByText('Coach')).toBeTruthy();
  });

  it('renders pin icon when pinned', () => {
    render(<NewsCard n={makeNews({ pinned: true })} primaryColor="#4285F4" />);
    expect(screen.getByText('push_pin')).toBeTruthy();
  });

  it('does not render pin icon when not pinned', () => {
    render(<NewsCard n={makeNews()} primaryColor="#4285F4" />);
    expect(screen.queryByText('push_pin')).toBeNull();
  });
});
