import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { TeamsSheet, MoreSheet } from './NavSheets';

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

function makeApp(overrides: Record<string, unknown> = {}) {
  const app = {
    state: {
      primaryColor: '#1565C0',
      teams: MOCK_TEAMS,
      activeTeamId: 'team-1',
      roles: MOCK_ROLES,
      busy: null,
      ...overrides,
    },
    can: vi.fn().mockReturnValue(true),
    activeTeam: vi.fn().mockReturnValue(MOCK_TEAMS[0]),
    myRoles: vi.fn().mockReturnValue([MOCK_ROLES[0]]),
    selectTeam: vi.fn(),
    openCreateTeam: vi.fn(),
    toastMsg: vi.fn(),
    setState: vi.fn(),
    go: vi.fn(),
  };
  mockUseApp.mockReturnValue(app as unknown as ReturnType<typeof useApp>);
  return app;
}

const SHEET = { type: 'teams' } as never;

// ─── TeamsSheet ───────────────────────────────────────────────────────────────

describe('TeamsSheet', () => {
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

  // Regression test: the team-switcher subtitle used to unconditionally
  // join role names and member count with ' · ', so a team where the
  // caller holds no role (e.g. their sole role assignment was deleted)
  // rendered a dangling leading separator like " · 8 Mitglieder".
  it('omits the separator for a team where the caller has no roles', () => {
    const app = makeApp({
      teams: [...MOCK_TEAMS, { ...MOCK_TEAMS[1], id: 'team-3', name: 'Roleless Club', myRoles: [] }],
    });
    render(<TeamsSheet app={app as never} sheet={SHEET} />);
    expect(screen.getByText('8 Mitglieder')).toBeTruthy();
    expect(screen.queryByText(/^·/)).toBeNull();
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

// ─── MoreSheet ────────────────────────────────────────────────────────────────

describe('MoreSheet', () => {
  it('renders navigation items: stats, news, polls, team, settings', () => {
    const app = makeApp();
    render(<MoreSheet app={app as never} sheet={SHEET} />);
    expect(screen.getByText('Statistik')).toBeTruthy();
    expect(screen.getByText('Neuigkeiten')).toBeTruthy();
    expect(screen.getByText('Umfragen')).toBeTruthy();
    expect(screen.getByText('Team')).toBeTruthy();
    expect(screen.getByText('Einstellungen')).toBeTruthy();
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

  // Regression test: previously only 'finances' called app.can() here --
  // stats/news/polls/team were hardcoded `true`, so a role with e.g.
  // news:none still saw and could tap a "Neuigkeiten" entry that bounced it
  // straight back to Home with a spurious forbidden toast.
  it('hides News when the caller lacks news:read', () => {
    const app = makeApp();
    app.can.mockImplementation((module: string) => module !== 'news');
    render(<MoreSheet app={app as never} sheet={SHEET} />);
    expect(screen.queryByText('Neuigkeiten')).toBeNull();
    expect(screen.getByText('Statistik')).toBeTruthy();
  });

  it('hides Umfragen when the caller lacks polls:read', () => {
    const app = makeApp();
    app.can.mockImplementation((module: string) => module !== 'polls');
    render(<MoreSheet app={app as never} sheet={SHEET} />);
    expect(screen.queryByText('Umfragen')).toBeNull();
  });

  it('hides Team when the caller lacks members:read', () => {
    const app = makeApp();
    app.can.mockImplementation((module: string) => module !== 'members');
    render(<MoreSheet app={app as never} sheet={SHEET} />);
    expect(screen.queryByText('Team')).toBeNull();
  });

  it('hides Statistik when the caller lacks stats:read', () => {
    const app = makeApp();
    app.can.mockImplementation((module: string) => module !== 'stats');
    render(<MoreSheet app={app as never} sheet={SHEET} />);
    expect(screen.queryByText('Statistik')).toBeNull();
  });

  // Regression test: stats used to piggyback on the events module (see
  // CLAUDE.md history) -- it must no longer be tied to it, so a team can
  // show event details to everyone while restricting attendance statistics.
  it('keeps showing Statistik when the caller lacks events:read but has stats:read', () => {
    const app = makeApp();
    app.can.mockImplementation((module: string) => module !== 'events');
    render(<MoreSheet app={app as never} sheet={SHEET} />);
    expect(screen.getByText('Statistik')).toBeTruthy();
  });

  // Settings is ungated (ROUTE_MODULE.settings is null), so it must stay
  // visible regardless of which module permission the caller lacks.
  it('always shows Einstellungen regardless of module permissions', () => {
    const app = makeApp();
    app.can.mockReturnValue(false);
    render(<MoreSheet app={app as never} sheet={SHEET} />);
    expect(screen.getByText('Einstellungen')).toBeTruthy();
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

  it('clicking settings navigates to settings route', () => {
    const app = makeApp();
    render(<MoreSheet app={app as never} sheet={SHEET} />);
    fireEvent.click(screen.getByText('Einstellungen'));
    expect(app.go).toHaveBeenCalledWith('settings');
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
    // Each item shows a chevron_right SVG icon
    const chevrons = screen.getAllByTestId('ChevronRightOutlinedIcon');
    expect(chevrons.length).toBeGreaterThan(0);
  });
});
