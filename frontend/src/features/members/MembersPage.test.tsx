import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MembersPage } from './MembersPage';

vi.mock('@/context/AppContext', () => ({
  useApp: vi.fn(),
  useAppActions: vi.fn().mockReturnValue({}),
}));

import { useApp } from '@/context/AppContext';
const mockUseApp = useApp as ReturnType<typeof vi.fn>;

function makeApp(overrides: Record<string, unknown> = {}) {
  return {
    state: {
      primaryColor: '#4285F4',
      members: [],
      user: { id: 'u1', name: 'Test User' },
      ...overrides,
    },
    can: vi.fn().mockReturnValue(false),
    isStaff: vi.fn().mockReturnValue(false),
    openMemberDetail: vi.fn(),
    openMemberForm: vi.fn(),
    openRoles: vi.fn(),
  };
}

function makeMember(overrides: Record<string, unknown> = {}) {
  return {
    membershipId: 'ms1',
    userId: 'u2',
    name: 'Anna Müller',
    email: 'anna@test.com',
    avatarColor: '#4285F4',
    photo: null,
    roles: [{ id: 'r1', name: 'Mitglied' }],
    primaryRole: null,
    joinedAt: '2025-01-01',
    ...overrides,
  };
}

describe('MembersPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders empty list when no members', () => {
    mockUseApp.mockReturnValue(makeApp({ members: [] }));
    render(<MembersPage />);
    expect(screen.getByText('0 Mitglieder')).toBeTruthy();
  });

  it('renders member list', () => {
    mockUseApp.mockReturnValue(makeApp({ members: [makeMember()] }));
    render(<MembersPage />);
    expect(screen.getByText('Anna Müller')).toBeTruthy();
  });

  it('marks current user with "Du" chip', () => {
    mockUseApp.mockReturnValue(
      makeApp({
        members: [makeMember({ membershipId: 'ms1', userId: 'u1', name: 'Test User' })],
        user: { id: 'u1', name: 'Test User' },
      }),
    );
    render(<MembersPage />);
    expect(screen.getByText('Du')).toBeTruthy();
  });

  it('renders multiple members', () => {
    mockUseApp.mockReturnValue(
      makeApp({
        members: [makeMember(), makeMember({ membershipId: 'ms2', userId: 'u3', name: 'Bob Schmidt' })],
      }),
    );
    render(<MembersPage />);
    expect(screen.getByText('Anna Müller')).toBeTruthy();
    expect(screen.getByText('Bob Schmidt')).toBeTruthy();
  });

  it('renders Rollen & Rechte button', () => {
    mockUseApp.mockReturnValue(makeApp());
    render(<MembersPage />);
    expect(screen.getByText('Rollen & Rechte')).toBeTruthy();
  });

  it('renders member with primary role chip', () => {
    mockUseApp.mockReturnValue(
      makeApp({
        members: [makeMember({ primaryRole: { id: 'r1', name: 'Kapitän', color: '#E91E63' } })],
      }),
    );
    render(<MembersPage />);
    expect(screen.getByText('Kapitän')).toBeTruthy();
  });

  it('calls openMemberDetail when clicking a member row', async () => {
    const app = makeApp({ members: [makeMember()] });
    mockUseApp.mockReturnValue(app);
    render(<MembersPage />);
    await userEvent.click(screen.getByText('Anna Müller').closest('button')!);
    expect(app.openMemberDetail).toHaveBeenCalledWith('ms1');
  });

  it('calls openRoles when clicking Rollen & Rechte button', async () => {
    const app = makeApp();
    mockUseApp.mockReturnValue(app);
    render(<MembersPage />);
    await userEvent.click(screen.getByText('Rollen & Rechte').closest('button')!);
    expect(app.openRoles).toHaveBeenCalled();
  });

  it('filters members by search query', async () => {
    const members = [
      makeMember({ name: 'Anna Müller', roles: [{ id: 'r1', name: 'Kapitän' }] }),
      makeMember({ membershipId: 'ms2', userId: 'u3', name: 'Bob Schmidt', roles: [{ id: 'r1', name: 'Mitglied' }] }),
    ];
    mockUseApp.mockReturnValue(makeApp({ members }));
    render(<MembersPage />);
    const searchInput = document.querySelector('input[type="search"]')!;
    await userEvent.type(searchInput, 'Anna');
    expect(screen.getByText('Anna Müller')).toBeTruthy();
    expect(screen.queryByText('Bob Schmidt')).toBeNull();
  });

  it('filters members by role name', async () => {
    const members = [
      makeMember({ name: 'Anna Müller', roles: [{ id: 'r1', name: 'Kapitän' }] }),
      makeMember({ membershipId: 'ms2', userId: 'u3', name: 'Bob Schmidt', roles: [{ id: 'r2', name: 'Torwart' }] }),
    ];
    mockUseApp.mockReturnValue(makeApp({ members }));
    render(<MembersPage />);
    const searchInput = document.querySelector('input[type="search"]')!;
    await userEvent.type(searchInput, 'Kapitän');
    expect(screen.getByText('Anna Müller')).toBeTruthy();
    expect(screen.queryByText('Bob Schmidt')).toBeNull();
  });

  // Regression test: a search query matching zero members used to render a
  // silent blank Box with no feedback -- every other list view in the app
  // (PollsPage, NewsPage, EventAbsences, ...) shows an EmptyState for the
  // zero-item case, but this was the one gap.
  it('shows an empty state when the search query matches no members', async () => {
    const members = [makeMember({ name: 'Anna Müller' })];
    mockUseApp.mockReturnValue(makeApp({ members }));
    render(<MembersPage />);
    const searchInput = document.querySelector('input[type="search"]')!;
    await userEvent.type(searchInput, 'Zzzznomatch');
    expect(screen.queryByText('Anna Müller')).toBeNull();
    expect(screen.getByText('Kein Mitglied gefunden.')).toBeTruthy();
  });

  // Regression test: the header count used to always read
  // state.members.length (the unfiltered total), so searching "Anna" and
  // seeing one matching row still showed the full team count -- the count
  // and the visible rows disagreed.
  it('updates the header count to reflect the active search filter', async () => {
    const members = [
      makeMember({ name: 'Anna Müller' }),
      makeMember({ membershipId: 'ms2', userId: 'u3', name: 'Bob Schmidt' }),
    ];
    mockUseApp.mockReturnValue(makeApp({ members }));
    render(<MembersPage />);
    expect(screen.getByText('2 Mitglieder')).toBeTruthy();
    const searchInput = document.querySelector('input[type="search"]')!;
    await userEvent.type(searchInput, 'Anna');
    expect(screen.getByText('1 Mitglied')).toBeTruthy();
    expect(screen.queryByText('2 Mitglieder')).toBeNull();
  });
});
