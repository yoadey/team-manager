import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { RolesSheet, RoleFormSheet } from './RoleSheets';

vi.mock('@/context/AppContext', () => {
  const useApp = vi.fn();
  return {
    useApp,
    // Actions + selector derive from the per-test useApp mock so migrated
    // atoms (TextInput/TextArea via useAppActions/useAppSelector) resolve.
    useAppActions: vi.fn(() => useApp()),
    useAppSelector: (sel: (s: { form: Record<string, unknown> }) => unknown) => sel(useApp().state),
  };
});

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

const MOCK_ROLES = [
  {
    id: 'r1',
    name: 'Trainer',
    color: '#1565C0',
    system: true,
    permissions: {
      events: 'write',
      members: 'read',
      finances: 'none',
      news: 'write',
      polls: 'read',
      settings: 'none',
    },
  },
  {
    id: 'r2',
    name: 'Kassierer',
    color: '#B71C1C',
    system: false,
    permissions: {
      events: 'read',
      members: 'none',
      finances: 'write',
      news: 'none',
      polls: 'none',
      settings: 'none',
    },
  },
];

function makeApp(overrides: Record<string, unknown> = {}) {
  const app = {
    state: {
      primaryColor: '#1565C0',
      roles: MOCK_ROLES,
      form: {},
      busy: null,
      ...overrides,
    },
    can: vi.fn().mockReturnValue(true),
    activeTeam: vi.fn().mockReturnValue({ id: 't1', name: 'Testteam' }),
    openCreateRole: vi.fn(),
    setRolePerm: vi.fn(),
    saveRole: vi.fn(),
    onFormInput: vi.fn(),
  };
  mockUseApp.mockReturnValue(app as unknown as ReturnType<typeof useApp>);
  return app;
}

const SHEET = { type: 'roles' } as never;

// ─── RolesSheet ───────────────────────────────────────────────────────────────

describe('RolesSheet', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders role names for all roles in state', () => {
    const app = makeApp();
    render(<RolesSheet app={app as never} sheet={SHEET} />);
    expect(screen.getByText('Trainer')).toBeTruthy();
    expect(screen.getByText('Kassierer')).toBeTruthy();
  });

  it('renders nothing for roles list when roles array is empty', () => {
    const app = makeApp({ roles: [] });
    render(<RolesSheet app={app as never} sheet={SHEET} />);
    expect(screen.queryByText('Trainer')).toBeNull();
    expect(screen.queryByText('Kassierer')).toBeNull();
  });

  it('shows "Standard" chip for system roles and "Eigen" for custom roles', () => {
    const app = makeApp();
    render(<RolesSheet app={app as never} sheet={SHEET} />);
    expect(screen.getByText('Standard')).toBeTruthy();
    expect(screen.getByText('Eigen')).toBeTruthy();
  });

  it('displays permission labels for each module', () => {
    const app = makeApp();
    render(<RolesSheet app={app as never} sheet={SHEET} />);
    // Module label "Termine" appears once per role (2 roles)
    const termineLabels = screen.getAllByText(/Termine/);
    expect(termineLabels.length).toBeGreaterThanOrEqual(2);
  });

  it('shows write, read, and none perm level labels', () => {
    const app = makeApp();
    render(<RolesSheet app={app as never} sheet={SHEET} />);
    // permWrite = 'Schreiben', permRead = 'Lesen', permNone = '—'
    const writeLabels = screen.getAllByText('Schreiben');
    expect(writeLabels.length).toBeGreaterThan(0);
    const readLabels = screen.getAllByText('Lesen');
    expect(readLabels.length).toBeGreaterThan(0);
    const noneLabels = screen.getAllByText('—');
    expect(noneLabels.length).toBeGreaterThan(0);
  });

  it('renders "Add role" button when user has settings:write permission', () => {
    const app = makeApp();
    app.can.mockReturnValue(true);
    render(<RolesSheet app={app as never} sheet={SHEET} />);
    expect(screen.getByText('Eigene Rolle definieren')).toBeTruthy();
  });

  it('does NOT render "Add role" button when user lacks settings:write permission', () => {
    const app = makeApp();
    app.can.mockReturnValue(false);
    render(<RolesSheet app={app as never} sheet={SHEET} />);
    expect(screen.queryByText('Eigene Rolle definieren')).toBeNull();
  });

  it('clicking "Add role" button calls openCreateRole', () => {
    const app = makeApp();
    app.can.mockReturnValue(true);
    render(<RolesSheet app={app as never} sheet={SHEET} />);
    fireEvent.click(screen.getByText('Eigene Rolle definieren'));
    expect(app.openCreateRole).toHaveBeenCalledTimes(1);
  });

  it('renders all 5 module rows for each role (e.g. Finanzen appears twice)', () => {
    const app = makeApp();
    render(<RolesSheet app={app as never} sheet={SHEET} />);
    const finLabel = screen.getAllByText(/Finanzen/);
    expect(finLabel.length).toBeGreaterThanOrEqual(2);
  });

  it('renders with a single role correctly', () => {
    const app = makeApp({ roles: [MOCK_ROLES[0]] });
    render(<RolesSheet app={app as never} sheet={SHEET} />);
    expect(screen.getByText('Trainer')).toBeTruthy();
    expect(screen.queryByText('Kassierer')).toBeNull();
  });

  it('shows module+permission text pairs for each role card', () => {
    const app = makeApp();
    render(<RolesSheet app={app as never} sheet={SHEET} />);
    // Termine appears in both role cards
    const newsLabels = screen.getAllByText(/Neuigkeiten/);
    expect(newsLabels.length).toBeGreaterThanOrEqual(2);
  });
});

// ─── RoleFormSheet ────────────────────────────────────────────────────────────

describe('RoleFormSheet', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  const FORM_SHEET = { type: 'roleForm' } as never;

  function makeFormApp(stateOverrides: Record<string, unknown> = {}) {
    const app = {
      state: {
        primaryColor: '#1565C0',
        roles: MOCK_ROLES,
        form: {
          name: '',
          perms: {
            events: 'none',
            members: 'none',
            finances: 'none',
            news: 'none',
            polls: 'none',
            settings: 'none',
          },
        },
        busy: null,
        ...stateOverrides,
      },
      can: vi.fn().mockReturnValue(true),
      activeTeam: vi.fn().mockReturnValue({ id: 't1', name: 'Testteam' }),
      setRolePerm: vi.fn(),
      saveRole: vi.fn(),
      onFormInput: vi.fn(),
    };
    mockUseApp.mockReturnValue(app as unknown as ReturnType<typeof useApp>);
    return app;
  }

  it('renders the role name input field with placeholder', () => {
    const app = makeFormApp();
    render(<RoleFormSheet app={app as never} sheet={FORM_SHEET} />);
    expect(screen.getByPlaceholderText('z. B. Social-Media-Team')).toBeTruthy();
  });

  it('renders the "Rechte je Modul" section label', () => {
    const app = makeFormApp();
    render(<RoleFormSheet app={app as never} sheet={FORM_SHEET} />);
    expect(screen.getByText('Rechte je Modul')).toBeTruthy();
  });

  it('renders module rows for all modules', () => {
    const app = makeFormApp();
    render(<RoleFormSheet app={app as never} sheet={FORM_SHEET} />);
    expect(screen.getByText('Termine')).toBeTruthy();
    expect(screen.getByText('Finanzen')).toBeTruthy();
    expect(screen.getByText('Mitglieder')).toBeTruthy();
  });

  it('renders —, Lesen, Schreiben permission buttons for each module', () => {
    const app = makeFormApp();
    render(<RoleFormSheet app={app as never} sheet={FORM_SHEET} />);
    const writeButtons = screen.getAllByText('Schreiben');
    const readButtons = screen.getAllByText('Lesen');
    const noneButtons = screen.getAllByText('—');
    // There are multiple modules, each with 3 buttons
    expect(writeButtons.length).toBeGreaterThan(1);
    expect(readButtons.length).toBeGreaterThan(1);
    expect(noneButtons.length).toBeGreaterThan(1);
  });

  it('clicking a "Schreiben" button calls setRolePerm with "write"', () => {
    const app = makeFormApp();
    render(<RoleFormSheet app={app as never} sheet={FORM_SHEET} />);
    const writeButtons = screen.getAllByText('Schreiben');
    fireEvent.click(writeButtons[0]);
    expect(app.setRolePerm).toHaveBeenCalledWith(expect.any(String), 'write');
  });

  it('clicking "Lesen" calls setRolePerm with "read"', () => {
    const app = makeFormApp();
    render(<RoleFormSheet app={app as never} sheet={FORM_SHEET} />);
    const readButtons = screen.getAllByText('Lesen');
    fireEvent.click(readButtons[0]);
    expect(app.setRolePerm).toHaveBeenCalledWith(expect.any(String), 'read');
  });

  it('clicking "—" (none) calls setRolePerm with "none"', () => {
    const app = makeFormApp();
    render(<RoleFormSheet app={app as never} sheet={FORM_SHEET} />);
    const noneButtons = screen.getAllByText('—');
    fireEvent.click(noneButtons[0]);
    expect(app.setRolePerm).toHaveBeenCalledWith(expect.any(String), 'none');
  });

  it('renders the save role button', () => {
    const app = makeFormApp();
    render(<RoleFormSheet app={app as never} sheet={FORM_SHEET} />);
    expect(screen.getByRole('button', { name: /Rolle speichern/i })).toBeTruthy();
  });

  it('clicking save button calls saveRole', () => {
    const app = makeFormApp();
    render(<RoleFormSheet app={app as never} sheet={FORM_SHEET} />);
    const saveBtn = screen.getByRole('button', { name: /Rolle speichern/i });
    fireEvent.click(saveBtn);
    expect(app.saveRole).toHaveBeenCalledTimes(1);
  });

  it('save button is disabled when busy is "save"', () => {
    const app = makeFormApp({ busy: 'save' });
    render(<RoleFormSheet app={app as never} sheet={FORM_SHEET} />);
    const saveBtn = screen.getByRole('button', { name: /Rolle speichern/i });
    expect(saveBtn).toBeDisabled();
  });

  it('renders the role name field label', () => {
    const app = makeFormApp();
    render(<RoleFormSheet app={app as never} sheet={FORM_SHEET} />);
    expect(screen.getByText('Rollenname')).toBeTruthy();
  });
});
