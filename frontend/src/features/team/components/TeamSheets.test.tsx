import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor, act } from '@testing-library/react';
import { CreateTeamSheet, InviteSheet, TeamSettingsSheet } from './TeamSheets';

// ── mocks ─────────────────────────────────────────────────────────────────────

vi.mock('@/context/AppContext', () => {
  const useApp = vi.fn();
  return {
    useApp,
    useAppActions: vi.fn(() => useApp()),
    useAppSelector: (sel: (s: { form: Record<string, unknown> }) => unknown) => sel(useApp().state),
  };
});

vi.mock('@/layouts/useCompact', () => ({
  shortName: (name: string) => name.split(' ')[0],
}));

import { useApp } from '@/context/AppContext';
const mockUseApp = vi.mocked(useApp);

// Mock buildTokens to return predictable values; keep rest of tokens module real.
vi.mock('@/styles/tokens', async (importOriginal) => {
  const mod = await importOriginal<typeof import('@/styles/tokens')>();
  return {
    ...mod,
    buildTokens: vi.fn().mockReturnValue({
      primary: '#4285F4',
      primaryContainer: '#E8F0FE',
      onPrimary: '#ffffff',
      onPrimaryContainer: '#001D35',
    }),
  };
});

import type { AppContextValue, AppState } from '@/context/AppContext';
import type { Role, TeamForUser } from '@/types';

// ── helpers ───────────────────────────────────────────────────────────────────

function makeRole(overrides: Partial<Role> = {}): Role {
  return {
    id: 'role-1',
    name: 'Spieler',
    color: '#1565C0',
    permissions: {},
    reasonRequired: false,
    ...overrides,
  } as Role;
}

function makeTeam(overrides: Partial<TeamForUser> = {}): TeamForUser {
  return {
    id: 'team1',
    name: 'SG Muster',
    description: '',
    memberCount: 5,
    myRoles: [],
    icon: '🏆',
    iconBg: '#E8F0FE',
    iconFg: '#4285F4',
    logo: null,
    photo: null,
    reasonVisibilityRoles: [],
    ...overrides,
  } as unknown as TeamForUser;
}

function makeState(overrides: Partial<AppState> = {}): AppState {
  return {
    primaryColor: '#4285F4',
    busy: null,
    members: [],
    finances: null,
    roles: [],
    ...overrides,
  } as AppState;
}

function makeApp(stateOverrides: Partial<AppState> = {}, methods: Partial<AppContextValue> = {}): AppContextValue {
  const state = makeState(stateOverrides);
  const app = {
    state,
    activeTeam: vi.fn().mockReturnValue(makeTeam()),
    can: vi.fn().mockReturnValue(true),
    setState: vi.fn(),
    onFile: vi.fn(),
    createTeam: vi.fn(),
    copyInvite: vi.fn(),
    saveTeamSettings: vi.fn(),
    saveTeamLogo: vi.fn(),
    saveTeamPhoto: vi.fn(),
    removeTeamPhoto: vi.fn(),
    setTeamIcon: vi.fn(),
    ...methods,
  } as unknown as AppContextValue;
  mockUseApp.mockReturnValue(app);
  return app;
}

// ════════════════════════════════════════════════════════════════════════════
// CreateTeamSheet
// ════════════════════════════════════════════════════════════════════════════

describe('CreateTeamSheet', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  const sheet = {} as never;

  it('renders the team name input field', () => {
    const app = makeApp();
    render(<CreateTeamSheet app={app} sheet={sheet} />);
    // Placeholder comes from t('team.teamNamePlaceholder') → 'z. B. C-Team TSC Schwarz-Gelb'
    expect(screen.getByPlaceholderText(/TSC/i)).toBeTruthy();
  });

  it('renders the create team hint text', () => {
    const app = makeApp();
    render(<CreateTeamSheet app={app} sheet={sheet} />);
    // t('team.createTeamHint') → 'Du wirst automatisch Administrator ...'
    expect(document.body.textContent).toContain('Administrator');
  });

  it('renders the create team button', () => {
    const app = makeApp();
    render(<CreateTeamSheet app={app} sheet={sheet} />);
    // t('team.createBtn') → 'Team anlegen'
    expect(screen.getByRole('button', { name: /Team anlegen/i })).toBeTruthy();
  });

  it('calls createTeam when the create button is clicked', async () => {
    const app = makeApp();
    render(<CreateTeamSheet app={app} sheet={sheet} />);
    fireEvent.change(screen.getByPlaceholderText(/TSC/i), { target: { value: 'Neues Team' } });
    fireEvent.click(screen.getByRole('button', { name: /Team anlegen/i }));
    await waitFor(() => {
      expect(app.createTeam).toHaveBeenCalledWith(expect.objectContaining({ name: 'Neues Team' }));
    });
  });

  it('renders all 12 icon buttons', () => {
    const app = makeApp();
    render(<CreateTeamSheet app={app} sheet={sheet} />);
    const icons = ['🏆', '⭐', '💃', '🕺', '🎭', '🔥', '👑', '🎯', '💎', '🦅', '⚡', '🌟'];
    icons.forEach((em) => {
      expect(screen.getByText(em)).toBeTruthy();
    });
  });

  it('clicking an icon button marks it selected', () => {
    const app = makeApp();
    render(<CreateTeamSheet app={app} sheet={sheet} />);
    const button = screen.getByText('👑').closest('button')!;
    fireEvent.click(button);
    expect(button).toHaveAttribute('aria-checked', 'true');
  });

  it('exposes icon selection as a radiogroup with aria-checked for screen-reader users', () => {
    const app = makeApp(); // defaultValues.icon is '⭐'
    render(<CreateTeamSheet app={app} sheet={sheet} />);
    const selected = screen.getByText('⭐').closest('button');
    const unselected = screen.getByText('🏆').closest('button');
    expect(selected).toHaveAttribute('role', 'radio');
    expect(selected).toHaveAttribute('aria-checked', 'true');
    expect(selected).toHaveAttribute('aria-label', '⭐');
    expect(unselected).toHaveAttribute('aria-checked', 'false');
  });

  it('shows photo upload label when no photo is selected', () => {
    const app = makeApp();
    render(<CreateTeamSheet app={app} sheet={sheet} />);
    // t('team.photoUpload') → 'Teamfoto hochladen (optional)'
    expect(document.body.textContent).toContain('Teamfoto hochladen');
  });

  it('shows photo selected label when a photo is picked', () => {
    const app = makeApp();
    render(<CreateTeamSheet app={app} sheet={sheet} />);
    const fileInput = document.querySelector('input[type="file"]')!;
    fireEvent.change(fileInput, { target: { files: [new File(['x'], 'photo.png', { type: 'image/png' })] } });
    const onFileCb = (app.onFile as ReturnType<typeof vi.fn>).mock.calls[0]![1] as (d: string) => void;
    act(() => onFileCb('data:image/png;base64,abc'));
    // t('team.photoSelected') → 'Teamfoto ausgewählt'
    expect(document.body.textContent).toContain('Teamfoto ausgewählt');
  });

  it('disables the create button and shows spinner when busy is "save"', () => {
    const app = makeApp({ busy: 'save' });
    render(<CreateTeamSheet app={app} sheet={sheet} />);
    const btn = screen.getByRole('button', { name: /Team anlegen/i });
    expect(btn).toBeDisabled();
  });

  it('renders a file input for photo upload', () => {
    const app = makeApp();
    render(<CreateTeamSheet app={app} sheet={sheet} />);
    const fileInput = document.querySelector('input[type="file"]');
    expect(fileInput).toBeTruthy();
    expect((fileInput as HTMLInputElement).accept).toBe('image/*');
  });

});

// ════════════════════════════════════════════════════════════════════════════
// InviteSheet — without invite
// ════════════════════════════════════════════════════════════════════════════

describe('InviteSheet — without invite', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  const sheetNoInvite = { invite: null } as never;

  it('shows generating text when invite is null', () => {
    const app = makeApp();
    render(<InviteSheet app={app} sheet={sheetNoInvite} />);
    // t('team.inviteGenerating') → 'Erzeuge Link…'
    expect(document.body.textContent).toContain('Erzeuge Link');
  });

  it('does NOT show the copy button when invite is null', () => {
    const app = makeApp();
    render(<InviteSheet app={app} sheet={sheetNoInvite} />);
    // t('team.inviteCopy') → 'Kopieren' — button only rendered when invite exists
    expect(screen.queryByText('Kopieren')).toBeNull();
  });

  it('does NOT show invite code section when invite is null', () => {
    const app = makeApp();
    render(<InviteSheet app={app} sheet={sheetNoInvite} />);
    // t('team.inviteCode') → 'Beitritts-Code:'
    expect(document.body.textContent).not.toContain('Beitritts-Code');
  });

  it('renders the hero icon element', () => {
    const app = makeApp();
    render(<InviteSheet app={app} sheet={sheetNoInvite} />);
    // The hero box renders the material icon name 'link'
    expect(document.body.textContent).toContain('link');
  });

  it('renders team short name in description', () => {
    const app = makeApp();
    // activeTeam returns { name: 'SG Muster' }, shortName('SG Muster') → 'SG'
    render(<InviteSheet app={app} sheet={sheetNoInvite} />);
    expect(document.body.textContent).toContain('SG');
  });

  // Regression test: the description used to interpolate the team's short
  // name into the template string and then locate it again via
  // .split(shortName(team.name)) to bold it -- breaking when the short name
  // collides with another word already in the surrounding sentence. The DE
  // template ('Teile diesen Link. Neue Mitglieder treten {teamName} bei.
  // Gültig 7 Tage.') contains the word "Neue"; a team named "Neue Elf" has
  // shortName -> "Neue", which the old .split() would also match inside
  // "Neue Mitglieder", producing duplicated/misplaced bolding.
  it('bolds only the team name, not a colliding word elsewhere in the sentence', () => {
    const app = makeApp({}, {});
    (app.activeTeam as ReturnType<typeof vi.fn>).mockReturnValue(makeTeam({ name: 'Neue Elf' }));
    const { container } = render(<InviteSheet app={app} sheet={sheetNoInvite} />);

    expect(document.body.textContent).toContain(
      'Teile diesen Link. Neue Mitglieder treten Neue bei. Gültig 7 Tage.',
    );
    const bolded = container.querySelectorAll('b');
    expect(bolded).toHaveLength(1);
    expect(bolded[0].textContent).toBe('Neue');
  });
});

// ════════════════════════════════════════════════════════════════════════════
// InviteSheet — with invite
// ════════════════════════════════════════════════════════════════════════════

describe('InviteSheet — with invite', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  const inviteData = {
    link: 'https://invite.link/join',
    code: 'ABC123',
    expiresAt: '2099-01-01',
  };
  const sheetWithInvite = { invite: inviteData } as never;

  it('renders the invite link when invite is provided', () => {
    const app = makeApp();
    render(<InviteSheet app={app} sheet={sheetWithInvite} />);
    expect(document.body.textContent).toContain('https://invite.link/join');
  });

  it('renders the copy button when invite is provided', () => {
    const app = makeApp();
    render(<InviteSheet app={app} sheet={sheetWithInvite} />);
    expect(screen.getByText('Kopieren')).toBeTruthy();
  });

  it('calls copyInvite when copy button is clicked', () => {
    const app = makeApp();
    render(<InviteSheet app={app} sheet={sheetWithInvite} />);
    fireEvent.click(screen.getByText('Kopieren'));
    expect(app.copyInvite).toHaveBeenCalledTimes(1);
  });

  it('renders the invite code', () => {
    const app = makeApp();
    render(<InviteSheet app={app} sheet={sheetWithInvite} />);
    expect(document.body.textContent).toContain('ABC123');
  });

  it('renders "Beitritts-Code:" label', () => {
    const app = makeApp();
    render(<InviteSheet app={app} sheet={sheetWithInvite} />);
    // t('team.inviteCode') → 'Beitritts-Code:'
    expect(document.body.textContent).toContain('Beitritts-Code');
  });

  it('shows "Kopiert" label when sheet.copied is true', () => {
    const app = makeApp();
    const sheetCopied = { invite: inviteData, copied: true } as never;
    render(<InviteSheet app={app} sheet={sheetCopied} />);
    // t('team.inviteCopied') → 'Kopiert'
    expect(document.body.textContent).toContain('Kopiert');
  });

  it('shows "Kopieren" label when sheet.copied is falsy', () => {
    const app = makeApp();
    render(<InviteSheet app={app} sheet={sheetWithInvite} />);
    expect(document.body.textContent).toContain('Kopieren');
  });
});

// ════════════════════════════════════════════════════════════════════════════
// TeamSettingsSheet
// ════════════════════════════════════════════════════════════════════════════

describe('TeamSettingsSheet', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  const sheet = {} as never;

  function makeSettingsApp(teamOverrides: Partial<TeamForUser> = {}, roles: Role[] = []): AppContextValue {
    const app = makeApp({ roles });
    (app.activeTeam as ReturnType<typeof vi.fn>).mockReturnValue(makeTeam(teamOverrides));
    return app;
  }

  it('renders the team name input field', () => {
    const app = makeSettingsApp();
    render(<TeamSettingsSheet app={app} sheet={sheet} />);
    // t('team.settingsNamePlaceholder') → 'Team-Name'
    expect(screen.getByPlaceholderText('Team-Name')).toBeTruthy();
  });

  it('renders the description textarea', () => {
    const app = makeSettingsApp();
    render(<TeamSettingsSheet app={app} sheet={sheet} />);
    // t('team.settingsDescPlaceholder') → 'Kurze Beschreibung des Teams…'
    expect(screen.getByPlaceholderText(/Kurze Beschreibung/i)).toBeTruthy();
  });

  // Regression test: the description textarea had no client-side maxLength,
  // unlike the name field on the same form, matching the backend's
  // 10,000-char validate.MaxLen bound.
  it('caps the description textarea at 10000 characters matching the backend limit', () => {
    const app = makeSettingsApp();
    render(<TeamSettingsSheet app={app} sheet={sheet} />);
    const textarea = screen.getByPlaceholderText(/Kurze Beschreibung/i) as HTMLTextAreaElement;
    expect(textarea.maxLength).toBe(10000);
  });

  it('renders the logo section heading', () => {
    const app = makeSettingsApp();
    render(<TeamSettingsSheet app={app} sheet={sheet} />);
    // t('team.settingsLogoSection') → 'Logo'
    expect(document.body.textContent).toContain('Logo');
  });

  it('renders the photo section heading', () => {
    const app = makeSettingsApp();
    render(<TeamSettingsSheet app={app} sheet={sheet} />);
    // t('team.settingsPhotoSection') → 'Gruppenbild'
    expect(document.body.textContent).toContain('Gruppenbild');
  });

  it('renders the visibility section heading', () => {
    const app = makeSettingsApp();
    render(<TeamSettingsSheet app={app} sheet={sheet} />);
    // t('team.settingsVisSection') → 'Sichtbarkeit von Absage-Kommentaren'
    expect(document.body.textContent).toContain('Sichtbarkeit');
  });

  it('renders the save button', () => {
    const app = makeSettingsApp();
    render(<TeamSettingsSheet app={app} sheet={sheet} />);
    // t('team.settingsSave') → 'Einstellungen speichern'
    expect(screen.getByRole('button', { name: /Einstellungen speichern/i })).toBeTruthy();
  });

  it('calls saveTeamSettings when save button is clicked', async () => {
    const app = makeSettingsApp();
    render(<TeamSettingsSheet app={app} sheet={sheet} />);
    fireEvent.click(screen.getByRole('button', { name: /Einstellungen speichern/i }));
    await waitFor(() => {
      expect(app.saveTeamSettings).toHaveBeenCalledWith(expect.objectContaining({ name: 'SG Muster' }));
    });
  });

  it('disables save button while saveTeamSettings is pending', async () => {
    const app = makeSettingsApp();
    let resolveSave!: () => void;
    (app.saveTeamSettings as ReturnType<typeof vi.fn>).mockImplementation(
      () => new Promise<void>((resolve) => (resolveSave = resolve)),
    );
    render(<TeamSettingsSheet app={app} sheet={sheet} />);
    const btn = screen.getByRole('button', { name: /Einstellungen speichern/i });
    fireEvent.click(btn);
    await waitFor(() => expect(btn).toBeDisabled());
    resolveSave();
  });

  it('renders 12 icon buttons in the logo section', () => {
    const app = makeSettingsApp();
    render(<TeamSettingsSheet app={app} sheet={sheet} />);
    const icons = ['🏆', '⭐', '💃', '🕺', '🎭', '🔥', '👑', '🎯', '💎', '🦅', '⚡', '🌟'];
    icons.forEach((em) => {
      expect(screen.getAllByText(em).length).toBeGreaterThanOrEqual(1);
    });
  });

  it('calls setTeamIcon when an icon button is clicked', () => {
    const app = makeSettingsApp();
    render(<TeamSettingsSheet app={app} sheet={sheet} />);
    fireEvent.click(screen.getAllByText('⭐')[0]!);
    expect(app.setTeamIcon).toHaveBeenCalledWith('⭐');
  });

  it('renders logo upload button when no logo is set', () => {
    const app = makeSettingsApp({ logo: null });
    render(<TeamSettingsSheet app={app} sheet={sheet} />);
    // t('team.settingsLogoUpload') → 'Bild-Logo hochladen'
    expect(document.body.textContent).toContain('Bild-Logo hochladen');
  });

  it('renders logo change button when logo is already set', () => {
    const app = makeSettingsApp({ logo: 'data:image/png;base64,abc' });
    render(<TeamSettingsSheet app={app} sheet={sheet} />);
    // t('team.settingsLogoChange') → 'Logo ändern'
    expect(document.body.textContent).toContain('Logo ändern');
  });

  it('renders photo upload button when team has no photo', () => {
    const app = makeSettingsApp();
    // activeTeam default has photo: null
    render(<TeamSettingsSheet app={app} sheet={sheet} />);
    // t('team.settingsPhotoUpload') → 'Gruppenbild hochladen'
    expect(document.body.textContent).toContain('Gruppenbild hochladen');
  });

  it('renders photo change button when team already has a photo', () => {
    const app = makeSettingsApp({ photo: 'data:image/png;base64,xyz' });
    render(<TeamSettingsSheet app={app} sheet={sheet} />);
    // t('team.settingsPhotoChange') → 'Bild ändern'
    expect(document.body.textContent).toContain('Bild ändern');
  });

  it('does not render a remove-photo button when the team has no photo', () => {
    const app = makeSettingsApp();
    render(<TeamSettingsSheet app={app} sheet={sheet} />);
    // t('team.settingsPhotoRemove') → 'Bild entfernen'
    expect(screen.queryByLabelText('Bild entfernen')).toBeNull();
  });

  it('renders a remove-photo button when the team has a photo, and calls removeTeamPhoto on click', () => {
    const app = makeSettingsApp({ photo: 'data:image/png;base64,xyz' });
    render(<TeamSettingsSheet app={app} sheet={sheet} />);
    const removeBtn = screen.getByLabelText('Bild entfernen');
    fireEvent.click(removeBtn);
    expect(app.removeTeamPhoto).toHaveBeenCalled();
  });

  it('renders role chips in visibility section', () => {
    const roles = [
      makeRole({ id: 'r1', name: 'Trainer', color: '#00796B' }),
      makeRole({ id: 'r2', name: 'Spieler', color: '#1565C0' }),
    ];
    const app = makeSettingsApp({ reasonVisibilityRoles: [] }, roles);
    render(<TeamSettingsSheet app={app} sheet={sheet} />);
    expect(screen.getByText('Trainer')).toBeTruthy();
    expect(screen.getByText('Spieler')).toBeTruthy();
  });

  it('clicking a role chip marks it checked', () => {
    const roles = [makeRole({ id: 'r1', name: 'Trainer', color: '#00796B' })];
    const app = makeSettingsApp({ reasonVisibilityRoles: [] }, roles);
    render(<TeamSettingsSheet app={app} sheet={sheet} />);
    const chip = screen.getByText('Trainer').closest('button')!;
    fireEvent.click(chip);
    expect(chip).toHaveAttribute('aria-checked', 'true');
  });

  it('renders check icon for roles already in reasonRoles', () => {
    const roles = [makeRole({ id: 'r1', name: 'Trainer', color: '#00796B' })];
    const app = makeSettingsApp({ reasonVisibilityRoles: ['r1'] }, roles);
    render(<TeamSettingsSheet app={app} sheet={sheet} />);
    // When selected, Sym name="check" renders the text 'check' inside the button
    expect(document.body.textContent).toContain('check');
  });

  it('shows icon emoji in logo preview box when no logo image is set', () => {
    const app = makeSettingsApp({ icon: '👑', logo: null });
    render(<TeamSettingsSheet app={app} sheet={sheet} />);
    // The logoPreview box displays the icon emoji when logo is null
    expect(screen.getAllByText('👑').length).toBeGreaterThanOrEqual(1);
  });

  it('renders two file inputs (one for logo, one for photo)', () => {
    const app = makeSettingsApp();
    render(<TeamSettingsSheet app={app} sheet={sheet} />);
    const fileInputs = document.querySelectorAll('input[type="file"]');
    expect(fileInputs.length).toBe(2);
  });

  it('renders the photo hint text', () => {
    const app = makeSettingsApp();
    render(<TeamSettingsSheet app={app} sheet={sheet} />);
    // t('team.settingsPhotoHint') → 'Wird als Titelbild auf der Startseite ...'
    expect(document.body.textContent).toContain('Titelbild');
  });

  it('renders the visibility hint text', () => {
    const app = makeSettingsApp();
    render(<TeamSettingsSheet app={app} sheet={sheet} />);
    // t('team.settingsVisHint') → 'Welche Rollen dürfen die Kommentare ...'
    expect(document.body.textContent).toContain('Kommentare');
  });

  // Regression test: the logo/photo file input's onChange used to call
  // app.onFile(e, cb) with cb reading app.state.activeTeamId itself only
  // once the FileReader read completed. Since onFile's read is async, a
  // team switch between file selection and the read completing meant the
  // upload silently landed on whichever team was active when the read
  // finished, not the team it was actually picked for -- a cross-team data
  // write, not just stale UI. The fix snapshots activeTeamId synchronously
  // in onChange, before onFile/FileReader ever starts, and threads it
  // through explicitly.
  it('saves a logo upload to the team it was picked for, even if the team changed before the read completed', () => {
    const app = makeApp({ activeTeamId: 'team1' });
    render(<TeamSettingsSheet app={app} sheet={sheet} />);
    const [logoInput] = document.querySelectorAll('input[type="file"]');
    fireEvent.change(logoInput!, { target: { files: [new File(['x'], 'logo.png', { type: 'image/png' })] } });

    expect(app.onFile).toHaveBeenCalledTimes(1);
    const onFileCb = (app.onFile as ReturnType<typeof vi.fn>).mock.calls[0]![1] as (d: string) => void;

    // Simulate the user switching teams while the FileReader read is still
    // in flight -- app.state is mutated in place, same object useApp() keeps
    // returning, mirroring how the real AppContext state updates.
    app.state.activeTeamId = 'team2';

    // FileReader.onload fires now, after the switch.
    onFileCb('data:image/png;base64,logodata');

    expect(app.saveTeamLogo).toHaveBeenCalledWith('data:image/png;base64,logodata', 'team1');
  });
});
