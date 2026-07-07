import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { EventFormSheet } from './EventFormSheet';

vi.mock('@/context/AppContext', () => {
  const useApp = vi.fn();
  return {
    useApp,
    // Actions + selector derive from the per-test useApp mock so migrated
    // atoms (TextInput/TextArea via useAppActions/useAppSelector) resolve.
    useAppActions: vi.fn(() => useApp()),
    useAppSelector: (sel: (s: { form: Record<string, unknown> }) => unknown) => sel(useApp().state),
  };
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

import { useApp } from '@/context/AppContext';
const mockUseApp = vi.mocked(useApp);

function makeApp(formOverrides: Record<string, unknown> = {}, errOverrides: Record<string, string> = {}) {
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
      formErrors: { title: '', date: '', ...errOverrides },
      roles: [],
      busy: null,
    },
    setFormErrors: vi.fn(),
    setFormVal: vi.fn(),
    onFormInput: vi.fn(),
    saveEvent: vi.fn(),
    toggleFormNomRole: vi.fn(),
  };
}

describe('EventFormSheet', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders event type buttons', () => {
    const app = makeApp();
    mockUseApp.mockReturnValue(app as never);
    render(<EventFormSheet app={app as never} sheet={{ type: 'eventForm', mode: 'create' } as never} />);
    expect(screen.getByText('events.typeTraining')).toBeTruthy();
    expect(screen.getByText('events.typeAuftritt')).toBeTruthy();
    expect(screen.getByText('events.typeEvent')).toBeTruthy();
  });

  it('renders title field', () => {
    const app = makeApp();
    mockUseApp.mockReturnValue(app as never);
    render(<EventFormSheet app={app as never} sheet={{ type: 'eventForm', mode: 'create' } as never} />);
    expect(screen.getByText('events.fieldTitle')).toBeTruthy();
  });

  it('renders date field', () => {
    const app = makeApp();
    mockUseApp.mockReturnValue(app as never);
    render(<EventFormSheet app={app as never} sheet={{ type: 'eventForm', mode: 'create' } as never} />);
    expect(screen.getByText('events.fieldDate')).toBeTruthy();
  });

  it('renders time fields', () => {
    const app = makeApp();
    mockUseApp.mockReturnValue(app as never);
    render(<EventFormSheet app={app as never} sheet={{ type: 'eventForm', mode: 'create' } as never} />);
    expect(screen.getByText('events.fieldMeetTime')).toBeTruthy();
    expect(screen.getByText('events.fieldStartTime')).toBeTruthy();
    expect(screen.getByText('events.fieldEndTime')).toBeTruthy();
  });

  it('renders response mode buttons', () => {
    const app = makeApp();
    mockUseApp.mockReturnValue(app as never);
    render(<EventFormSheet app={app as never} sheet={{ type: 'eventForm', mode: 'create' } as never} />);
    expect(screen.getByText('events.modeOptIn')).toBeTruthy();
    expect(screen.getByText('events.modeOptOut')).toBeTruthy();
  });

  it('renders location field', () => {
    const app = makeApp();
    mockUseApp.mockReturnValue(app as never);
    render(<EventFormSheet app={app as never} sheet={{ type: 'eventForm', mode: 'create' } as never} />);
    expect(screen.getByText('events.fieldLocation')).toBeTruthy();
  });

  it('renders note field', () => {
    const app = makeApp();
    mockUseApp.mockReturnValue(app as never);
    render(<EventFormSheet app={app as never} sheet={{ type: 'eventForm', mode: 'create' } as never} />);
    expect(screen.getByText('events.fieldNote')).toBeTruthy();
  });

  // Regression test: location/note had no client-side maxLength, unlike
  // every other create/edit form field, matching the backend's
  // validate.MaxLen bounds (255 / 10000).
  it('caps location and note inputs matching the backend limits', () => {
    const app = makeApp();
    mockUseApp.mockReturnValue(app as never);
    render(<EventFormSheet app={app as never} sheet={{ type: 'eventForm', mode: 'create' } as never} />);
    const location = document.querySelector('input[name="location"]') as HTMLInputElement;
    const note = document.querySelector('textarea[name="note"]') as HTMLTextAreaElement;
    expect(location.maxLength).toBe(255);
    expect(note.maxLength).toBe(10000);
  });

  it('renders recurring toggle in create mode', () => {
    const app = makeApp();
    mockUseApp.mockReturnValue(app as never);
    render(<EventFormSheet app={app as never} sheet={{ type: 'eventForm', mode: 'create' } as never} />);
    expect(screen.getByText('events.recurWeekly')).toBeTruthy();
  });

  it('does not render recurring toggle in edit mode', () => {
    const app = makeApp();
    mockUseApp.mockReturnValue(app as never);
    render(<EventFormSheet app={app as never} sheet={{ type: 'eventForm', mode: 'edit' } as never} />);
    expect(screen.queryByText('events.recurWeekly')).toBeNull();
  });

  it('submit button is disabled when title and date are empty', () => {
    const app = makeApp({ title: '', date: '' });
    mockUseApp.mockReturnValue(app as never);
    render(<EventFormSheet app={app as never} sheet={{ type: 'eventForm', mode: 'create' } as never} />);
    const btn = screen.getByRole('button', { name: /events.createEvent/i });
    expect(btn).toBeDisabled();
  });

  it('submit button is enabled when title and date are filled', () => {
    const app = makeApp({ title: 'Sommerball', date: '2026-07-01' });
    mockUseApp.mockReturnValue(app as never);
    render(<EventFormSheet app={app as never} sheet={{ type: 'eventForm', mode: 'create' } as never} />);
    const btn = screen.getByRole('button', { name: /events.createEvent/i });
    expect(btn).not.toBeDisabled();
  });

  it('shows create label in create mode', () => {
    const app = makeApp({ title: 'Test', date: '2026-07-01' });
    mockUseApp.mockReturnValue(app as never);
    render(<EventFormSheet app={app as never} sheet={{ type: 'eventForm', mode: 'create' } as never} />);
    expect(screen.getByText('events.createEvent')).toBeTruthy();
  });

  it('shows save label in edit mode', () => {
    const app = makeApp({ title: 'Test', date: '2026-07-01' });
    mockUseApp.mockReturnValue(app as never);
    render(<EventFormSheet app={app as never} sheet={{ type: 'eventForm', mode: 'edit' } as never} />);
    expect(screen.getByText('events.saveChanges')).toBeTruthy();
  });

  it('clicking type button calls setFormVal', () => {
    const app = makeApp({ type: 'training' });
    mockUseApp.mockReturnValue(app as never);
    render(<EventFormSheet app={app as never} sheet={{ type: 'eventForm', mode: 'create' } as never} />);
    fireEvent.click(screen.getByText('events.typeAuftritt').closest('button')!);
    expect(app.setFormVal).toHaveBeenCalledWith({ type: 'auftritt' });
  });

  it('clicking opt_in mode button calls setFormVal', () => {
    const app = makeApp({ responseMode: 'opt_out' });
    mockUseApp.mockReturnValue(app as never);
    render(<EventFormSheet app={app as never} sheet={{ type: 'eventForm', mode: 'create' } as never} />);
    fireEvent.click(screen.getByText('events.modeOptIn').closest('button')!);
    expect(app.setFormVal).toHaveBeenCalledWith({ responseMode: 'opt_in' });
  });

  it('clicking recurring toggle calls setFormVal', () => {
    const app = makeApp({ recurring: false });
    mockUseApp.mockReturnValue(app as never);
    render(<EventFormSheet app={app as never} sheet={{ type: 'eventForm', mode: 'create' } as never} />);
    fireEvent.click(screen.getByText('events.recurWeekly').closest('button')!);
    expect(app.setFormVal).toHaveBeenCalledWith({ recurring: true });
  });

  it('shows repeatWeeks field when recurring is true', () => {
    const app = makeApp({ recurring: true });
    mockUseApp.mockReturnValue(app as never);
    render(<EventFormSheet app={app as never} sheet={{ type: 'eventForm', mode: 'create' } as never} />);
    expect(screen.getByText('events.recurWeeks')).toBeTruthy();
  });

  it('clicking meetTimeMandatory toggle calls setFormVal', () => {
    const app = makeApp({ meetTimeMandatory: false });
    mockUseApp.mockReturnValue(app as never);
    render(<EventFormSheet app={app as never} sheet={{ type: 'eventForm', mode: 'create' } as never} />);
    fireEvent.click(screen.getByText('events.meetTimeMandatory').closest('button')!);
    expect(app.setFormVal).toHaveBeenCalledWith({ meetTimeMandatory: true });
  });

  it('clicking submit calls saveEvent', () => {
    const app = makeApp({ title: 'Test Event', date: '2026-07-01' });
    mockUseApp.mockReturnValue(app as never);
    render(<EventFormSheet app={app as never} sheet={{ type: 'eventForm', mode: 'create' } as never} />);
    fireEvent.click(screen.getByRole('button', { name: /events.createEvent/i }));
    expect(app.saveEvent).toHaveBeenCalledWith('single');
  });

  it('shows title error when title error is set', () => {
    const app = makeApp({ title: '' }, { title: 'events.fieldTitleError' });
    mockUseApp.mockReturnValue(app as never);
    render(<EventFormSheet app={app as never} sheet={{ type: 'eventForm', mode: 'create' } as never} />);
    expect(screen.getByText('events.fieldTitleError')).toBeTruthy();
  });

  it('renders nominated roles label', () => {
    const app = makeApp();
    mockUseApp.mockReturnValue(app as never);
    render(<EventFormSheet app={app as never} sheet={{ type: 'eventForm', mode: 'create' } as never} />);
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
    render(<EventFormSheet app={app as never} sheet={{ type: 'eventForm', mode: 'create' } as never} />);
    expect(screen.getByText('Musiker')).toBeTruthy();
    expect(screen.getByText('Dirigent')).toBeTruthy();
  });

  it('clicking role chip calls toggleFormNomRole', () => {
    const app = {
      ...makeApp(),
      state: {
        ...makeApp().state,
        roles: [{ id: 'r1', name: 'Musiker', color: '#4285F4', teamId: 't1', system: false, permissions: {} }],
      },
      toggleFormNomRole: vi.fn(),
    };
    mockUseApp.mockReturnValue(app as never);
    render(<EventFormSheet app={app as never} sheet={{ type: 'eventForm', mode: 'create' } as never} />);
    fireEvent.click(screen.getByText('Musiker').closest('button')!);
    expect(app.toggleFormNomRole).toHaveBeenCalledWith('r1');
  });

  it('shows series buttons in edit mode when seriesId is set', () => {
    const app = makeApp({ title: 'Test', date: '2026-07-01', seriesId: 'series1' });
    mockUseApp.mockReturnValue(app as never);
    render(<EventFormSheet app={app as never} sheet={{ type: 'eventForm', mode: 'edit' } as never} />);
    expect(screen.getByText('events.seriesSingle')).toBeTruthy();
    expect(screen.getByText('events.seriesAll')).toBeTruthy();
  });

  it('clicking seriesSingle calls saveEvent with single', () => {
    const app = makeApp({ title: 'Test', date: '2026-07-01', seriesId: 'series1' });
    mockUseApp.mockReturnValue(app as never);
    render(<EventFormSheet app={app as never} sheet={{ type: 'eventForm', mode: 'edit' } as never} />);
    fireEvent.click(screen.getByText('events.seriesSingle').closest('button')!);
    expect(app.saveEvent).toHaveBeenCalledWith('single');
  });

  it('clicking seriesAll calls saveEvent with series', () => {
    const app = makeApp({ title: 'Test', date: '2026-07-01', seriesId: 'series1' });
    mockUseApp.mockReturnValue(app as never);
    render(<EventFormSheet app={app as never} sheet={{ type: 'eventForm', mode: 'edit' } as never} />);
    fireEvent.click(screen.getByText('events.seriesAll').closest('button')!);
    expect(app.saveEvent).toHaveBeenCalledWith('series');
  });

  it('validates title on blur when title is empty', () => {
    const app = makeApp({ title: '' });
    mockUseApp.mockReturnValue(app as never);
    render(<EventFormSheet app={app as never} sheet={{ type: 'eventForm', mode: 'create' } as never} />);
    const titleInput = document.querySelector('input[name="title"]') as HTMLInputElement;
    fireEvent.blur(titleInput);
    expect(app.setFormErrors).toHaveBeenCalledWith({ title: expect.stringMatching(/\S+/) });
  });

  it('clears title error on blur when title has value', () => {
    const app = makeApp({ title: 'Sommerball' });
    mockUseApp.mockReturnValue(app as never);
    render(<EventFormSheet app={app as never} sheet={{ type: 'eventForm', mode: 'create' } as never} />);
    const titleInput = document.querySelector('input[name="title"]') as HTMLInputElement;
    fireEvent.blur(titleInput);
    expect(app.setFormErrors).toHaveBeenCalledWith({ title: '' });
  });

  it('renders event type section label', () => {
    const app = makeApp();
    mockUseApp.mockReturnValue(app as never);
    render(<EventFormSheet app={app as never} sheet={{ type: 'eventForm', mode: 'create' } as never} />);
    expect(screen.getByText('events.eventType')).toBeTruthy();
  });

  it('renders response mode section label', () => {
    const app = makeApp();
    mockUseApp.mockReturnValue(app as never);
    render(<EventFormSheet app={app as never} sheet={{ type: 'eventForm', mode: 'create' } as never} />);
    expect(screen.getByText('events.responseMode')).toBeTruthy();
  });
});
