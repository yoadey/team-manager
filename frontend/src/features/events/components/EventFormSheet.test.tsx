import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { EventFormSheet } from './EventFormSheet';

vi.mock('@/context/AppContext', () => {
  const useApp = vi.fn();
  return { useApp };
});

vi.mock('@/styles/tokens', () => ({
  buildTokens: vi.fn().mockReturnValue({
    primary: '#4285F4',
    primaryDark: '#1565C0',
    onPrimary: '#fff',
    primaryContainer: '#D7E3FF',
    onPrimaryContainer: '#001B3E',
    error: '#B00020',
  }),
  typeMeta: vi.fn().mockImplementation((tp: string) => {
    if (tp === 'training') return { icon: 'sports', label: 'Training', color: '#1565C0', bg: '#E3F2FD' };
    if (tp === 'auftritt') return { icon: 'music_note', label: 'Auftritt', color: '#6A1B9A', bg: '#F3E5F5' };
    return { icon: 'celebration', label: 'Event', color: '#2E7D32', bg: '#E8F5E9' };
  }),
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
  hhmm: vi.fn().mockImplementation((v: string) => v || ''),
}));

vi.mock('@/i18n', () => ({
  t: vi.fn().mockImplementation((key: string) => key),
}));

vi.mock('../hooks/useEventQueries', () => ({
  useEventsQuery: vi.fn().mockReturnValue({ data: [] }),
}));

import { useApp } from '@/context/AppContext';
import { useEventsQuery } from '../hooks/useEventQueries';
const mockUseApp = vi.mocked(useApp);
const mockUseEventsQuery = vi.mocked(useEventsQuery);

function makeApp(formOverrides: Record<string, unknown> = {}) {
  return {
    state: {
      primaryColor: '#4285F4',
      form: {
        type: 'training',
        title: '',
        location: '',
        date: '',
        meetT: '',
        startT: '',
        endT: '',
        responseMode: 'opt_out',
        repeatWeeks: '',
        recurring: false,
        meetTimeMandatory: false,
        nominatedRoleIds: [],
        seriesId: null,
        note: '',
        ...formOverrides,
      },
      roles: [],
      busy: null,
    },
    saveEvent: vi.fn(),
  };
}

describe('EventFormSheet', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders event type buttons', () => {
    const app = makeApp();
    mockUseApp.mockReturnValue(app as never);
    render(<EventFormSheet app={app as never} sheet={{ type: 'eventForm', mode: 'create', formInitial: app.state.form } as never} />);
    expect(screen.getByText('events.typeTraining')).toBeTruthy();
    expect(screen.getByText('events.typeAuftritt')).toBeTruthy();
    expect(screen.getByText('events.typeEvent')).toBeTruthy();
  });

  it('renders title field', () => {
    const app = makeApp();
    mockUseApp.mockReturnValue(app as never);
    render(<EventFormSheet app={app as never} sheet={{ type: 'eventForm', mode: 'create', formInitial: app.state.form } as never} />);
    expect(screen.getByText('events.fieldTitle')).toBeTruthy();
  });

  it('renders date field', () => {
    const app = makeApp();
    mockUseApp.mockReturnValue(app as never);
    render(<EventFormSheet app={app as never} sheet={{ type: 'eventForm', mode: 'create', formInitial: app.state.form } as never} />);
    expect(screen.getByText('events.fieldDate')).toBeTruthy();
  });

  it('renders the multi-day end-date field when not recurring', () => {
    const app = makeApp();
    mockUseApp.mockReturnValue(app as never);
    render(<EventFormSheet app={app as never} sheet={{ type: 'eventForm', mode: 'create', formInitial: app.state.form } as never} />);
    expect(screen.getByText('events.fieldMultiDayEndDate')).toBeTruthy();
  });

  it('hides the multi-day end-date field once recurring is toggled on', () => {
    const app = makeApp();
    mockUseApp.mockReturnValue(app as never);
    render(<EventFormSheet app={app as never} sheet={{ type: 'eventForm', mode: 'create', formInitial: app.state.form } as never} />);
    fireEvent.click(screen.getByRole('switch', { name: 'events.recurWeekly' }));
    expect(screen.queryByText('events.fieldMultiDayEndDate')).toBeFalsy();
  });

  it('renders time fields', () => {
    const app = makeApp();
    mockUseApp.mockReturnValue(app as never);
    render(<EventFormSheet app={app as never} sheet={{ type: 'eventForm', mode: 'create', formInitial: app.state.form } as never} />);
    expect(screen.getByText('events.fieldMeetTime')).toBeTruthy();
    expect(screen.getByText('events.fieldStartTime')).toBeTruthy();
    expect(screen.getByText('events.fieldEndTime')).toBeTruthy();
  });

  it('renders response mode buttons', () => {
    const app = makeApp();
    mockUseApp.mockReturnValue(app as never);
    render(<EventFormSheet app={app as never} sheet={{ type: 'eventForm', mode: 'create', formInitial: app.state.form } as never} />);
    expect(screen.getByText('events.modeOptIn')).toBeTruthy();
    expect(screen.getByText('events.modeOptOut')).toBeTruthy();
  });

  it('renders location field', () => {
    const app = makeApp();
    mockUseApp.mockReturnValue(app as never);
    render(<EventFormSheet app={app as never} sheet={{ type: 'eventForm', mode: 'create', formInitial: app.state.form } as never} />);
    expect(screen.getByText('events.fieldLocation')).toBeTruthy();
  });

  it('suggests previously used, de-duplicated locations for the location field', () => {
    mockUseEventsQuery.mockReturnValue({
      data: [
        { location: 'Turnhalle A' },
        { location: 'Sportplatz' },
        { location: 'turnhalle a' },
        { location: '' },
      ],
    } as never);
    const app = makeApp();
    mockUseApp.mockReturnValue(app as never);
    render(<EventFormSheet app={app as never} sheet={{ type: 'eventForm', mode: 'create', formInitial: app.state.form } as never} />);
    const location = document.querySelector('input[name="location"]') as HTMLInputElement;
    expect(location.getAttribute('list')).toBe('event-location-suggestions');
    const options = Array.from(document.querySelectorAll('#event-location-suggestions option')).map((o) =>
      o.getAttribute('value'),
    );
    expect(options).toEqual(['Turnhalle A', 'Sportplatz']);
  });

  it('renders note field', () => {
    const app = makeApp();
    mockUseApp.mockReturnValue(app as never);
    render(<EventFormSheet app={app as never} sheet={{ type: 'eventForm', mode: 'create', formInitial: app.state.form } as never} />);
    expect(screen.getByText('events.fieldNote')).toBeTruthy();
  });

  it('caps location and note inputs matching the backend limits', () => {
    const app = makeApp();
    mockUseApp.mockReturnValue(app as never);
    render(<EventFormSheet app={app as never} sheet={{ type: 'eventForm', mode: 'create', formInitial: app.state.form } as never} />);
    const location = document.querySelector('input[name="location"]') as HTMLInputElement;
    const note = document.querySelector('textarea[name="note"]') as HTMLTextAreaElement;
    expect(location.maxLength).toBe(255);
    expect(note.maxLength).toBe(10000);
  });

  it('caps the title input matching the backend limit', () => {
    const app = makeApp();
    mockUseApp.mockReturnValue(app as never);
    render(<EventFormSheet app={app as never} sheet={{ type: 'eventForm', mode: 'create', formInitial: app.state.form } as never} />);
    const title = document.querySelector('input[name="title"]') as HTMLInputElement;
    expect(title.maxLength).toBe(255);
  });

  it('renders recurring toggle in create mode', () => {
    const app = makeApp();
    mockUseApp.mockReturnValue(app as never);
    render(<EventFormSheet app={app as never} sheet={{ type: 'eventForm', mode: 'create', formInitial: app.state.form } as never} />);
    expect(screen.getByText('events.recurWeekly')).toBeTruthy();
  });

  it('does not render recurring toggle in edit mode', () => {
    const app = makeApp();
    mockUseApp.mockReturnValue(app as never);
    render(<EventFormSheet app={app as never} sheet={{ type: 'eventForm', mode: 'edit', formInitial: app.state.form } as never} />);
    expect(screen.queryByText('events.recurWeekly')).toBeNull();
  });

  it('submit button is disabled when title and date are empty', () => {
    const app = makeApp({ title: '', date: '' });
    mockUseApp.mockReturnValue(app as never);
    render(<EventFormSheet app={app as never} sheet={{ type: 'eventForm', mode: 'create', formInitial: app.state.form } as never} />);
    const btn = screen.getByRole('button', { name: /events.createEvent/i });
    expect(btn).toBeDisabled();
  });

  it('submit button is enabled when title and date are filled', () => {
    const app = makeApp({ title: 'Sommerball', date: '2026-07-01' });
    mockUseApp.mockReturnValue(app as never);
    render(<EventFormSheet app={app as never} sheet={{ type: 'eventForm', mode: 'create', formInitial: app.state.form } as never} />);
    const btn = screen.getByRole('button', { name: /events.createEvent/i });
    expect(btn).not.toBeDisabled();
  });

  it('shows create label in create mode', () => {
    const app = makeApp({ title: 'Test', date: '2026-07-01' });
    mockUseApp.mockReturnValue(app as never);
    render(<EventFormSheet app={app as never} sheet={{ type: 'eventForm', mode: 'create', formInitial: app.state.form } as never} />);
    expect(screen.getByText('events.createEvent')).toBeTruthy();
  });

  it('shows save label in edit mode', () => {
    const app = makeApp({ title: 'Test', date: '2026-07-01' });
    mockUseApp.mockReturnValue(app as never);
    render(<EventFormSheet app={app as never} sheet={{ type: 'eventForm', mode: 'edit', formInitial: app.state.form } as never} />);
    expect(screen.getByText('events.saveChanges')).toBeTruthy();
  });

  it('clicking type button updates the selected button', () => {
    const app = makeApp({ type: 'training' });
    mockUseApp.mockReturnValue(app as never);
    render(<EventFormSheet app={app as never} sheet={{ type: 'eventForm', mode: 'create', formInitial: app.state.form } as never} />);
    const btn = screen.getByText('events.typeAuftritt').closest('button')!;
    fireEvent.click(btn);
    expect(btn.getAttribute('aria-pressed')).toBe('true');
  });

  it('clicking opt_in mode button updates the selected button', () => {
    const app = makeApp({ responseMode: 'opt_out' });
    mockUseApp.mockReturnValue(app as never);
    render(<EventFormSheet app={app as never} sheet={{ type: 'eventForm', mode: 'create', formInitial: app.state.form } as never} />);
    const btn = screen.getByText('events.modeOptIn').closest('button')!;
    fireEvent.click(btn);
    expect(btn.getAttribute('aria-pressed')).toBe('true');
  });

  it('clicking recurring toggle updates the toggle switch', () => {
    const app = makeApp({ recurring: false });
    mockUseApp.mockReturnValue(app as never);
    render(<EventFormSheet app={app as never} sheet={{ type: 'eventForm', mode: 'create', formInitial: app.state.form } as never} />);
    const btn = screen.getByText('events.recurWeekly').closest('button')!;
    fireEvent.click(btn);
    expect(btn.getAttribute('aria-checked')).toBe('true');
  });

  it('shows repeatWeeks field when recurring is true', () => {
    const app = makeApp({ recurring: true });
    mockUseApp.mockReturnValue(app as never);
    render(<EventFormSheet app={app as never} sheet={{ type: 'eventForm', mode: 'create', formInitial: app.state.form } as never} />);
    expect(screen.getByText('events.recurWeeks')).toBeTruthy();
  });

  it('shows the weeks/until-date toggle when recurring is true, defaulting to weeks', () => {
    const app = makeApp({ recurring: true });
    mockUseApp.mockReturnValue(app as never);
    render(<EventFormSheet app={app as never} sheet={{ type: 'eventForm', mode: 'create', formInitial: app.state.form } as never} />);
    expect(screen.getByText('events.recurModeWeeks')).toBeTruthy();
    expect(screen.getByText('events.recurModeUntil')).toBeTruthy();
    expect(screen.getByText('events.recurModeWeeks').closest('button')!.getAttribute('aria-pressed')).toBe('true');
    // Weeks is the default -- the repeatWeeks number input shows, not the end-date picker.
    expect(document.querySelector('input[name="repeatWeeks"]')).toBeTruthy();
    expect(document.querySelector('input[name="repeatEndDate"]')).toBeNull();
  });

  it('switching the recurrence toggle to "until date" shows the end-date picker instead of repeatWeeks', () => {
    const app = makeApp({ recurring: true });
    mockUseApp.mockReturnValue(app as never);
    render(<EventFormSheet app={app as never} sheet={{ type: 'eventForm', mode: 'create', formInitial: app.state.form } as never} />);
    fireEvent.click(screen.getByText('events.recurModeUntil').closest('button')!);
    expect(document.querySelector('input[name="repeatEndDate"]')).toBeTruthy();
    expect(document.querySelector('input[name="repeatWeeks"]')).toBeNull();
    expect(screen.getByText('events.recurEndDate')).toBeTruthy();
  });

  it('clicking meetTimeMandatory toggle updates the checkbox', () => {
    const app = makeApp({ meetTimeMandatory: false });
    mockUseApp.mockReturnValue(app as never);
    render(<EventFormSheet app={app as never} sheet={{ type: 'eventForm', mode: 'create', formInitial: app.state.form } as never} />);
    const btn = screen.getByText('events.meetTimeMandatory').closest('button')!;
    fireEvent.click(btn);
    expect(btn.getAttribute('aria-checked')).toBe('true');
  });

  it('clicking submit calls saveEvent', async () => {
    const app = makeApp({ title: 'Test Event', date: '2026-07-01' });
    mockUseApp.mockReturnValue(app as never);
    render(<EventFormSheet app={app as never} sheet={{ type: 'eventForm', mode: 'create', formInitial: app.state.form } as never} />);
    fireEvent.click(screen.getByRole('button', { name: /events.createEvent/i }));
    await waitFor(() => {
      expect(app.saveEvent).toHaveBeenCalled();
    });
  });

  it('renders nominated roles label', () => {
    const app = makeApp();
    mockUseApp.mockReturnValue(app as never);
    render(<EventFormSheet app={app as never} sheet={{ type: 'eventForm', mode: 'create', formInitial: app.state.form } as never} />);
    expect(screen.getByText('events.nominatedRoles')).toBeTruthy();
  });

  it('renders role chips when roles exist', () => {
    const app = {
      ...makeApp(),
      state: {
        ...makeApp().state,
        roles: [
          { id: 'r1', name: 'Musiker', color: '#4285F4', teamId: 't1', system: false, permissions: {} },
          { id: 'r2', name: 'Dirigent', color: '#B71C1C', teamId: 't1', system: false, permissions: {} },
        ],
      },
    };
    mockUseApp.mockReturnValue(app as never);
    render(<EventFormSheet app={app as never} sheet={{ type: 'eventForm', mode: 'create', formInitial: app.state.form } as never} />);
    expect(screen.getByText('Musiker')).toBeTruthy();
    expect(screen.getByText('Dirigent')).toBeTruthy();
  });

  it('clicking role chip updates the selection state', () => {
    const app = {
      ...makeApp(),
      state: {
        ...makeApp().state,
        roles: [{ id: 'r1', name: 'Musiker', color: '#4285F4', teamId: 't1', system: false, permissions: {} }],
      },
    };
    mockUseApp.mockReturnValue(app as never);
    render(<EventFormSheet app={app as never} sheet={{ type: 'eventForm', mode: 'create', formInitial: app.state.form } as never} />);
    const btn = screen.getByText('Musiker').closest('button')!;
    fireEvent.click(btn);
    expect(btn.getAttribute('aria-checked')).toBe('true');
  });

  it('shows series buttons in edit mode when seriesId is set', () => {
    const app = makeApp({ title: 'Test', date: '2026-07-01', seriesId: '123e4567-e89b-12d3-a456-426614174000' });
    mockUseApp.mockReturnValue(app as never);
    render(<EventFormSheet app={app as never} sheet={{ type: 'eventForm', mode: 'edit', formInitial: app.state.form } as never} />);
    expect(screen.getByText('events.seriesSingle')).toBeTruthy();
    expect(screen.getByText('events.seriesAll')).toBeTruthy();
  });

  it('clicking seriesSingle calls saveEvent with single', async () => {
    const app = makeApp({ title: 'Test', date: '2026-07-01', seriesId: '123e4567-e89b-12d3-a456-426614174000' });
    mockUseApp.mockReturnValue(app as never);
    render(<EventFormSheet app={app as never} sheet={{ type: 'eventForm', mode: 'edit', formInitial: app.state.form } as never} />);
    fireEvent.click(screen.getByText('events.seriesSingle').closest('button')!);
    await waitFor(() => {
      expect(app.saveEvent).toHaveBeenCalledWith(expect.any(Object), 'single');
    });
  });

  it('clicking seriesAll calls saveEvent with series', async () => {
    const app = makeApp({ title: 'Test', date: '2026-07-01', seriesId: '123e4567-e89b-12d3-a456-426614174000' });
    mockUseApp.mockReturnValue(app as never);
    render(<EventFormSheet app={app as never} sheet={{ type: 'eventForm', mode: 'edit', formInitial: app.state.form } as never} />);
    fireEvent.click(screen.getByText('events.seriesAll').closest('button')!);
    await waitFor(() => {
      expect(app.saveEvent).toHaveBeenCalledWith(expect.any(Object), 'series');
    });
  });

  it('validates title on blur when title is empty', async () => {
    const app = makeApp({ title: '' });
    mockUseApp.mockReturnValue(app as never);
    render(<EventFormSheet app={app as never} sheet={{ type: 'eventForm', mode: 'create', formInitial: app.state.form } as never} />);
    const titleInput = document.querySelector('input[name="title"]') as HTMLInputElement;
    fireEvent.blur(titleInput);
    await waitFor(() => {
      expect(screen.getByText('validation.eventTitleMissing')).toBeTruthy();
    });
  });

  it('renders event type section label', () => {
    const app = makeApp();
    mockUseApp.mockReturnValue(app as never);
    render(<EventFormSheet app={app as never} sheet={{ type: 'eventForm', mode: 'create', formInitial: app.state.form } as never} />);
    expect(screen.getByText('events.eventType')).toBeTruthy();
  });

  it('renders response mode section label', () => {
    const app = makeApp();
    mockUseApp.mockReturnValue(app as never);
    render(<EventFormSheet app={app as never} sheet={{ type: 'eventForm', mode: 'create', formInitial: app.state.form } as never} />);
    expect(screen.getByText('events.responseMode')).toBeTruthy();
  });

  it('renders the cancellation lead-time hours/minutes fields with visible, always-on unit labels', () => {
    const app = makeApp();
    mockUseApp.mockReturnValue(app as never);
    render(<EventFormSheet app={app as never} sheet={{ type: 'eventForm', mode: 'create', formInitial: app.state.form } as never} />);
    expect(screen.getByText('events.fieldCancelLeadTime')).toBeTruthy();
    expect(screen.getByText('events.fieldCancelLeadHours')).toBeTruthy();
    expect(screen.getByText('events.fieldCancelLeadMinutes')).toBeTruthy();
    expect(screen.getByText('events.fieldCancelLeadHoursShort')).toBeTruthy();
    expect(screen.getByText('events.fieldCancelLeadMinutesShort')).toBeTruthy();

    const hoursInput = document.querySelector('input[name="cancelLeadHours"]') as HTMLInputElement;
    const minutesInput = document.querySelector('input[name="cancelLeadMinutes"]') as HTMLInputElement;
    expect(hoursInput.type).toBe('number');
    expect(minutesInput.type).toBe('number');
  });

  it('validates repeatEndDate on blur when left empty in "until date" mode', async () => {
    const app = makeApp({ recurring: true, date: '2026-07-01' });
    mockUseApp.mockReturnValue(app as never);
    render(<EventFormSheet app={app as never} sheet={{ type: 'eventForm', mode: 'create', formInitial: app.state.form } as never} />);
    fireEvent.click(screen.getByText('events.recurModeUntil').closest('button')!);
    const endDateInput = document.querySelector('input[name="repeatEndDate"]') as HTMLInputElement;
    fireEvent.blur(endDateInput);
    await waitFor(() => {
      expect(screen.getByText('validation.eventRepeatEndDateInvalid')).toBeTruthy();
    });
  });

  it('validates repeatEndDate on blur when it falls before the event date', async () => {
    const app = makeApp({ recurring: true, date: '2026-07-10' });
    mockUseApp.mockReturnValue(app as never);
    render(<EventFormSheet app={app as never} sheet={{ type: 'eventForm', mode: 'create', formInitial: app.state.form } as never} />);
    fireEvent.click(screen.getByText('events.recurModeUntil').closest('button')!);
    const endDateInput = document.querySelector('input[name="repeatEndDate"]') as HTMLInputElement;
    fireEvent.change(endDateInput, { target: { value: '2026-07-01' } });
    fireEvent.blur(endDateInput);
    await waitFor(() => {
      expect(screen.getByText('validation.eventRepeatEndDateBeforeStart')).toBeTruthy();
    });
  });
});
