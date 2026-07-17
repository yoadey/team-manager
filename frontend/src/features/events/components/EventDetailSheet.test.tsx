import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { EventDetailSheet as RealEventDetailSheet } from './EventDetailSheet';
import type { SheetProps } from '@/sheets/types';

vi.mock('@/context/AppContext', () => ({
  useApp: vi.fn(),
  useAppActions: vi.fn().mockReturnValue({}),
}));

vi.mock('../hooks/useEventQueries', () => ({
  useEventDetailQuery: vi.fn(),
}));

import { useEventDetailQuery } from '../hooks/useEventQueries';
const mockUseEventDetailQuery = vi.mocked(useEventDetailQuery);

/**
 * EventDetailSheet fetches its own data via `useEventDetailQuery` now
 * (mocked above) instead of reading it off `sheet` -- this shadow component
 * keeps every existing test's `sheet={{ event, rows, comments, eventNotFound }}`
 * fixture working unchanged by translating it into the mocked query's return
 * value, then rendering the real component with just a bare `eventId`.
 */
function deriveQueryResult(s: {
  event?: { id: string } | null;
  rows?: unknown[];
  comments?: unknown[];
  eventNotFound?: boolean;
}) {
  if (s.event === undefined) return { data: undefined, isLoading: true, isError: false, error: null };
  if (s.event === null) {
    if (s.eventNotFound)
      return { data: { event: null, rows: [], comments: [] }, isLoading: false, isError: false, error: null };
    return { data: undefined, isLoading: true, isError: false, error: null };
  }
  return {
    data: { event: s.event, rows: s.rows ?? [], comments: s.comments ?? [] },
    isLoading: false,
    isError: false,
    error: null,
  };
}

function EventDetailSheet({ app, sheet }: SheetProps) {
  const s = sheet as unknown as {
    event?: { id: string } | null;
    rows?: unknown[];
    comments?: unknown[];
    eventNotFound?: boolean;
  };
  mockUseEventDetailQuery.mockReturnValue(deriveQueryResult(s) as never);
  return <RealEventDetailSheet app={app} sheet={{ type: 'eventDetail', eventId: s.event?.id ?? 'ev1' }} />;
}

vi.mock('@/styles/tokens', () => ({
  buildTokens: vi.fn().mockReturnValue({
    primary: '#4285F4',
    primaryDark: '#1565C0',
    onPrimary: '#fff',
    primaryContainer: '#D7E3FF',
    onPrimaryContainer: '#001B3E',
    error: '#B00020',
  }),
  typeMeta: vi.fn().mockReturnValue({ icon: 'celebration', label: 'Event', color: '#1565C0', bg: '#E3F2FD' }),
  statusMeta: vi.fn().mockReturnValue({ icon: 'check_circle', label: 'Zusagen', color: '#2E7D32', bg: '#E8F5E9' }),
  initials: vi.fn().mockImplementation((name: string) => name.slice(0, 2).toUpperCase()),
  NEUTRAL: {
    surface: '#FAFAFA',
    card: '#FFFFFF',
    appBg: '#F5F5F5',
    line: '#E0E0E0',
    secondary: '#757575',
    error: '#B00020',
    errorBg: '#FFEBEE',
    primaryText: '#212121',
    on: '#000000',
    faint: '#BDBDBD',
    success: '#2E7D32',
    successBg: '#E8F5E9',
  },
  fmtDateLong: vi.fn().mockReturnValue('1. Juli 2026'),
  fmtDateTime: vi.fn().mockReturnValue('01.07.2026 19:00'),
  hhmm: vi.fn().mockImplementation((v) => v || ''),
}));

vi.mock('@/i18n', () => ({
  t: vi.fn().mockImplementation((key: string) => key),
}));

vi.mock('@/utils/date', () => ({
  todayLocalDate: vi.fn().mockReturnValue('2026-06-20'),
}));

import { useApp } from '@/context/AppContext';
const mockUseApp = vi.mocked(useApp);

const makeEvent = (overrides = {}) => ({
  id: 'ev1',
  title: 'Sommerball',
  date: '2026-07-01',
  type: 'event' as const,
  status: 'active' as const,
  myStatus: 'yes' as const,
  myAuto: false,
  myReason: '',
  recurring: false,
  location: 'Sporthalle',
  note: null,
  result: null,
  startTime: '19:00',
  endTime: '21:00',
  meetTime: null,
  meetTimeMandatory: false,
  responseMode: 'opt_out' as const,
  nominatedRoleIds: [],
  seriesId: null,
  teamId: 'team1',
  summary: { yes: 3, no: 1, maybe: 0, pending: 2, notNominated: 0, nominated: 6, total: 6 },
  ...overrides,
});

function makeApp(overrides: Record<string, unknown> = {}) {
  return {
    state: {
      primaryColor: '#4285F4',
      form: { newEventComment: '' },
      user: { id: 'user1', name: 'Max Mustermann' },
      roles: [],
      busy: null,
      ...overrides,
    },
    can: vi.fn().mockReturnValue(false),
    canSeeComment: vi.fn().mockReturnValue(true),
    setState: vi.fn(),
    toastMsg: vi.fn(),
    logout: vi.fn(),
    setMyStatus: vi.fn(),
    setStatusFor: vi.fn(),
    openEventForm: vi.fn(),
    askEventAction: vi.fn(),
    openComment: vi.fn(),
    toggleNomination: vi.fn(),
    postEventComment: vi.fn().mockResolvedValue(true),
    removeEventComment: vi.fn(),
  };
}

describe('EventDetailSheet', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders SpinnerBox when event is null and still loading', () => {
    const app = makeApp();
    mockUseApp.mockReturnValue(app as never);
    const { container } = render(
      <EventDetailSheet app={app as never} sheet={{ type: 'eventDetail', event: null } as never} />,
    );
    // SpinnerBox renders without the event details
    expect(container.firstChild).toBeTruthy();
    expect(screen.queryByText('events.detailNotFound')).toBeNull();
  });

  // Regression test: EventDetailSheet used to render SpinnerBox forever for a
  // confirmed-missing event (deleted, or inaccessible) -- `sheet.event` is
  // null both while still loading AND once a reload resolves with a
  // confirmed 404, with no way to distinguish the two, so a stale
  // bookmarked/deep-linked URL, or an event deleted in another tab, left the
  // user staring at an infinite spinner with no explanation and no way out
  // except manually closing the sheet.
  it('renders a not-found empty state when the event is confirmed missing', () => {
    const app = makeApp();
    mockUseApp.mockReturnValue(app as never);
    render(
      <EventDetailSheet
        app={app as never}
        sheet={{ type: 'eventDetail', event: null, eventNotFound: true } as never}
      />,
    );
    expect(screen.getByText('events.detailNotFound')).toBeTruthy();
  });

  it('renders event title in the document', () => {
    const app = makeApp();
    mockUseApp.mockReturnValue(app as never);
    const event = makeEvent();
    render(
      <EventDetailSheet app={app as never} sheet={{ type: 'eventDetail', event, rows: [], comments: [] } as never} />,
    );
    // The date line with fmtDateLong is rendered
    expect(screen.getByText('1. Juli 2026')).toBeTruthy();
  });

  it('renders type chip label', () => {
    const app = makeApp();
    mockUseApp.mockReturnValue(app as never);
    const event = makeEvent();
    render(
      <EventDetailSheet app={app as never} sheet={{ type: 'eventDetail', event, rows: [], comments: [] } as never} />,
    );
    expect(screen.getByText('Event')).toBeTruthy();
  });

  it('renders recurring chip when event is recurring', () => {
    const app = makeApp();
    mockUseApp.mockReturnValue(app as never);
    const event = makeEvent({ recurring: true });
    render(
      <EventDetailSheet app={app as never} sheet={{ type: 'eventDetail', event, rows: [], comments: [] } as never} />,
    );
    expect(screen.getByText('events.weekly')).toBeTruthy();
  });

  it('does not render recurring chip when event is not recurring', () => {
    const app = makeApp();
    mockUseApp.mockReturnValue(app as never);
    const event = makeEvent({ recurring: false });
    render(
      <EventDetailSheet app={app as never} sheet={{ type: 'eventDetail', event, rows: [], comments: [] } as never} />,
    );
    expect(screen.queryByText('events.weekly')).toBeNull();
  });

  it('renders cancel banner when event is cancelled', () => {
    const app = makeApp();
    mockUseApp.mockReturnValue(app as never);
    const event = makeEvent({ status: 'cancelled' });
    render(
      <EventDetailSheet app={app as never} sheet={{ type: 'eventDetail', event, rows: [], comments: [] } as never} />,
    );
    expect(screen.getByText('events.cancelledBanner')).toBeTruthy();
  });

  it('does not render cancel banner when event is active', () => {
    const app = makeApp();
    mockUseApp.mockReturnValue(app as never);
    const event = makeEvent({ status: 'active' });
    render(
      <EventDetailSheet app={app as never} sheet={{ type: 'eventDetail', event, rows: [], comments: [] } as never} />,
    );
    expect(screen.queryByText('events.cancelledBanner')).toBeNull();
  });

  it('renders reactivate button in cancel banner when user can edit', () => {
    const app = makeApp();
    (app.can as ReturnType<typeof vi.fn>).mockReturnValue(true);
    mockUseApp.mockReturnValue(app as never);
    const event = makeEvent({ status: 'cancelled' });
    render(
      <EventDetailSheet app={app as never} sheet={{ type: 'eventDetail', event, rows: [], comments: [] } as never} />,
    );
    expect(screen.getByText('events.reactivate')).toBeTruthy();
  });

  it('does not render reactivate button when user cannot edit', () => {
    const app = makeApp();
    (app.can as ReturnType<typeof vi.fn>).mockReturnValue(false);
    mockUseApp.mockReturnValue(app as never);
    const event = makeEvent({ status: 'cancelled' });
    render(
      <EventDetailSheet app={app as never} sheet={{ type: 'eventDetail', event, rows: [], comments: [] } as never} />,
    );
    expect(screen.queryByText('events.reactivate')).toBeNull();
  });

  it('clicking reactivate calls askEventAction', () => {
    const app = makeApp();
    (app.can as ReturnType<typeof vi.fn>).mockReturnValue(true);
    mockUseApp.mockReturnValue(app as never);
    const event = makeEvent({ status: 'cancelled' });
    render(
      <EventDetailSheet app={app as never} sheet={{ type: 'eventDetail', event, rows: [], comments: [] } as never} />,
    );
    fireEvent.click(screen.getByText('events.reactivate'));
    expect(app.askEventAction).toHaveBeenCalledWith('reactivate', event);
  });

  it('renders edit and delete buttons when user can edit', () => {
    const app = makeApp();
    (app.can as ReturnType<typeof vi.fn>).mockReturnValue(true);
    mockUseApp.mockReturnValue(app as never);
    const event = makeEvent();
    render(
      <EventDetailSheet app={app as never} sheet={{ type: 'eventDetail', event, rows: [], comments: [] } as never} />,
    );
    expect(screen.getByText('events.edit')).toBeTruthy();
    expect(screen.getByText('events.delete')).toBeTruthy();
  });

  it('does not render edit buttons when user cannot edit', () => {
    const app = makeApp();
    (app.can as ReturnType<typeof vi.fn>).mockReturnValue(false);
    mockUseApp.mockReturnValue(app as never);
    const event = makeEvent();
    render(
      <EventDetailSheet app={app as never} sheet={{ type: 'eventDetail', event, rows: [], comments: [] } as never} />,
    );
    expect(screen.queryByText('events.edit')).toBeNull();
  });

  it('renders cancel button when event is active and user can edit', () => {
    const app = makeApp();
    (app.can as ReturnType<typeof vi.fn>).mockReturnValue(true);
    mockUseApp.mockReturnValue(app as never);
    const event = makeEvent({ status: 'active' });
    render(
      <EventDetailSheet app={app as never} sheet={{ type: 'eventDetail', event, rows: [], comments: [] } as never} />,
    );
    expect(screen.getByText('events.cancel')).toBeTruthy();
  });

  it('does not render cancel button when event is already cancelled', () => {
    const app = makeApp();
    (app.can as ReturnType<typeof vi.fn>).mockReturnValue(true);
    mockUseApp.mockReturnValue(app as never);
    const event = makeEvent({ status: 'cancelled' });
    render(
      <EventDetailSheet app={app as never} sheet={{ type: 'eventDetail', event, rows: [], comments: [] } as never} />,
    );
    expect(screen.queryByText('events.cancel')).toBeNull();
  });

  it('clicking edit calls openEventForm', () => {
    const app = makeApp();
    (app.can as ReturnType<typeof vi.fn>).mockReturnValue(true);
    mockUseApp.mockReturnValue(app as never);
    const event = makeEvent();
    render(
      <EventDetailSheet app={app as never} sheet={{ type: 'eventDetail', event, rows: [], comments: [] } as never} />,
    );
    fireEvent.click(screen.getByText('events.edit'));
    expect(app.openEventForm).toHaveBeenCalledWith(event);
  });

  it('clicking delete calls askEventAction with delete', () => {
    const app = makeApp();
    (app.can as ReturnType<typeof vi.fn>).mockReturnValue(true);
    mockUseApp.mockReturnValue(app as never);
    const event = makeEvent();
    render(
      <EventDetailSheet app={app as never} sheet={{ type: 'eventDetail', event, rows: [], comments: [] } as never} />,
    );
    // Find delete button via text
    fireEvent.click(screen.getByText('events.delete').closest('button')!);
    expect(app.askEventAction).toHaveBeenCalledWith('delete', event);
  });

  it('renders participants section title', () => {
    const app = makeApp();
    mockUseApp.mockReturnValue(app as never);
    const event = makeEvent();
    render(
      <EventDetailSheet app={app as never} sheet={{ type: 'eventDetail', event, rows: [], comments: [] } as never} />,
    );
    expect(screen.getByText('events.participants')).toBeTruthy();
  });

  it('renders comments section title', () => {
    const app = makeApp();
    mockUseApp.mockReturnValue(app as never);
    const event = makeEvent();
    render(
      <EventDetailSheet app={app as never} sheet={{ type: 'eventDetail', event, rows: [], comments: [] } as never} />,
    );
    expect(screen.getByText('events.comments')).toBeTruthy();
  });

  it('renders no-comments placeholder when comments is empty', () => {
    const app = makeApp();
    mockUseApp.mockReturnValue(app as never);
    const event = makeEvent();
    render(
      <EventDetailSheet app={app as never} sheet={{ type: 'eventDetail', event, rows: [], comments: [] } as never} />,
    );
    expect(screen.getByText('events.noComments')).toBeTruthy();
  });

  it('renders comments when comments array is provided', () => {
    const app = makeApp();
    mockUseApp.mockReturnValue(app as never);
    const event = makeEvent();
    const comments = [
      {
        id: 'c1',
        eventId: 'ev1',
        userId: 'user2',
        text: 'Tolles Event!',
        createdAt: '2026-06-01T10:00:00Z',
        name: 'Anna',
        color: '#4285F4',
        photo: null,
      },
    ];
    render(<EventDetailSheet app={app as never} sheet={{ type: 'eventDetail', event, rows: [], comments } as never} />);
    expect(screen.getByText('Tolles Event!')).toBeTruthy();
    expect(screen.getByText('Anna')).toBeTruthy();
  });

  it('renders comment count in section title when comments present', () => {
    const app = makeApp();
    mockUseApp.mockReturnValue(app as never);
    const event = makeEvent();
    const comments = [
      {
        id: 'c1',
        eventId: 'ev1',
        userId: 'user2',
        text: 'Super!',
        createdAt: '2026-06-01T10:00:00Z',
        name: 'Anna',
        color: '#4285F4',
        photo: null,
      },
    ];
    render(<EventDetailSheet app={app as never} sheet={{ type: 'eventDetail', event, rows: [], comments } as never} />);
    // Comments count appears in parentheses
    expect(screen.getByText('events.comments (1)')).toBeTruthy();
  });

  it('renders comment input field', () => {
    const app = makeApp();
    mockUseApp.mockReturnValue(app as never);
    const event = makeEvent();
    render(
      <EventDetailSheet app={app as never} sheet={{ type: 'eventDetail', event, rows: [], comments: [] } as never} />,
    );
    const input = document.querySelector('input[name="newEventComment"]');
    expect(input).toBeTruthy();
  });

  it('renders RSVP buttons when event is future and not cancelled', () => {
    const app = makeApp();
    mockUseApp.mockReturnValue(app as never);
    const event = makeEvent({ date: '2026-07-01', myStatus: 'yes' });
    render(
      <EventDetailSheet app={app as never} sheet={{ type: 'eventDetail', event, rows: [], comments: [] } as never} />,
    );
    expect(screen.getByText('events.rsvpYes')).toBeTruthy();
    expect(screen.getByText('events.rsvpMaybe')).toBeTruthy();
    expect(screen.getByText('events.rsvpNo')).toBeTruthy();
  });

  it('does not show RSVP buttons when event is in the past', () => {
    const app = makeApp();
    mockUseApp.mockReturnValue(app as never);
    const event = makeEvent({ date: '2026-01-01', myStatus: 'yes' });
    render(
      <EventDetailSheet app={app as never} sheet={{ type: 'eventDetail', event, rows: [], comments: [] } as never} />,
    );
    expect(screen.queryByText('events.rsvpYes')).toBeNull();
  });

  it('does not show RSVP buttons when event is cancelled', () => {
    const app = makeApp();
    mockUseApp.mockReturnValue(app as never);
    const event = makeEvent({ date: '2026-07-01', status: 'cancelled' });
    render(
      <EventDetailSheet app={app as never} sheet={{ type: 'eventDetail', event, rows: [], comments: [] } as never} />,
    );
    expect(screen.queryByText('events.rsvpYes')).toBeNull();
  });

  it('shows not-nominated message when myStatus is not_nominated', () => {
    const app = makeApp();
    mockUseApp.mockReturnValue(app as never);
    const event = makeEvent({ date: '2026-07-01', myStatus: 'not_nominated' });
    render(
      <EventDetailSheet app={app as never} sheet={{ type: 'eventDetail', event, rows: [], comments: [] } as never} />,
    );
    expect(screen.getByText('events.notNominated')).toBeTruthy();
  });

  // Regression test: the backend's opt_out/absence-based defaulting
  // (computeEffectiveAttendance) checks a covering planned absence BEFORE
  // responseMode, so a member with an absence covering an opt_out event's
  // date gets myStatus="no", myAuto=true -- but this banner used to check
  // responseMode==='opt_out' first, showing "you're automatically counted
  // as attending" (autoOptOut) directly above RSVP buttons that
  // simultaneously highlight "No" as selected.
  it('shows the auto-absent banner, not the auto-opt-out banner, when an opt_out event auto-defaults to no', () => {
    const app = makeApp();
    mockUseApp.mockReturnValue(app as never);
    const event = makeEvent({ date: '2026-07-01', responseMode: 'opt_out', myStatus: 'no', myAuto: true });
    render(
      <EventDetailSheet app={app as never} sheet={{ type: 'eventDetail', event, rows: [], comments: [] } as never} />,
    );
    expect(screen.getByText('events.autoAbsent')).toBeTruthy();
    expect(screen.queryByText('events.autoOptOut')).toBeNull();
  });

  it('still shows the auto-opt-out banner when an opt_out event auto-defaults to yes', () => {
    const app = makeApp();
    mockUseApp.mockReturnValue(app as never);
    const event = makeEvent({ date: '2026-07-01', responseMode: 'opt_out', myStatus: 'yes', myAuto: true });
    render(
      <EventDetailSheet app={app as never} sheet={{ type: 'eventDetail', event, rows: [], comments: [] } as never} />,
    );
    expect(screen.getByText('events.autoOptOut')).toBeTruthy();
    expect(screen.queryByText('events.autoAbsent')).toBeNull();
  });

  it('clicking RSVP yes calls setMyStatus with yes', () => {
    const app = makeApp();
    mockUseApp.mockReturnValue(app as never);
    const event = makeEvent({ date: '2026-07-01', myStatus: 'no' });
    render(
      <EventDetailSheet app={app as never} sheet={{ type: 'eventDetail', event, rows: [], comments: [] } as never} />,
    );
    fireEvent.click(screen.getByText('events.rsvpYes'));
    expect(app.setMyStatus).toHaveBeenCalledWith('ev1', 'yes', '');
  });

  it('clicking RSVP no calls setMyStatus with no', () => {
    const app = makeApp();
    mockUseApp.mockReturnValue(app as never);
    const event = makeEvent({ date: '2026-07-01', myStatus: 'yes' });
    render(
      <EventDetailSheet app={app as never} sheet={{ type: 'eventDetail', event, rows: [], comments: [] } as never} />,
    );
    fireEvent.click(screen.getByText('events.rsvpNo'));
    expect(app.setMyStatus).toHaveBeenCalledWith('ev1', 'no', '');
  });

  it('renders attendance rows when provided', () => {
    const app = makeApp();
    (app.can as ReturnType<typeof vi.fn>).mockReturnValue(false);
    mockUseApp.mockReturnValue(app as never);
    const event = makeEvent({ date: '2026-07-01' });
    const rows = [
      {
        userId: 'user2',
        name: 'Anna Müller',
        avatarColor: '#4285F4',
        photo: null,
        group: 'Gruppe A',
        primaryRole: null,
        status: 'yes' as const,
        reason: '',
        reasonId: null,
        reasonVisibility: null,
        auto: false,
        absent: false,
      },
    ];
    render(<EventDetailSheet app={app as never} sheet={{ type: 'eventDetail', event, rows, comments: [] } as never} />);
    expect(screen.getByText('Anna Müller')).toBeTruthy();
  });

  it('renders location in info box', () => {
    const app = makeApp();
    mockUseApp.mockReturnValue(app as never);
    const event = makeEvent({ location: 'Sporthalle Mitte' });
    render(
      <EventDetailSheet app={app as never} sheet={{ type: 'eventDetail', event, rows: [], comments: [] } as never} />,
    );
    expect(screen.getByText('Sporthalle Mitte')).toBeTruthy();
  });

  it('renders note when event has note', () => {
    const app = makeApp();
    mockUseApp.mockReturnValue(app as never);
    const event = makeEvent({ note: 'Bitte pünktlich kommen!' });
    render(
      <EventDetailSheet app={app as never} sheet={{ type: 'eventDetail', event, rows: [], comments: [] } as never} />,
    );
    expect(screen.getByText('Bitte pünktlich kommen!')).toBeTruthy();
  });

  it('renders result when event has result', () => {
    const app = makeApp();
    mockUseApp.mockReturnValue(app as never);
    const event = makeEvent({ result: '3:2' });
    render(
      <EventDetailSheet app={app as never} sheet={{ type: 'eventDetail', event, rows: [], comments: [] } as never} />,
    );
    // result is rendered via t('events.result', { val: '3:2' }) -> 'events.result'
    expect(screen.getByText('events.result')).toBeTruthy();
  });

  it('shows delete icon button for own comment when user is comment author', () => {
    const app = makeApp();
    (app.can as ReturnType<typeof vi.fn>).mockReturnValue(false);
    mockUseApp.mockReturnValue(app as never);
    const event = makeEvent();
    const comments = [
      {
        id: 'c1',
        eventId: 'ev1',
        userId: 'user1',
        text: 'Mein Kommentar',
        createdAt: '2026-06-01T10:00:00Z',
        name: 'Max',
        color: '#4285F4',
        photo: null,
      },
    ];
    render(<EventDetailSheet app={app as never} sheet={{ type: 'eventDetail', event, rows: [], comments } as never} />);
    expect(screen.getByText('Mein Kommentar')).toBeTruthy();
  });

  it('pressing Enter in comment input calls postEventComment', async () => {
    const app = makeApp();
    mockUseApp.mockReturnValue(app as never);
    const event = makeEvent();
    render(
      <EventDetailSheet app={app as never} sheet={{ type: 'eventDetail', event, rows: [], comments: [] } as never} />,
    );
    const input = document.querySelector('input[name="newEventComment"]') as HTMLInputElement;
    fireEvent.change(input, { target: { value: 'Tolles Spiel!' } });
    fireEvent.keyDown(input, { key: 'Enter' });
    await waitFor(() => {
      expect(app.postEventComment).toHaveBeenCalledWith('ev1', 'Tolles Spiel!');
    });
  });

  it('renders meet time row when meetTime is present', () => {
    const app = makeApp();
    mockUseApp.mockReturnValue(app as never);
    const event = makeEvent({ meetTime: '18:30' });
    render(
      <EventDetailSheet app={app as never} sheet={{ type: 'eventDetail', event, rows: [], comments: [] } as never} />,
    );
    expect(screen.getByText('events.meetTime')).toBeTruthy();
  });

  // Regression test: a failed BACKGROUND refetch (e.g. the query invalidation
  // after an unrelated attendance mutation hitting a transient network blip)
  // used to close the sheet and discard whatever was already showing, even
  // though React Query keeps the last successful `data` around during a
  // failed refetch. The sheet must keep rendering the still-valid cached
  // event instead of treating this the same as an initial-load failure.
  it('keeps showing the cached event when a background refetch fails, instead of closing the sheet', () => {
    const app = makeApp();
    mockUseApp.mockReturnValue(app as never);
    const event = makeEvent();
    mockUseEventDetailQuery.mockReturnValue({
      data: { event, rows: [], comments: [] },
      isLoading: false,
      isError: true,
      error: new Error('transient'),
    } as never);
    render(<RealEventDetailSheet app={app as never} sheet={{ type: 'eventDetail', eventId: 'ev1' } as never} />);
    expect(screen.getByText('1. Juli 2026')).toBeTruthy();
    expect(app.setState).not.toHaveBeenCalled();
  });
});
