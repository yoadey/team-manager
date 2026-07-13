import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { EventAbsences } from './EventAbsences';

vi.mock('@/context/AppContext', () => ({
  useApp: vi.fn(),
  useAppActions: vi.fn().mockReturnValue({}),
}));

import { useApp } from '@/context/AppContext';
const mockUseApp = vi.mocked(useApp);

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
    ...overrides,
  };
}

function makeApp(overrides: Record<string, unknown> = {}) {
  return {
    state: {
      primaryColor: '#4285F4',
      absences: null,
      user: { id: 'u1' },
      ...overrides,
    },
    openAbsenceForm: vi.fn(),
    removeAbsence: vi.fn(),
  };
}

describe('EventAbsences', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('shows spinner when absences is null', () => {
    mockUseApp.mockReturnValue(makeApp() as never);
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
    const buttons = document.querySelectorAll('button');
    // Find the edit button (first icon button in the absence row)
    const absenceRowBtns = Array.from(buttons).filter((b) => !b.textContent?.includes('Abwesenheit'));
    if (absenceRowBtns.length > 0) {
      await userEvent.click(absenceRowBtns[0]);
      expect(app.openAbsenceForm).toHaveBeenCalledWith(myAbsence);
    }
  });

  it('clicking remove button on own absence calls removeAbsence', async () => {
    const myAbsence = makeAbsence({ userId: 'u1', id: 'ab1' });
    const app = makeApp({ absences: [myAbsence] });
    mockUseApp.mockReturnValue(app as never);
    render(<EventAbsences />);
    const buttons = document.querySelectorAll('button');
    // The remove button follows the edit button in the DOM
    const absenceRowBtns = Array.from(buttons).filter((b) => !b.textContent?.includes('Abwesenheit'));
    if (absenceRowBtns.length > 1) {
      await userEvent.click(absenceRowBtns[1]);
      expect(app.removeAbsence).toHaveBeenCalledWith('ab1');
    }
  });
});
