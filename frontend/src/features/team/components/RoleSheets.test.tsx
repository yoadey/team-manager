import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
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
    openRoleForm: vi.fn(),
    saveRole: vi.fn(),
    removeRole: vi.fn(),
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

  it('clicking "Add role" button calls openRoleForm with no argument', () => {
    const app = makeApp();
    app.can.mockReturnValue(true);
    render(<RolesSheet app={app as never} sheet={SHEET} />);
    fireEvent.click(screen.getByText('Eigene Rolle definieren'));
    expect(app.openRoleForm).toHaveBeenCalledWith();
  });

  it('shows edit/delete actions for custom roles but not system roles when settings:write is held', () => {
    const app = makeApp();
    app.can.mockReturnValue(true);
    render(<RolesSheet app={app as never} sheet={SHEET} />);
    expect(screen.getAllByLabelText('Rolle bearbeiten')).toHaveLength(1);
    expect(screen.getAllByLabelText('Rolle löschen')).toHaveLength(1);
  });

  it('hides edit/delete actions entirely without settings:write, even for custom roles', () => {
    const app = makeApp();
    app.can.mockReturnValue(false);
    render(<RolesSheet app={app as never} sheet={SHEET} />);
    expect(screen.queryByLabelText('Rolle bearbeiten')).toBeNull();
    expect(screen.queryByLabelText('Rolle löschen')).toBeNull();
  });

  it('clicking the edit action opens the role form pre-filled with that role', () => {
    const app = makeApp();
    app.can.mockReturnValue(true);
    render(<RolesSheet app={app as never} sheet={SHEET} />);
    fireEvent.click(screen.getByLabelText('Rolle bearbeiten'));
    expect(app.openRoleForm).toHaveBeenCalledWith(MOCK_ROLES[1]);
  });

  it('clicking the delete action calls removeRole with the role id', () => {
    const app = makeApp();
    app.can.mockReturnValue(true);
    render(<RolesSheet app={app as never} sheet={SHEET} />);
    fireEvent.click(screen.getByLabelText('Rolle löschen'));
    expect(app.removeRole).toHaveBeenCalledWith('r2');
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

  function makeFormApp(stateOverrides: Record<string, unknown> = {}) {
    const app = {
      state: {
        primaryColor: '#1565C0',
        roles: MOCK_ROLES,
        busy: null,
        ...stateOverrides,
      },
      can: vi.fn().mockReturnValue(true),
      activeTeam: vi.fn().mockReturnValue({ id: 't1', name: 'Testteam' }),
      saveRole: vi.fn(),
    };
    mockUseApp.mockReturnValue(app as unknown as ReturnType<typeof useApp>);
    return app;
  }

  function makeFormSheet(formOverrides: Record<string, unknown> = {}) {
    return {
      type: 'roleForm',
      formInitial: {
        name: '',
        perms: { events: 'none', members: 'none', finances: 'none', news: 'none', polls: 'none', settings: 'none' },
        ...formOverrides,
      },
    } as never;
  }

  it('renders the role name input field with placeholder', () => {
    const app = makeFormApp();
    render(<RoleFormSheet app={app as never} sheet={makeFormSheet()} />);
    expect(screen.getByPlaceholderText('z. B. Social-Media-Team')).toBeTruthy();
  });

  it('renders the "Rechte je Modul" section label', () => {
    const app = makeFormApp();
    render(<RoleFormSheet app={app as never} sheet={makeFormSheet()} />);
    expect(screen.getByText('Rechte je Modul')).toBeTruthy();
  });

  it('renders module rows for all modules', () => {
    const app = makeFormApp();
    render(<RoleFormSheet app={app as never} sheet={makeFormSheet()} />);
    expect(screen.getByText('Termine')).toBeTruthy();
    expect(screen.getByText('Finanzen')).toBeTruthy();
    expect(screen.getByText('Mitglieder')).toBeTruthy();
  });

  it('renders —, Lesen, Schreiben permission buttons for each module', () => {
    const app = makeFormApp();
    render(<RoleFormSheet app={app as never} sheet={makeFormSheet()} />);
    const writeButtons = screen.getAllByText('Schreiben');
    const readButtons = screen.getAllByText('Lesen');
    const noneButtons = screen.getAllByText('—');
    // There are multiple modules, each with 3 buttons
    expect(writeButtons.length).toBeGreaterThan(1);
    expect(readButtons.length).toBeGreaterThan(1);
    expect(noneButtons.length).toBeGreaterThan(1);
  });

  it('clicking a "Schreiben" button marks it pressed', () => {
    const app = makeFormApp();
    render(<RoleFormSheet app={app as never} sheet={makeFormSheet()} />);
    const writeButton = screen.getAllByText('Schreiben')[0].closest('button')!;
    fireEvent.click(writeButton);
    expect(writeButton).toHaveAttribute('aria-pressed', 'true');
  });

  it('clicking "Lesen" marks it pressed', () => {
    const app = makeFormApp();
    render(<RoleFormSheet app={app as never} sheet={makeFormSheet()} />);
    const readButton = screen.getAllByText('Lesen')[0].closest('button')!;
    fireEvent.click(readButton);
    expect(readButton).toHaveAttribute('aria-pressed', 'true');
  });

  it('clicking "—" (none) marks it pressed', () => {
    const app = makeFormApp();
    const sheet = makeFormSheet({
      name: 'Trainer',
      perms: { events: 'write', members: 'none', finances: 'none', news: 'none', polls: 'none', settings: 'none' },
    });
    render(<RoleFormSheet app={app as never} sheet={sheet} />);
    const noneButton = screen.getAllByText('—')[0].closest('button')!;
    fireEvent.click(noneButton);
    expect(noneButton).toHaveAttribute('aria-pressed', 'true');
  });

  it('saving after changing a permission submits the updated value', async () => {
    const app = makeFormApp();
    const sheet = makeFormSheet({ name: 'Trainer' });
    render(<RoleFormSheet app={app as never} sheet={sheet} />);
    const writeButton = screen.getAllByText('Schreiben')[0].closest('button')!;
    fireEvent.click(writeButton);
    const saveBtn = screen.getByRole('button', { name: /Rolle speichern/i });
    fireEvent.click(saveBtn);
    await waitFor(() => {
      expect(app.saveRole).toHaveBeenCalledWith(
        expect.objectContaining({ perms: expect.objectContaining({ events: 'write' }) }),
      );
    });
  });

  it('renders the save role button', () => {
    const app = makeFormApp();
    render(<RoleFormSheet app={app as never} sheet={makeFormSheet()} />);
    expect(screen.getByRole('button', { name: /Rolle speichern/i })).toBeTruthy();
  });

  it('clicking save button calls saveRole', async () => {
    const app = makeFormApp();
    const sheet = makeFormSheet({ name: 'Trainer' });
    render(<RoleFormSheet app={app as never} sheet={sheet} />);
    const saveBtn = screen.getByRole('button', { name: /Rolle speichern/i });
    fireEvent.click(saveBtn);
    await waitFor(() => {
      expect(app.saveRole).toHaveBeenCalledTimes(1);
    });
  });

  it('disables the save button when the name is empty', () => {
    const app = makeFormApp();
    render(<RoleFormSheet app={app as never} sheet={makeFormSheet()} />);
    const saveBtn = screen.getByRole('button', { name: /Rolle speichern/i });
    fireEvent.click(saveBtn);
    expect(app.saveRole).not.toHaveBeenCalled();
  });

  it('save button is disabled while saveRole is pending', async () => {
    const app = makeFormApp();
    let resolveSave!: () => void;
    app.saveRole = vi.fn(() => new Promise<void>((resolve) => (resolveSave = resolve)));
    const sheet = makeFormSheet({ name: 'Trainer' });
    render(<RoleFormSheet app={app as never} sheet={sheet} />);
    const saveBtn = screen.getByRole('button', { name: /Rolle speichern/i });
    fireEvent.click(saveBtn);
    await waitFor(() => expect(saveBtn).toBeDisabled());
    resolveSave();
  });

  it('renders the role name field label', () => {
    const app = makeFormApp();
    render(<RoleFormSheet app={app as never} sheet={makeFormSheet()} />);
    expect(screen.getByText('Rollenname')).toBeTruthy();
  });
});
