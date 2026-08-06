import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { ContribCreateSheet } from './ContribCreateSheet';

vi.mock('@/context/AppContext', () => ({
  useApp: vi.fn(),
}));

vi.mock('@/features/members/hooks/useMemberQueries', () => ({
  useMembersQuery: vi.fn(),
}));

import { useApp } from '@/context/AppContext';
import { useMembersQuery } from '@/features/members/hooks/useMemberQueries';
const mockUseApp = vi.mocked(useApp);
const mockUseMembersQuery = useMembersQuery as ReturnType<typeof vi.fn>;

const makeMember = (overrides = {}) => ({
  membershipId: 'ms1',
  userId: 'u1',
  name: 'Anna Müller',
  email: 'anna@test.com',
  avatarColor: '#4285F4',
  photo: null,
  roles: [],
  primaryRole: null,
  joinedAt: '2025-01-01',
  ...overrides,
});

function makeApp(formOverrides: Record<string, unknown> = {}, members: unknown[] = [makeMember(), makeMember({ membershipId: 'ms2', userId: 'u2', name: 'Bob Fischer' })]) {
  mockUseMembersQuery.mockReturnValue({ data: members });
  const app = {
    api: {},
    state: {
      primaryColor: '#4285F4',
      activeTeamId: 't1',
      savingContribCreate: false,
    },
    saveContribCreate: vi.fn(),
  };
  mockUseApp.mockReturnValue(app as unknown as ReturnType<typeof useApp>);
  const formInitial = { label: '', amount: '', dueDate: '', userIds: [], ...formOverrides };
  return { app, formInitial };
}

describe('ContribCreateSheet', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders name, amount, and due date fields', () => {
    const { app, formInitial } = makeApp();
    render(<ContribCreateSheet app={app as never} sheet={{ formInitial } as never} />);
    expect(screen.getByPlaceholderText('z. B. Mitgliedsbeitrag Januar 2026')).toBeTruthy();
    expect(document.querySelector('input[type="number"]')).toBeTruthy();
    expect(document.querySelector('input[type="date"]')).toBeTruthy();
  });

  it('lists every member as a selectable option', () => {
    const { app, formInitial } = makeApp();
    render(<ContribCreateSheet app={app as never} sheet={{ formInitial } as never} />);
    expect(screen.getByText('Anna Müller')).toBeTruthy();
    expect(screen.getByText('Bob Fischer')).toBeTruthy();
  });

  it('submit is disabled until a name, a valid amount, and at least one member are set', () => {
    const { app, formInitial } = makeApp();
    render(<ContribCreateSheet app={app as never} sheet={{ formInitial } as never} />);
    const btn = screen.getByRole('button', { name: /Beitrag anlegen/i });
    expect(btn).toBeDisabled();
  });

  it('toggling a member enables submit once name and amount are also valid', () => {
    const { app, formInitial } = makeApp({ label: 'Mitgliedsbeitrag', amount: '25' });
    render(<ContribCreateSheet app={app as never} sheet={{ formInitial } as never} />);
    const btn = screen.getByRole('button', { name: /Beitrag anlegen/i });
    expect(btn).toBeDisabled();
    fireEvent.click(screen.getByText('Anna Müller').closest('button')!);
    expect(btn).not.toBeDisabled();
  });

  it('"select all" selects every member, and toggles to "deselect all"', () => {
    const { app, formInitial } = makeApp({ label: 'Mitgliedsbeitrag', amount: '25' });
    render(<ContribCreateSheet app={app as never} sheet={{ formInitial } as never} />);
    fireEvent.click(screen.getByText('Alle auswählen'));
    const btn = screen.getByRole('button', { name: /Beitrag anlegen/i });
    expect(btn).not.toBeDisabled();
    expect(screen.getByText('Auswahl aufheben')).toBeTruthy();
  });

  it('calls saveContribCreate with the selected members on submit', async () => {
    const { app, formInitial } = makeApp({ label: 'Mitgliedsbeitrag', amount: '25' });
    render(<ContribCreateSheet app={app as never} sheet={{ formInitial } as never} />);
    fireEvent.click(screen.getByText('Anna Müller').closest('button')!);
    fireEvent.click(screen.getByRole('button', { name: /Beitrag anlegen/i }));
    await waitFor(() => {
      expect(app.saveContribCreate).toHaveBeenCalledWith(
        expect.objectContaining({ label: 'Mitgliedsbeitrag', amount: '25', userIds: ['u1'] }),
      );
    });
  });
});
