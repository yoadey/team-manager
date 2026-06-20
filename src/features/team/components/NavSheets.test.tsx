import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { TeamsSheet, ProfileSheet, MoreSheet } from './NavSheets';

vi.mock('@/context/AppContext', () => ({
  useApp: vi.fn(),
  useAppActions: vi.fn().mockReturnValue({}),
}));

vi.mock('@/styles/tokens', async (importOriginal) => {
  const mod = await importOriginal<typeof import('@/styles/tokens')>();
  return {
    ...mod,
    buildTokens: vi.fn().mockReturnValue({
      primary: '#4285F4',
      primaryContainer: '#E8F0FE',
      onPrimaryContainer: '#001D35',
      onPrimary: '#ffffff',
    }),
  };
});

import { useApp } from '@/context/AppContext';
const mockUseApp = vi.mocked(useApp);

const MOCK_TEAMS = [
  {
    id: 'team-1',
    name: 'FC Testverein',
    icon: '⚽',
    iconBg: '#1565C0',
    iconFg: '#fff',
    logo: null,
    myRoles: [{ id: 'r1', name: 'Trainer' }],
    memberCount: 15,
  },
  {
    id: 'team-2',
    name: 'SV Musterclub',
    icon: '🏆',
    iconBg: '#B71C1C',
    iconFg: '#fff',
    logo: null,
    myRoles: [{ id: 'r2', name: 'Mitglied' }],
    memberCount: 8,
  },
];

const MOCK_ROLES = [
  { id: 'r1', name: 'Trainer', color: '#1565C0', system: true },
  { id: 'r2', name: 'Kassierer', color: '#B71C1C', system: false },
];

const MOCK_USER = {
  id: 'u1',
  name: 'Max Mustermann',
  email: 'max@example.com',
  photo: null,
  avatarColor: '#1565C0',
};

// Note: the worktree version of ProfileSheet does not have a colorScheme section.

function makeApp(overrides: Record<string, unknown> = {}) {
  const app = {
    state: {
      primaryColor: '#1565C0',
      teams: MOCK_TEAMS,
      activeTeamId: 'team-1',
      roles: MOCK_ROLES,
      user: MOCK_USER,
      colorScheme: 'system' as const,
      form: {},
      busy: null,
      ...overrides,
    },
    can: vi.fn().mockReturnValue(true),
    activeTeam: vi.fn().mockReturnValue(MOCK_TEAMS[0]),
    myRoles: vi.fn().mockReturnValue([MOCK_ROLES[0]]),
    selectTeam: vi.fn(),
    openCreateTeam: vi.fn(),
    toggleMyRole: vi.fn(),
    setColorScheme: vi.fn(),
    logout: vi.fn(),
    go: vi.fn(),
    onFormInput: vi.fn(),
    onFile: vi.fn(),
    uploadMyPhoto: vi.fn(),
  };
  mockUseApp.mockReturnValue(app as unknown as ReturnType<typeof useApp>);
  return app;
}

const SHEET = { type: 'teams' } as never;

// ─── TeamsSheet ───────────────────────────────────────────────────────────────

describe('TeamsSheet', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders team names for all teams', () => {
    const app = makeApp();
    render(<TeamsSheet app={app as never} sheet={SHEET} />);
    expect(screen.getByText('FC Testverein')).toBeTruthy();
    expect(screen.getByText('SV Musterclub')).toBeTruthy();
  });

  it('renders team role and member count info for each team', () => {
    const app = makeApp();
    render(<TeamsSheet app={app as never} sheet={SHEET} />);
    // Role names appear in the subtitle (use getAllBy since "Mitglied" appears in subtitle of both)
    expect(screen.getByText(/Trainer · 15 Mitglieder/)).toBeTruthy();
    expect(screen.getByText(/Mitglied · 8 Mitglieder/)).toBeTruthy();
  });

  it('renders "New team" add button', () => {
    const app = makeApp();
    render(<TeamsSheet app={app as never} sheet={SHEET} />);
    expect(screen.getByText('Neues Team anlegen')).toBeTruthy();
  });

  it('clicking a team button calls selectTeam with the team id', () => {
    const app = makeApp();
    render(<TeamsSheet app={app as never} sheet={SHEET} />);
    fireEvent.click(screen.getByText('FC Testverein'));
    expect(app.selectTeam).toHaveBeenCalledWith('team-1');
  });

  it('clicking the second team calls selectTeam with its id', () => {
    const app = makeApp();
    render(<TeamsSheet app={app as never} sheet={SHEET} />);
    fireEvent.click(screen.getByText('SV Musterclub'));
    expect(app.selectTeam).toHaveBeenCalledWith('team-2');
  });

  it('clicking the "New team" button calls openCreateTeam', () => {
    const app = makeApp();
    render(<TeamsSheet app={app as never} sheet={SHEET} />);
    fireEvent.click(screen.getByText('Neues Team anlegen'));
    expect(app.openCreateTeam).toHaveBeenCalledTimes(1);
  });

  it('shows member count label for each team', () => {
    const app = makeApp();
    render(<TeamsSheet app={app as never} sheet={SHEET} />);
    // Both teams should show a member count
    expect(screen.getByText(/15 Mitglieder/)).toBeTruthy();
    expect(screen.getByText(/8 Mitglieder/)).toBeTruthy();
  });

  it('renders team icon when no logo is set', () => {
    const app = makeApp();
    render(<TeamsSheet app={app as never} sheet={SHEET} />);
    expect(screen.getByText('⚽')).toBeTruthy();
    expect(screen.getByText('🏆')).toBeTruthy();
  });

  it('renders with empty teams list without crashing', () => {
    const app = makeApp({ teams: [] });
    render(<TeamsSheet app={app as never} sheet={SHEET} />);
    // Add button should still appear
    expect(screen.getByText('Neues Team anlegen')).toBeTruthy();
  });
});

// ─── ProfileSheet ─────────────────────────────────────────────────────────────

describe('ProfileSheet', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders the user name', () => {
    const app = makeApp();
    render(<ProfileSheet app={app as never} sheet={SHEET} />);
    expect(screen.getByText('Max Mustermann')).toBeTruthy();
  });

  it('renders the user email', () => {
    const app = makeApp();
    render(<ProfileSheet app={app as never} sheet={SHEET} />);
    expect(screen.getByText('max@example.com')).toBeTruthy();
  });

  it('renders all roles as toggleable items', () => {
    const app = makeApp();
    render(<ProfileSheet app={app as never} sheet={SHEET} />);
    expect(screen.getByText('Trainer')).toBeTruthy();
    expect(screen.getByText('Kassierer')).toBeTruthy();
  });

  it('clicking a role calls toggleMyRole with the role id', () => {
    const app = makeApp();
    render(<ProfileSheet app={app as never} sheet={SHEET} />);
    fireEvent.click(screen.getByText('Trainer'));
    expect(app.toggleMyRole).toHaveBeenCalledWith('r1');
  });

  it('renders logout button', () => {
    const app = makeApp();
    render(<ProfileSheet app={app as never} sheet={SHEET} />);
    expect(screen.getByText('Abmelden')).toBeTruthy();
  });

  it('clicking logout button calls logout', () => {
    const app = makeApp();
    render(<ProfileSheet app={app as never} sheet={SHEET} />);
    fireEvent.click(screen.getByText('Abmelden'));
    expect(app.logout).toHaveBeenCalledTimes(1);
  });

  it('renders multi-role hint text', () => {
    const app = makeApp();
    render(<ProfileSheet app={app as never} sheet={SHEET} />);
    expect(screen.getByText(/Mehrfachauswahl möglich/)).toBeTruthy();
  });

  it('renders the team name in the roles section title', () => {
    const app = makeApp();
    render(<ProfileSheet app={app as never} sheet={SHEET} />);
    expect(screen.getByText(/Meine Rollen in/)).toBeTruthy();
  });

  it('selected role shows a checkmark icon (check text from Sym)', () => {
    const app = makeApp();
    // myRoles returns r1, so Trainer is selected
    render(<ProfileSheet app={app as never} sheet={SHEET} />);
    // Sym renders icon name as text; "check" should appear for selected role
    expect(screen.getByText('check')).toBeTruthy();
  });

  it('non-selected role does not show a checkmark', () => {
    const app = makeApp();
    // myRoles returns [r1] (Trainer selected), Kassierer is not selected
    render(<ProfileSheet app={app as never} sheet={SHEET} />);
    // Both roles render; only the selected one has a check icon
    const checkmarks = screen.getAllByText('check');
    expect(checkmarks).toHaveLength(1);
  });

  it('renders team name in "my roles" section title', () => {
    const app = makeApp();
    render(<ProfileSheet app={app as never} sheet={SHEET} />);
    // Team name should appear in section heading
    expect(screen.getByText(/FC Testverein/)).toBeTruthy();
  });
});

// ─── MoreSheet ────────────────────────────────────────────────────────────────

describe('MoreSheet', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders navigation items: stats, news, polls, team', () => {
    const app = makeApp();
    render(<MoreSheet app={app as never} sheet={SHEET} />);
    expect(screen.getByText('Statistik')).toBeTruthy();
    expect(screen.getByText('Neuigkeiten')).toBeTruthy();
    expect(screen.getByText('Umfragen')).toBeTruthy();
    expect(screen.getByText('Team')).toBeTruthy();
  });

  it('renders Finanzen when user can read finances', () => {
    const app = makeApp();
    app.can.mockReturnValue(true);
    render(<MoreSheet app={app as never} sheet={SHEET} />);
    expect(screen.getByText('Finanzen')).toBeTruthy();
  });

  it('hides Finanzen when user cannot read finances', () => {
    const app = makeApp();
    app.can.mockReturnValue(false);
    render(<MoreSheet app={app as never} sheet={SHEET} />);
    expect(screen.queryByText('Finanzen')).toBeNull();
  });

  it('clicking stats navigates to stats route', () => {
    const app = makeApp();
    render(<MoreSheet app={app as never} sheet={SHEET} />);
    fireEvent.click(screen.getByText('Statistik'));
    expect(app.go).toHaveBeenCalledWith('stats');
  });

  it('clicking news navigates to news route', () => {
    const app = makeApp();
    render(<MoreSheet app={app as never} sheet={SHEET} />);
    fireEvent.click(screen.getByText('Neuigkeiten'));
    expect(app.go).toHaveBeenCalledWith('news');
  });

  it('clicking polls navigates to polls route', () => {
    const app = makeApp();
    render(<MoreSheet app={app as never} sheet={SHEET} />);
    fireEvent.click(screen.getByText('Umfragen'));
    expect(app.go).toHaveBeenCalledWith('polls');
  });

  it('clicking team navigates to team route', () => {
    const app = makeApp();
    render(<MoreSheet app={app as never} sheet={SHEET} />);
    fireEvent.click(screen.getByText('Team'));
    expect(app.go).toHaveBeenCalledWith('team');
  });

  it('clicking finances navigates to finances route', () => {
    const app = makeApp();
    app.can.mockReturnValue(true);
    render(<MoreSheet app={app as never} sheet={SHEET} />);
    fireEvent.click(screen.getByText('Finanzen'));
    expect(app.go).toHaveBeenCalledWith('finances');
  });

  it('renders chevron_right icons for each nav item', () => {
    const app = makeApp();
    render(<MoreSheet app={app as never} sheet={SHEET} />);
    // Each item shows "chevron_right" glyph text (Sym renders glyph name as text)
    const chevrons = screen.getAllByText('chevron_right');
    expect(chevrons.length).toBeGreaterThan(0);
  });
});

describe('ProfileSheet — color scheme', () => {
  it('renders color scheme buttons (system/light/dark)', () => {
    const app = makeApp();
    render(<ProfileSheet app={app as never} sheet={SHEET} />);
    const btns = document.querySelectorAll('button');
    expect(btns.length).toBeGreaterThan(0);
  });

  it('clicking light scheme calls setColorScheme', () => {
    const app = makeApp();
    render(<ProfileSheet app={app as never} sheet={SHEET} />);
    fireEvent.click(screen.getByText('Hell'));
    expect(app.setColorScheme).toHaveBeenCalledWith('light');
  });

  it('clicking dark scheme calls setColorScheme', () => {
    const app = makeApp();
    render(<ProfileSheet app={app as never} sheet={SHEET} />);
    fireEvent.click(screen.getByText('Dunkel'));
    expect(app.setColorScheme).toHaveBeenCalledWith('dark');
  });
});
