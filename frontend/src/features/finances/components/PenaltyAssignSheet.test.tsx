import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { PenaltyAssignSheet } from './PenaltyAssignSheet';

vi.mock('@/context/AppContext', () => ({
  useApp: vi.fn(),
  useAppActions: vi.fn().mockReturnValue({}),
}));

vi.mock('@/features/members/hooks/useMemberQueries', () => ({
  useMembersQuery: vi.fn(),
}));

import { useApp } from '@/context/AppContext';
import { useMembersQuery } from '@/features/members/hooks/useMemberQueries';
const mockUseApp = vi.mocked(useApp);
const mockUseMembersQuery = useMembersQuery as ReturnType<typeof vi.fn>;

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

function makeApp(formOverrides: Record<string, unknown> = {}, members: unknown[] = [makeMember()]) {
  mockUseMembersQuery.mockReturnValue({ data: members });
  return {
    api: {},
    state: {
      primaryColor: '#4285F4',
      activeTeamId: 't1',
      form: { userId: '', penaltyId: '', ...formOverrides },
      formErrors: { userId: '', penaltyId: '' },
      busy: null,
      finances: { penalties: [makePenalty()] },
    },
    setFormErrors: vi.fn(),
    setFormVal: vi.fn(),
    onFormInput: vi.fn(),
    savePenaltyAssign: vi.fn(),
    can: vi.fn().mockReturnValue(true),
  };
}

describe('PenaltyAssignSheet', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  const sheet = {} as never;

  it('renders person select dropdown', () => {
    mockUseApp.mockReturnValue(makeApp() as never);
    const app = mockUseApp();
    render(<PenaltyAssignSheet app={app as never} sheet={sheet} />);
    expect(screen.getByRole('combobox')).toBeTruthy();
  });

  it('shows member name in select options', () => {
    mockUseApp.mockReturnValue(makeApp() as never);
    const app = mockUseApp();
    render(<PenaltyAssignSheet app={app as never} sheet={sheet} />);
    expect(screen.getByText('Anna Müller')).toBeTruthy();
  });

  it('renders penalty option buttons', () => {
    mockUseApp.mockReturnValue(makeApp() as never);
    const app = mockUseApp();
    render(<PenaltyAssignSheet app={app as never} sheet={sheet} />);
    expect(screen.getByText('Versäumtes Training')).toBeTruthy();
  });

  it('submit button is disabled when form is empty', () => {
    mockUseApp.mockReturnValue(makeApp({ userId: '', penaltyId: '' }) as never);
    const app = mockUseApp();
    render(<PenaltyAssignSheet app={app as never} sheet={sheet} />);
    expect(screen.getByRole('button', { name: /Strafe erfassen/i })).toBeDisabled();
  });

  it('submit button is enabled when userId and penaltyId set', () => {
    mockUseApp.mockReturnValue(makeApp({ userId: 'u2', penaltyId: 'p1' }) as never);
    const app = mockUseApp();
    render(<PenaltyAssignSheet app={app as never} sheet={sheet} />);
    expect(screen.getByRole('button', { name: /Strafe erfassen/i })).not.toBeDisabled();
  });

  it('clicking penalty button calls setFormVal', () => {
    const app = makeApp();
    mockUseApp.mockReturnValue(app as never);
    render(<PenaltyAssignSheet app={app as never} sheet={sheet} />);
    fireEvent.click(screen.getByText('Versäumtes Training').closest('button')!);
    expect(app.setFormVal).toHaveBeenCalledWith({ penaltyId: 'p1' });
  });

  it('validates userId field on blur', () => {
    const app = makeApp({ userId: '' });
    mockUseApp.mockReturnValue(app as never);
    render(<PenaltyAssignSheet app={app as never} sheet={sheet} />);
    fireEvent.blur(screen.getByRole('combobox'));
    expect(app.setFormErrors).toHaveBeenCalledWith({ userId: expect.stringMatching(/\S+/) });
  });

  it('handles empty finances (no penalties)', () => {
    const app = { ...makeApp(), state: { ...makeApp().state, finances: null } };
    mockUseApp.mockReturnValue(app as never);
    render(<PenaltyAssignSheet app={app as never} sheet={sheet} />);
    // Should still render without crash
    expect(screen.getByRole('combobox')).toBeTruthy();
  });
});
