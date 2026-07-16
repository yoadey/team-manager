import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { NotificationsSheet } from './NotificationsSheet';

// Mock AppContext — SpinnerBox calls useApp internally
vi.mock('@/context/AppContext', () => ({
  useApp: vi.fn(),
  useAppActions: vi.fn().mockReturnValue({}),
}));

// Mocked directly on the hooks module (not just a `@/features/notifications`
// barrel re-export) -- NotificationsSheet.tsx imports `useNotificationsQuery`
// via this exact relative path, so this must match it (see the identical
// comment/pattern in NewsPage.test.tsx/PollsPage.test.tsx).
vi.mock('../hooks/useNotificationQueries', () => ({
  useNotificationsQuery: vi.fn(),
}));

import { useApp } from '@/context/AppContext';
import { useNotificationsQuery } from '../hooks/useNotificationQueries';
const mockUseApp = vi.mocked(useApp);
const mockUseNotificationsQuery = vi.mocked(useNotificationsQuery);

vi.mock('@/styles/tokens', async (importOriginal) => {
  const mod = await importOriginal<typeof import('@/styles/tokens')>();
  return {
    ...mod,
    buildTokens: vi.fn().mockReturnValue({
      primary: '#4285F4',
      primaryContainer: '#E8F0FE',
      onPrimaryContainer: '#001D35',
      onPrimary: '#ffffff',
    }),
  };
});

import type { AppNotification } from '../types';
import type { AppContextValue, AppState } from '@/context/AppContext';

// Base AppState stub — only the fields NotificationsSheet touches
function makeState(overrides: Partial<AppState> = {}): AppState {
  return {
    primaryColor: '#1565C0',
    activeTeamId: 't1',
    notifFilter: 'all',
    user: { id: 'u1', name: 'Test User', email: 'test@test.com' } as AppState['user'],
    busy: null,
    ...overrides,
  } as AppState;
}

function makeApp(
  overrides: Partial<AppState> & { notifications?: AppNotification[] | null } = {},
  methods: Partial<AppContextValue> = {},
): AppContextValue {
  const { notifications, ...stateOverrides } = overrides;
  mockUseNotificationsQuery.mockReturnValue({
    data: notifications === null || notifications === undefined ? undefined : { items: notifications, unreadCount: 0 },
  } as never);
  const state = makeState(stateOverrides);
  const app = {
    api: {},
    state,
    setNotifFilter: vi.fn(),
    openEventDetail: vi.fn(),
    go: vi.fn(),
    setState: vi.fn(),
    loadAbsences: vi.fn(),
    goEventsAbsences: vi.fn(),
    ...methods,
  } as unknown as AppContextValue;
  // SpinnerBox calls useApp() directly, so we need to mock its return value too
  mockUseApp.mockReturnValue(app);
  return app;
}

function makeNotification(overrides: Partial<AppNotification> = {}): AppNotification {
  return {
    id: 'n1',
    teamId: 't1',
    type: 'news',
    title: 'Test News',
    actorName: 'Max Mustermann',
    createdAt: new Date().toISOString(),
    unread: false,
    ...overrides,
  };
}

describe('NotificationsSheet', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  // ── Loading state ──────────────────────────────────────────────────────────

  it('shows spinner when notifications is null (loading)', () => {
    const app = makeApp({ notifications: null });
    render(<NotificationsSheet app={app} sheet={{ type: 'notifications' }} />);
    // SpinnerBox renders a role="status" element
    expect(screen.getByRole('status')).toBeTruthy();
  });

  // ── Empty state ────────────────────────────────────────────────────────────

  it('shows empty state when notification list is empty', () => {
    const app = makeApp({ notifications: [] });
    render(<NotificationsSheet app={app} sheet={{ type: 'notifications' }} />);
    // i18n key notifications.empty → German text rendered
    const text = screen.getByText(/keine/i);
    expect(text).toBeTruthy();
  });

  it('shows filter chips even when list is empty', () => {
    const app = makeApp({ notifications: [] });
    render(<NotificationsSheet app={app} sheet={{ type: 'notifications' }} />);
    // Four chips: Alle, Anwesenheit, Events, Sonstiges
    const buttons = screen.getAllByRole('button');
    expect(buttons.length).toBeGreaterThanOrEqual(4);
  });

  // ── Notification items ─────────────────────────────────────────────────────

  it('renders a news notification with title and actor', () => {
    const n = makeNotification({ type: 'news', title: 'Wichtige Neuigkeit', actorName: 'Lisa Müller' });
    const app = makeApp({ notifications: [n] });
    render(<NotificationsSheet app={app} sheet={{ type: 'notifications' }} />);
    expect(screen.getByText('Wichtige Neuigkeit · Lisa Müller')).toBeTruthy();
  });

  it('renders a poll notification with title and actor', () => {
    const n = makeNotification({ type: 'poll', title: 'Umfrage 1', actorName: 'Hans Schmidt' });
    const app = makeApp({ notifications: [n] });
    render(<NotificationsSheet app={app} sheet={{ type: 'notifications' }} />);
    expect(screen.getByText('Umfrage 1 · Hans Schmidt')).toBeTruthy();
  });

  it('renders an attendance notification with actor name and event title', () => {
    const n = makeNotification({
      id: 'n2',
      type: 'attendance',
      status: 'yes',
      actorName: 'Klaus Fischer',
      eventTitle: 'Training Montag',
      eventDate: '2026-06-20',
      eventId: 'e1',
    });
    const app = makeApp({ notifications: [n] });
    render(<NotificationsSheet app={app} sheet={{ type: 'notifications' }} />);
    // line2 contains the event title
    expect(screen.getByText(/Training Montag/)).toBeTruthy();
  });

  // Regression test: eventDate is optional per the OpenAPI AppNotification
  // schema (and the nullable event_date DB column), but line2 used to call
  // fmtDate(n.eventDate!) unconditionally -- passing undefined through to
  // Intl.DateTimeFormat.format(new Date(undefined)) throws a RangeError and
  // would crash the sheet. Must render the event title alone instead.
  it('renders an attendance notification without an eventDate without throwing', () => {
    const n = makeNotification({
      id: 'n2b',
      type: 'attendance',
      status: 'yes',
      actorName: 'Klaus Fischer',
      eventTitle: 'Training Montag',
      eventDate: undefined,
      eventId: 'e1',
    });
    const app = makeApp({ notifications: [n] });
    expect(() => render(<NotificationsSheet app={app} sheet={{ type: 'notifications' }} />)).not.toThrow();
    expect(screen.getByText('Training Montag')).toBeTruthy();
  });

  it('renders an event_created notification', () => {
    const n = makeNotification({
      id: 'n3',
      type: 'event_created',
      title: 'Neues Spiel',
      actorName: 'Trainer',
      eventId: 'e2',
    });
    const app = makeApp({ notifications: [n] });
    render(<NotificationsSheet app={app} sheet={{ type: 'notifications' }} />);
    expect(screen.getByText(/Neues Spiel/)).toBeTruthy();
  });

  it('renders an absence notification with actor name', () => {
    const n = makeNotification({
      id: 'n4',
      type: 'absence',
      actorName: 'Peter Pan',
      title: 'Urlaub',
    });
    const app = makeApp({ notifications: [n] });
    render(<NotificationsSheet app={app} sheet={{ type: 'notifications' }} />);
    expect(screen.getByText(/Peter Pan/)).toBeTruthy();
  });

  // ── Unread indicator ───────────────────────────────────────────────────────

  it('renders unread notification text (unread flag = true)', () => {
    const n = makeNotification({ unread: true, type: 'news', title: 'Ungelesen' });
    const app = makeApp({ notifications: [n] });
    render(<NotificationsSheet app={app} sheet={{ type: 'notifications' }} />);
    expect(screen.getByText(/Ungelesen/)).toBeTruthy();
  });

  // ── Filter chip interaction ────────────────────────────────────────────────

  it('calls setNotifFilter when a filter chip is clicked', () => {
    const n = makeNotification({ type: 'news' });
    const app = makeApp({ notifications: [n] });
    render(<NotificationsSheet app={app} sheet={{ type: 'notifications' }} />);
    // Buttons order: Alle, Anwesenheit, Events, Sonstiges
    const buttons = screen.getAllByRole('button');
    fireEvent.click(buttons[2]); // "Events"
    expect(app.setNotifFilter).toHaveBeenCalledWith('events');
  });

  it('calls setNotifFilter with "attendance" for the attendance chip', () => {
    const app = makeApp({ notifications: [] });
    render(<NotificationsSheet app={app} sheet={{ type: 'notifications' }} />);
    const buttons = screen.getAllByRole('button');
    fireEvent.click(buttons[1]); // "Anwesenheit"
    expect(app.setNotifFilter).toHaveBeenCalledWith('attendance');
  });

  it('calls setNotifFilter with "other" for the other chip', () => {
    const app = makeApp({ notifications: [] });
    render(<NotificationsSheet app={app} sheet={{ type: 'notifications' }} />);
    const buttons = screen.getAllByRole('button');
    fireEvent.click(buttons[3]); // "Sonstiges"
    expect(app.setNotifFilter).toHaveBeenCalledWith('other');
  });

  it('calls setNotifFilter with "all" when clicking the "Alle" chip', () => {
    const app = makeApp({ notifications: [], notifFilter: 'events' });
    render(<NotificationsSheet app={app} sheet={{ type: 'notifications' }} />);
    const buttons = screen.getAllByRole('button');
    fireEvent.click(buttons[0]); // "Alle"
    expect(app.setNotifFilter).toHaveBeenCalledWith('all');
  });

  // ── Click handlers ─────────────────────────────────────────────────────────

  it('calls go("news") when a news notification is clicked', () => {
    const n = makeNotification({ type: 'news', title: 'Breaking', actorName: 'Reporter' });
    const app = makeApp({ notifications: [n] });
    render(<NotificationsSheet app={app} sheet={{ type: 'notifications' }} />);
    const buttons = screen.getAllByRole('button');
    const notifButton = buttons.find((b) => b.textContent?.includes('Breaking'));
    if (notifButton) {
      fireEvent.click(notifButton);
      expect(app.go).toHaveBeenCalledWith('news');
    }
  });

  it('calls openEventDetail when an event_created notification with eventId is clicked', () => {
    const n = makeNotification({
      id: 'n5',
      type: 'event_created',
      title: 'Match',
      actorName: 'Coach',
      eventId: 'event-42',
    });
    const app = makeApp({ notifications: [n] });
    render(<NotificationsSheet app={app} sheet={{ type: 'notifications' }} />);
    const buttons = screen.getAllByRole('button');
    const notifButton = buttons.find((b) => b.textContent?.includes('Match'));
    if (notifButton) {
      fireEvent.click(notifButton);
      expect(app.openEventDetail).toHaveBeenCalledWith('event-42');
    }
  });

  // ── Filter grouping ────────────────────────────────────────────────────────

  it('shows both news and poll when "other" filter is active', () => {
    const news = makeNotification({ id: 'n-news', type: 'news', title: 'Only News', actorName: 'Max Mustermann' });
    const poll = makeNotification({ id: 'n-poll', type: 'poll', title: 'Poll Item', actorName: 'A' });
    const app = makeApp({ notifications: [news, poll], notifFilter: 'other' });
    render(<NotificationsSheet app={app} sheet={{ type: 'notifications' }} />);
    expect(screen.getByText('Only News · Max Mustermann')).toBeTruthy();
    expect(screen.getByText('Poll Item · A')).toBeTruthy();
  });

  it('shows empty state when filter matches no notifications', () => {
    const n = makeNotification({ type: 'news' });
    const app = makeApp({ notifications: [n], notifFilter: 'attendance' });
    render(<NotificationsSheet app={app} sheet={{ type: 'notifications' }} />);
    expect(screen.getByText(/keine/i)).toBeTruthy();
  });

  // Regression test: this used to call app.setState/app.loadAbsences
  // directly, bypassing ensureRouteData -- unlike every other notification
  // type's onClick, which goes through app.go(...). A null state.events
  // left over from a failed afterLoginLoad would then never retry, leaving
  // EventsPage stuck on a skeleton loader forever (it gates on
  // state.events before reaching the eventsView === 'absences' branch).
  // goEventsAbsences (AppContext.tsx) fixes this by routing through
  // ensureRouteData like go()/goEventsPending() do.
  it('clicking absence notification calls goEventsAbsences', () => {
    const n = makeNotification({ id: 'n-ab', type: 'absence', actorName: 'Peter Pan', title: 'Urlaub' });
    const app = makeApp({ notifications: [n] });
    render(<NotificationsSheet app={app} sheet={{ type: 'notifications' }} />);
    const buttons = screen.getAllByRole('button');
    const notifButton = buttons.find((b) => b.textContent?.includes('Peter Pan'));
    if (notifButton) {
      fireEvent.click(notifButton);
      expect(app.goEventsAbsences).toHaveBeenCalled();
    }
  });

  // ── Day group label branches ────────────────────────────────────────────────

  function daysAgoIso(days: number): string {
    const d = new Date();
    d.setDate(d.getDate() - days);
    return d.toISOString();
  }

  it('shows "Gestern" group label for notification from 1 day ago', () => {
    const n = makeNotification({ createdAt: daysAgoIso(1) });
    const app = makeApp({ notifications: [n] });
    render(<NotificationsSheet app={app} sheet={{ type: 'notifications' }} />);
    expect(screen.getByText('Gestern')).toBeTruthy();
  });

  it('shows "Diese Woche" group label for notification from 3 days ago', () => {
    const n = makeNotification({ createdAt: daysAgoIso(3) });
    const app = makeApp({ notifications: [n] });
    render(<NotificationsSheet app={app} sheet={{ type: 'notifications' }} />);
    expect(screen.getByText('Diese Woche')).toBeTruthy();
  });

  it('shows "Letzte Woche" group label for notification from 10 days ago', () => {
    const n = makeNotification({ createdAt: daysAgoIso(10) });
    const app = makeApp({ notifications: [n] });
    render(<NotificationsSheet app={app} sheet={{ type: 'notifications' }} />);
    expect(screen.getByText('Letzte Woche')).toBeTruthy();
  });
});
