import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { TeamPage } from './TeamPage';

vi.mock('@/context/AppContext', () => ({
  useApp: vi.fn(),
  useAppActions: vi.fn().mockReturnValue({}),
}));

import { useApp } from '@/context/AppContext';
const mockUseApp = useApp as ReturnType<typeof vi.fn>;

function makeTeam(overrides: Record<string, unknown> = {}) {
  return {
    id: 'team1',
    name: 'FC Test',
    icon: '⚽',
    iconBg: '#000',
    iconFg: '#fff',
    logo: null,
    photo: null,
    memberCount: 10,
    myRoles: [{ id: 'r1', name: 'Mitglied' }],
    ...overrides,
  };
}

function makeApp(overrides: Record<string, unknown> = {}) {
  return {
    state: {
      primaryColor: '#4285F4',
      teams: [makeTeam()],
      activeTeamId: 'team1',
      user: { id: 'u1', name: 'Test User' },
      ...overrides,
    },
    can: vi.fn().mockReturnValue(false),
    myRoles: vi.fn().mockReturnValue([{ id: 'r1', name: 'Mitglied' }]),
    activeTeam: vi.fn().mockReturnValue(makeTeam()),
    openInvite: vi.fn(),
    openTeamSettings: vi.fn(),
    openRoles: vi.fn(),
    go: vi.fn(),
    selectTeam: vi.fn(),
    openCreateTeam: vi.fn(),
  };
}

describe('TeamPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders team name in header', () => {
    mockUseApp.mockReturnValue(makeApp());
    render(<TeamPage />);
    expect(screen.getAllByText('FC Test').length).toBeGreaterThan(0);
  });

  it('renders Rollen & Rechte action', () => {
    mockUseApp.mockReturnValue(makeApp());
    render(<TeamPage />);
    expect(screen.getByText('Rollen & Rechte')).toBeTruthy();
  });

  it('renders Mitglieder action', () => {
    mockUseApp.mockReturnValue(makeApp());
    render(<TeamPage />);
    expect(screen.getByText('Mitglieder')).toBeTruthy();
  });

  it('shows team settings link for admin users', () => {
    const app = makeApp();
    app.can = vi.fn().mockReturnValue(true);
    mockUseApp.mockReturnValue(app);
    render(<TeamPage />);
    expect(screen.getByText('Team-Einstellungen')).toBeTruthy();
    expect(screen.getByText('Einladungslink erstellen')).toBeTruthy();
  });

  it('renders Meine Teams section with teams', () => {
    mockUseApp.mockReturnValue(makeApp());
    render(<TeamPage />);
    expect(screen.getByText('Meine Teams')).toBeTruthy();
  });

  it('renders create team button', () => {
    mockUseApp.mockReturnValue(makeApp());
    render(<TeamPage />);
    expect(screen.getByText('Neues Team anlegen')).toBeTruthy();
  });

  it('renders team with photo background', () => {
    const app = makeApp();
    app.activeTeam = vi.fn().mockReturnValue(makeTeam({ photo: 'data:image/png;base64,abc' }));
    mockUseApp.mockReturnValue(app);
    render(<TeamPage />);
    expect(screen.getAllByText('FC Test').length).toBeGreaterThan(0);
  });

  it('renders multiple teams in team list', () => {
    const app = makeApp({
      teams: [makeTeam(), makeTeam({ id: 'team2', name: 'TSV München' })],
    });
    mockUseApp.mockReturnValue(app);
    render(<TeamPage />);
    expect(screen.getAllByText('FC Test').length).toBeGreaterThan(0);
    expect(screen.getByText('TSV München')).toBeTruthy();
  });

  it('calls openRoles when clicking Rollen & Rechte', async () => {
    const app = makeApp();
    mockUseApp.mockReturnValue(app);
    render(<TeamPage />);
    await userEvent.click(screen.getByText('Rollen & Rechte').closest('button')!);
    expect(app.openRoles).toHaveBeenCalled();
  });

  it('calls go("members") when clicking Mitglieder', async () => {
    const app = makeApp();
    mockUseApp.mockReturnValue(app);
    render(<TeamPage />);
    await userEvent.click(screen.getByText('Mitglieder').closest('button')!);
    expect(app.go).toHaveBeenCalledWith('members');
  });

  it('calls openCreateTeam when clicking Neues Team anlegen', async () => {
    const app = makeApp();
    mockUseApp.mockReturnValue(app);
    render(<TeamPage />);
    await userEvent.click(screen.getByText('Neues Team anlegen').closest('button')!);
    expect(app.openCreateTeam).toHaveBeenCalled();
  });

  it('calls selectTeam when clicking a team row', async () => {
    const app = makeApp({
      teams: [makeTeam(), makeTeam({ id: 'team2', name: 'TSV München' })],
    });
    mockUseApp.mockReturnValue(app);
    render(<TeamPage />);
    await userEvent.click(screen.getByText('TSV München').closest('button')!);
    expect(app.selectTeam).toHaveBeenCalledWith('team2');
  });

  it('calls openInvite when clicking Einladungslink erstellen (admin)', async () => {
    const app = makeApp();
    app.can = vi.fn().mockReturnValue(true);
    mockUseApp.mockReturnValue(app);
    render(<TeamPage />);
    await userEvent.click(screen.getByText('Einladungslink erstellen').closest('button')!);
    expect(app.openInvite).toHaveBeenCalled();
  });
});
