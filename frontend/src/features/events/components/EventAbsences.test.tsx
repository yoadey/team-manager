import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { EventAbsences } from './EventAbsences';

vi.mock('@/context/AppContext', () => ({
  useApp: vi.fn(),
  useAppActions: vi.fn().mockReturnValue({}),
}));

// Mocked directly on the hooks module (not just a `@/features/events` barrel
// re-export) -- EventAbsences.tsx imports `useAbsencesQuery` via this exact
// relative path, so this must match it (see the identical comment/pattern in
// PollsPage.test.tsx/NewsPage.test.tsx).
vi.mock('../hooks/useAbsenceQueries', () => ({
  useAbsencesQuery: vi.fn(),
}));

import { useApp } from '@/context/AppContext';
import { useAbsencesQuery } from '../hooks/useAbsenceQueries';
const mockUseApp = vi.mocked(useApp);
const mockUseAbsencesQuery = vi.mocked(useAbsencesQuery);

function makeAbsence(overrides: Record<string, unknown> = {}) {
  return {
    id: 'abs1',
    userId: 'u2',
    name: 'Anna Müller',
    photo: null,
    avatarColor: '#4285F4',
    roleColor: '#E91E63',
    from: '2099-01-10',
    to: '2099-01-20',
    reason: 'Urlaub',
    notRelevantForStats: false,
    ...overrides,
  };
}

function makeApp(overrides: Record<string, unknown> = {}) {
  const { absences, ...stateOverrides } = overrides;
  mockUseAbsencesQuery.mockReturnValue({ data: 'absences' in overrides ? absences : [] } as never);
  return {
    api: {},
    state: {
      primaryColor: '#4285F4',
      activeTeamId: 't1',
      user: { id: 'u1' },
      ...stateOverrides,
    },
    can: vi.fn().mockReturnValue(false),
    openAbsenceForm: vi.fn(),
    removeAbsence: vi.fn(),
    setAbsenceStatsRelevance: vi.fn(),
  };
}

describe('EventAbsences', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('shows spinner when absences is null', () => {
    mockUseApp.mockReturnValue(makeApp({ absences: null }) as never);
    const { container } = render(<EventAbsences />);
    // SpinnerBox renders a role="status" element
    expect(container.querySelector('[role="status"]')).toBeTruthy();
  });

  it('renders add absence button when absences loaded', () => {
    mockUseApp.mockReturnValue(makeApp({ absences: [] }) as never);
    render(<EventAbsences />);
    expect(screen.getByText('Eigene Abwesenheit eintragen')).toBeTruthy();
  });

  it('shows empty state when no upcoming absences', () => {
    mockUseApp.mockReturnValue(makeApp({ absences: [] }) as never);
    render(<EventAbsences />);
    // EmptyState renders when list is empty
    expect(screen.getByText(/Keine geplanten Abwesenheiten/i)).toBeTruthy();
  });

  it('renders absence rows for upcoming absences', () => {
    const absence = makeAbsence();
    mockUseApp.mockReturnValue(makeApp({ absences: [absence] }) as never);
    render(<EventAbsences />);
    expect(screen.getByText('Anna Müller')).toBeTruthy();
    expect(screen.getByText(/Urlaub/)).toBeTruthy();
  });

  // Regression test: an absence with no reason (a genuinely optional field,
  // and now easy to save empty since round 75 removed the hardcoded
  // 'Urlaub' default) used to unconditionally append " · " + reason,
  // leaving a dangling trailing separator like "Jan 10 – Jan 20 · " with
  // nothing after it -- EventDetailSheet.tsx already has the correct
  // conditional pattern nearby for the same class of optional trailing text.
  it('omits the separator when the absence has no reason', () => {
    const absence = makeAbsence({ reason: '' });
    mockUseApp.mockReturnValue(makeApp({ absences: [absence] }) as never);
    render(<EventAbsences />);
    expect(screen.queryByText(/·/)).toBeNull();
  });

  it('filters out past absences (to < today)', () => {
    const past = makeAbsence({ to: '2020-01-01' });
    mockUseApp.mockReturnValue(makeApp({ absences: [past] }) as never);
    render(<EventAbsences />);
    expect(screen.queryByText('Anna Müller')).toBeNull();
  });

  it('shows "Du" chip and edit/delete buttons for own absence', () => {
    const myAbsence = makeAbsence({ userId: 'u1' });
    mockUseApp.mockReturnValue(makeApp({ absences: [myAbsence] }) as never);
    render(<EventAbsences />);
    expect(screen.getByText('Du')).toBeTruthy();
  });

  it('does not show edit/delete buttons for other user absence', () => {
    const otherAbsence = makeAbsence({ userId: 'u2' });
    const app = makeApp({ absences: [otherAbsence] });
    mockUseApp.mockReturnValue(app as never);
    render(<EventAbsences />);
    // No "Du" chip for other user
    expect(screen.queryByText('Du')).toBeNull();
  });

  it('clicking add absence button calls openAbsenceForm', async () => {
    const app = makeApp({ absences: [] });
    mockUseApp.mockReturnValue(app as never);
    render(<EventAbsences />);
    await userEvent.click(screen.getByText('Eigene Abwesenheit eintragen').closest('button')!);
    expect(app.openAbsenceForm).toHaveBeenCalledWith();
  });

  it('clicking edit button on own absence calls openAbsenceForm with absence', async () => {
    const myAbsence = makeAbsence({ userId: 'u1' });
    const app = makeApp({ absences: [myAbsence] });
    mockUseApp.mockReturnValue(app as never);
    render(<EventAbsences />);
    await userEvent.click(screen.getByLabelText('Abwesenheit bearbeiten'));
    expect(app.openAbsenceForm).toHaveBeenCalledWith(myAbsence);
  });

  it('clicking remove button on own absence calls removeAbsence', async () => {
    const myAbsence = makeAbsence({ userId: 'u1', id: 'ab1' });
    const app = makeApp({ absences: [myAbsence] });
    mockUseApp.mockReturnValue(app as never);
    render(<EventAbsences />);
    await userEvent.click(screen.getByLabelText('Abwesenheit löschen'));
    expect(app.removeAbsence).toHaveBeenCalledWith('ab1');
  });

  it('shows the stats-relevance toggle for own absence and toggles it on click', async () => {
    const myAbsence = makeAbsence({ userId: 'u1' });
    const app = makeApp({ absences: [myAbsence] });
    mockUseApp.mockReturnValue(app as never);
    render(<EventAbsences />);
    const toggle = screen.getByLabelText('Als nicht statistikrelevant markieren');
    await userEvent.click(toggle);
    expect(app.setAbsenceStatsRelevance).toHaveBeenCalledWith('abs1', true);
  });

  it('shows the stats-relevance toggle for another member when the viewer holds events:write', () => {
    const otherAbsence = makeAbsence({ userId: 'u2' });
    const app = makeApp({ absences: [otherAbsence] });
    (app.can as ReturnType<typeof vi.fn>).mockReturnValue(true);
    mockUseApp.mockReturnValue(app as never);
    render(<EventAbsences />);
    expect(screen.getByLabelText('Als nicht statistikrelevant markieren')).toBeTruthy();
  });

  it('hides the stats-relevance toggle for another member without events:write', () => {
    const otherAbsence = makeAbsence({ userId: 'u2' });
    const app = makeApp({ absences: [otherAbsence] });
    mockUseApp.mockReturnValue(app as never);
    render(<EventAbsences />);
    expect(screen.queryByLabelText('Als nicht statistikrelevant markieren')).toBeNull();
  });
});
