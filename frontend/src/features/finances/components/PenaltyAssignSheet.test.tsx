import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { PenaltyAssignSheet } from './PenaltyAssignSheet';

vi.mock('@/context/AppContext', () => ({
  useApp: vi.fn(),
}));

vi.mock('@/features/members/hooks/useMemberQueries', () => ({
  useMembersQuery: vi.fn(),
}));

vi.mock('../hooks/useFinanceQueries', () => ({
  useFinanceOverviewQuery: vi.fn(),
}));

import { useApp } from '@/context/AppContext';
import { useMembersQuery } from '@/features/members/hooks/useMemberQueries';
import { useFinanceOverviewQuery } from '../hooks/useFinanceQueries';
const mockUseApp = vi.mocked(useApp);
const mockUseMembersQuery = useMembersQuery as ReturnType<typeof vi.fn>;
const mockUseFinanceOverviewQuery = useFinanceOverviewQuery as ReturnType<typeof vi.fn>;

const makePenalty = (overrides = {}) => ({
  id: 'p1',
  label: 'Versäumtes Training',
  amount: 10,
  ...overrides,
});

const makeMember = (overrides = {}) => ({
  membershipId: 'ms1',
  userId: 'u2',
  name: 'Anna Müller',
  email: 'anna@test.com',
  avatarColor: '#4285F4',
  photo: null,
  roles: [],
  primaryRole: null,
  joinedAt: '2025-01-01',
  ...overrides,
});

function makeApp(
  formOverrides: Record<string, unknown> = {},
  members: unknown[] = [makeMember()],
  finances: unknown = { penalties: [makePenalty()] },
) {
  mockUseMembersQuery.mockReturnValue({ data: members });
  mockUseFinanceOverviewQuery.mockReturnValue({ data: finances });
  const app = {
    api: {},
    state: {
      primaryColor: '#4285F4',
      activeTeamId: 't1',
    },
    savePenaltyAssign: vi.fn(),
    can: vi.fn().mockReturnValue(true),
    openTxForm: vi.fn(),
  };
  const formInitial = { userId: '', penaltyId: '', ...formOverrides };
  mockUseApp.mockReturnValue(app as never);
  return { app, formInitial };
}

describe('PenaltyAssignSheet', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders person select dropdown', () => {
    const { app, formInitial } = makeApp();
    render(<PenaltyAssignSheet app={app as never} sheet={{ formInitial } as never} />);
    expect(screen.getByRole('combobox')).toBeTruthy();
  });

  it('shows member name in select options', () => {
    const { app, formInitial } = makeApp();
    render(<PenaltyAssignSheet app={app as never} sheet={{ formInitial } as never} />);
    expect(screen.getByText('Anna Müller')).toBeTruthy();
  });

  it('renders penalty option buttons', () => {
    const { app, formInitial } = makeApp();
    render(<PenaltyAssignSheet app={app as never} sheet={{ formInitial } as never} />);
    expect(screen.getByText('Versäumtes Training')).toBeTruthy();
  });

  it('submit button is disabled when form is empty', () => {
    const { app, formInitial } = makeApp({ userId: '', penaltyId: '' });
    render(<PenaltyAssignSheet app={app as never} sheet={{ formInitial } as never} />);
    expect(screen.getByRole('button', { name: /Strafe erfassen/i })).toBeDisabled();
  });

  it('submit button is enabled when userId and penaltyId set', () => {
    const { app, formInitial } = makeApp({ userId: 'u2', penaltyId: 'p1' });
    render(<PenaltyAssignSheet app={app as never} sheet={{ formInitial } as never} />);
    expect(screen.getByRole('button', { name: /Strafe erfassen/i })).not.toBeDisabled();
  });

  it('clicking penalty button updates selection', () => {
    const { app, formInitial } = makeApp({ penaltyId: '' });
    render(<PenaltyAssignSheet app={app as never} sheet={{ formInitial } as never} />);
    const btn = screen.getByText('Versäumtes Training').closest('button')!;
    fireEvent.click(btn);
    expect(btn.getAttribute('aria-checked')).toBe('true');
  });

  it('validates userId field on blur', async () => {
    const { app, formInitial } = makeApp({ userId: '' });
    render(<PenaltyAssignSheet app={app as never} sheet={{ formInitial } as never} />);
    fireEvent.blur(screen.getByRole('combobox'));
    await waitFor(() => {
      expect(screen.getByText('Bitte Person wählen.')).toBeTruthy();
    });
  });

  it('handles empty finances (no penalties)', () => {
    const { app, formInitial } = makeApp({}, [makeMember()], null);
    render(<PenaltyAssignSheet app={app as never} sheet={{ formInitial } as never} />);
    expect(screen.getByRole('combobox')).toBeTruthy();
  });

  it('renders a date input defaulting to the formInitial date', () => {
    const { app, formInitial } = makeApp({ date: '2026-06-01' });
    render(<PenaltyAssignSheet app={app as never} sheet={{ formInitial } as never} />);
    const dateInput = document.querySelector('input[type="date"]') as HTMLInputElement;
    expect(dateInput).toBeTruthy();
    expect(dateInput.value).toBe('2026-06-01');
  });

  it('allows editing the date to a past value', () => {
    const { app, formInitial } = makeApp({ date: '2026-06-01' });
    render(<PenaltyAssignSheet app={app as never} sheet={{ formInitial } as never} />);
    const dateInput = document.querySelector('input[type="date"]') as HTMLInputElement;
    fireEvent.change(dateInput, { target: { value: '2026-01-15' } });
    expect(dateInput.value).toBe('2026-01-15');
  });

  it('renders an optional note field', () => {
    const { app, formInitial } = makeApp({ note: '' });
    render(<PenaltyAssignSheet app={app as never} sheet={{ formInitial } as never} />);
    expect(screen.getByPlaceholderText('z. B. Grund der Strafe')).toBeTruthy();
  });

  it('passes the entered note and date through on submit', async () => {
    const { app, formInitial } = makeApp({ userId: 'u2', penaltyId: 'p1', date: '2026-06-01', note: '' });
    render(<PenaltyAssignSheet app={app as never} sheet={{ formInitial } as never} />);
    const noteField = screen.getByPlaceholderText('z. B. Grund der Strafe');
    fireEvent.change(noteField, { target: { value: 'Zu spät gekommen' } });
    fireEvent.click(screen.getByRole('button', { name: /Strafe erfassen/i }));
    await waitFor(() => {
      expect(app.savePenaltyAssign).toHaveBeenCalledWith(
        expect.objectContaining({ userId: 'u2', penaltyId: 'p1', date: '2026-06-01', note: 'Zu spät gekommen' }),
      );
    });
  });

  it('submits successfully with the note left empty', async () => {
    const { app, formInitial } = makeApp({ userId: 'u2', penaltyId: 'p1', date: '2026-06-01', note: '' });
    render(<PenaltyAssignSheet app={app as never} sheet={{ formInitial } as never} />);
    fireEvent.click(screen.getByRole('button', { name: /Strafe erfassen/i }));
    await waitFor(() => {
      expect(app.savePenaltyAssign).toHaveBeenCalledWith(expect.objectContaining({ userId: 'u2', penaltyId: 'p1' }));
    });
  });

  describe('view mode (detail)', () => {
    it('lists transactions linked to the assignment', () => {
      const { app, formInitial } = makeApp(
        { id: 'a1' },
        [makeMember()],
        {
          penalties: [makePenalty()],
          assignments: [{ id: 'a1', name: 'Anna Müller', label: 'Versäumtes Training', date: '2026-06-01', amount: 10 }],
          transactions: [
            { id: 'tx1', title: 'Bar erhalten', date: '2026-06-02', amount: 10, penaltyAssignmentId: 'a1' },
            { id: 'tx2', title: 'Unrelated', date: '2026-06-03', amount: 5, penaltyAssignmentId: 'other' },
          ],
        },
      );
      render(<PenaltyAssignSheet app={app as never} sheet={{ mode: 'view', formInitial } as never} />);
      expect(screen.getByText('Bar erhalten')).toBeTruthy();
      expect(screen.queryByText('Unrelated')).toBeNull();
    });

    it('shows an empty state when no transaction is linked yet', () => {
      const { app, formInitial } = makeApp(
        { id: 'a1' },
        [makeMember()],
        {
          penalties: [makePenalty()],
          assignments: [{ id: 'a1', name: 'Anna Müller', label: 'Versäumtes Training', date: '2026-06-01', amount: 10 }],
          transactions: [],
        },
      );
      render(<PenaltyAssignSheet app={app as never} sheet={{ mode: 'view', formInitial } as never} />);
      expect(screen.getByText('Keine verknüpften Buchungen')).toBeTruthy();
    });

    it('clicking a linked transaction opens it via openTxForm', () => {
      const tx = { id: 'tx1', title: 'Bar erhalten', date: '2026-06-02', amount: 10, penaltyAssignmentId: 'a1' };
      const { app, formInitial } = makeApp(
        { id: 'a1' },
        [makeMember()],
        {
          penalties: [makePenalty()],
          assignments: [{ id: 'a1', name: 'Anna Müller', label: 'Versäumtes Training', date: '2026-06-01', amount: 10 }],
          transactions: [tx],
        },
      );
      app.openTxForm = vi.fn();
      render(<PenaltyAssignSheet app={app as never} sheet={{ mode: 'view', formInitial } as never} />);
      fireEvent.click(screen.getByText('Bar erhalten'));
      expect(app.openTxForm).toHaveBeenCalledWith(tx);
    });
  });
});
