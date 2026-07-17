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
});
