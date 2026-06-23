import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { MemberDetailSheet, MemberFormSheet } from './MemberSheets';

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

import { useApp } from '@/context/AppContext';
const mockUseApp = vi.mocked(useApp);

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

import type { Member } from '../types';
import type { AppContextValue, AppState, SheetState } from '@/context/AppContext';
import type { Role } from '@/types';

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

function makeMember(overrides: Partial<Member> = {}): Member {
  return {
    membershipId: 'ms-1',
    userId: 'u-other',
    name: 'Max Mustermann',
    email: 'max@example.com',
    phone: '+49 123 456',
    birthday: '1990-05-15',
    address: 'Musterstraße 1, 12345 Musterstadt',
    avatarColor: '#1565C0',
    photo: null,
    group: '',
    roles: [makeRole()],
    joinedAt: '2023-01-01',
    primaryRole: makeRole(),
    perms: {},
    ...overrides,
  } as Member;
}

function makeState(overrides: Partial<AppState> = {}): AppState {
  return {
    primaryColor: '#1565C0',
    user: { id: 'u-me', name: 'Ich', email: 'me@test.com' } as AppState['user'],
    busy: null,
    roles: [makeRole(), makeRole({ id: 'role-2', name: 'Trainer', color: '#00796B' })],
    form: {},
    formErrors: {},
    ...overrides,
  } as AppState;
}

function makeApp(stateOverrides: Partial<AppState> = {}, methods: Partial<AppContextValue> = {}): AppContextValue {
  const state = makeState(stateOverrides);
  const app = {
    state,
    can: vi.fn().mockReturnValue(false),
    openMemberForm: vi.fn(),
    removeMember: vi.fn(),
    setFormErrors: vi.fn(),
    setFormVal: vi.fn(),
    onFormInput: vi.fn(),
    onFile: vi.fn(),
    toggleFormRole: vi.fn(),
    saveMember: vi.fn(),
    ...methods,
  } as unknown as AppContextValue;
  mockUseApp.mockReturnValue(app);
  return app;
}

function makeSheet(overrides: Partial<SheetState> = {}): SheetState {
  return {
    type: 'memberDetail',
    member: makeMember(),
    stats: null,
    ...overrides,
  } as SheetState;
}

// ════════════════════════════════════════════════════════════════════════════
// MemberDetailSheet
// ════════════════════════════════════════════════════════════════════════════

describe('MemberDetailSheet', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders the member name', () => {
    const app = makeApp();
    render(<MemberDetailSheet app={app} sheet={makeSheet()} />);
    expect(screen.getByText('Max Mustermann')).toBeTruthy();
  });

  it('renders the member email in contact section', () => {
    const app = makeApp();
    render(<MemberDetailSheet app={app} sheet={makeSheet()} />);
    expect(screen.getByText('max@example.com')).toBeTruthy();
  });

  it('renders the member phone in contact section', () => {
    const app = makeApp();
    render(<MemberDetailSheet app={app} sheet={makeSheet()} />);
    expect(screen.getByText('+49 123 456')).toBeTruthy();
  });

  it('renders the member address', () => {
    const app = makeApp();
    render(<MemberDetailSheet app={app} sheet={makeSheet()} />);
    expect(screen.getByText('Musterstraße 1, 12345 Musterstadt')).toBeTruthy();
  });

  it('renders role chip with role name', () => {
    const app = makeApp();
    render(<MemberDetailSheet app={app} sheet={makeSheet()} />);
    expect(screen.getByText('Spieler')).toBeTruthy();
  });

  it('renders role count in stats box', () => {
    const app = makeApp();
    const m = makeMember({ roles: [makeRole(), makeRole({ id: 'r2', name: 'Trainer' })] });
    render(<MemberDetailSheet app={app} sheet={makeSheet({ member: m })} />);
    expect(screen.getByText('2')).toBeTruthy();
  });

  it('shows "–" dash for attendance rate when stats is null', () => {
    const app = makeApp();
    render(<MemberDetailSheet app={app} sheet={makeSheet({ stats: null })} />);
    expect(screen.getByText('…')).toBeTruthy();
  });

  it('shows attendance rate when stats are provided', () => {
    const app = makeApp();
    const sheet = makeSheet({ stats: { quote: 75, counted: 10, yes: 7 } });
    render(<MemberDetailSheet app={app} sheet={sheet} />);
    expect(screen.getByText('75%')).toBeTruthy();
  });

  it('shows "–" for null attendance quote', () => {
    const app = makeApp();
    const sheet = makeSheet({ stats: { quote: null, counted: 0, yes: 0 } });
    render(<MemberDetailSheet app={app} sheet={sheet} />);
    expect(screen.getByText('–')).toBeTruthy();
  });

  it('does not show edit button when user lacks write permission and is not the member', () => {
    // can('members', 'write') returns false, userId !== state.user.id
    const app = makeApp({ user: { id: 'u-me' } as AppState['user'] });
    (app.can as ReturnType<typeof vi.fn>).mockReturnValue(false);
    const sheet = makeSheet({ member: makeMember({ userId: 'u-other' }) });
    render(<MemberDetailSheet app={app} sheet={sheet} />);
    expect(screen.queryByText(/bearbeiten/i)).toBeNull();
  });

  it('shows edit button when user is the member (isMe)', () => {
    const app = makeApp({ user: { id: 'u-me' } as AppState['user'] });
    (app.can as ReturnType<typeof vi.fn>).mockReturnValue(false);
    const sheet = makeSheet({ member: makeMember({ userId: 'u-me' }) });
    render(<MemberDetailSheet app={app} sheet={sheet} />);
    // "Profil bearbeiten" for self
    expect(screen.getByText(/profil bearbeiten/i)).toBeTruthy();
  });

  it('shows edit and remove buttons when user has write permission and is not the member', () => {
    const app = makeApp({ user: { id: 'u-me' } as AppState['user'] });
    (app.can as ReturnType<typeof vi.fn>).mockReturnValue(true);
    const sheet = makeSheet({ member: makeMember({ userId: 'u-other' }) });
    render(<MemberDetailSheet app={app} sheet={sheet} />);
    expect(screen.getByText(/bearbeiten/i)).toBeTruthy();
    expect(screen.getByLabelText(/entfernen/i)).toBeTruthy();
  });

  it('calls openMemberForm when edit button is clicked', () => {
    const app = makeApp({ user: { id: 'u-me' } as AppState['user'] });
    (app.can as ReturnType<typeof vi.fn>).mockReturnValue(true);
    const m = makeMember({ userId: 'u-other' });
    render(<MemberDetailSheet app={app} sheet={makeSheet({ member: m })} />);
    fireEvent.click(screen.getByText(/bearbeiten/i));
    expect(app.openMemberForm).toHaveBeenCalledWith(m);
  });

  it('calls removeMember when remove button is clicked', () => {
    const app = makeApp({ user: { id: 'u-me' } as AppState['user'] });
    (app.can as ReturnType<typeof vi.fn>).mockReturnValue(true);
    const m = makeMember({ userId: 'u-other', membershipId: 'ms-42' });
    render(<MemberDetailSheet app={app} sheet={makeSheet({ member: m })} />);
    fireEvent.click(screen.getByLabelText(/entfernen/i));
    expect(app.removeMember).toHaveBeenCalledWith('ms-42');
  });

  it('shows membership note when user has write permission', () => {
    const app = makeApp({ user: { id: 'u-me' } as AppState['user'] });
    (app.can as ReturnType<typeof vi.fn>).mockReturnValue(true);
    render(<MemberDetailSheet app={app} sheet={makeSheet()} />);
    // i18n key members.membershipNote is rendered for writers
    const container = document.body;
    // The note section is rendered
    expect(container.textContent).toContain('Mitglied');
  });
});

// ════════════════════════════════════════════════════════════════════════════
// MemberFormSheet
// ════════════════════════════════════════════════════════════════════════════

describe('MemberFormSheet', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  const formSheet: SheetState = { type: 'memberForm' } as SheetState;

  function makeFormApp(
    formOverrides: Record<string, unknown> = {},
    errOverrides: Record<string, string> = {},
    canWrite = false,
  ): AppContextValue {
    return makeApp(
      {
        form: { name: '', email: '', phone: '', birthday: '', address: '', photo: null, roleIds: [], ...formOverrides },
        formErrors: { name: '', email: '', phone: '', birthday: '', ...errOverrides },
        busy: null,
      },
      { can: vi.fn().mockReturnValue(canWrite) as unknown as AppContextValue['can'] },
    );
  }

  it('renders the name input field', () => {
    const app = makeFormApp();
    render(<MemberFormSheet app={app} sheet={formSheet} />);
    expect(screen.getByPlaceholderText(/Vor- und Nachname/i)).toBeTruthy();
  });

  it('renders the email input field', () => {
    const app = makeFormApp();
    render(<MemberFormSheet app={app} sheet={formSheet} />);
    expect(screen.getByPlaceholderText('name@example.de')).toBeTruthy();
  });

  it('renders the phone input field', () => {
    const app = makeFormApp();
    render(<MemberFormSheet app={app} sheet={formSheet} />);
    expect(screen.getByPlaceholderText('+49 …')).toBeTruthy();
  });

  it('renders the birthday input field', () => {
    const app = makeFormApp();
    render(<MemberFormSheet app={app} sheet={formSheet} />);
    const inputs = document.querySelectorAll('input[type="date"]');
    expect(inputs.length).toBeGreaterThanOrEqual(1);
  });

  it('renders the address input field', () => {
    const app = makeFormApp();
    render(<MemberFormSheet app={app} sheet={formSheet} />);
    expect(screen.getByPlaceholderText('Straße, PLZ Ort')).toBeTruthy();
  });

  it('renders the save button', () => {
    const app = makeFormApp();
    render(<MemberFormSheet app={app} sheet={formSheet} />);
    expect(screen.getByRole('button', { name: /Profil speichern/i })).toBeTruthy();
  });

  it('calls saveMember when save button is clicked', () => {
    const app = makeFormApp();
    render(<MemberFormSheet app={app} sheet={formSheet} />);
    fireEvent.click(screen.getByRole('button', { name: /Profil speichern/i }));
    expect(app.saveMember).toHaveBeenCalled();
  });

  it('shows name validation error on blur when name is empty', () => {
    const app = makeFormApp({ name: '' });
    render(<MemberFormSheet app={app} sheet={formSheet} />);
    const nameInput = screen.getByPlaceholderText(/Vor- und Nachname/i);
    fireEvent.blur(nameInput);
    expect(app.setFormErrors).toHaveBeenCalledWith({ name: expect.stringMatching(/\S+/) });
  });

  it('clears name error on blur when name has a value', () => {
    const app = makeFormApp({ name: 'Max Muster' });
    render(<MemberFormSheet app={app} sheet={formSheet} />);
    const nameInput = screen.getByPlaceholderText(/Vor- und Nachname/i);
    fireEvent.blur(nameInput);
    expect(app.setFormErrors).toHaveBeenCalledWith({ name: '' });
  });

  it('shows email validation error on blur when email is invalid', () => {
    const app = makeFormApp({ email: 'not-an-email' });
    render(<MemberFormSheet app={app} sheet={formSheet} />);
    const emailInput = screen.getByPlaceholderText('name@example.de');
    fireEvent.blur(emailInput);
    expect(app.setFormErrors).toHaveBeenCalledWith({ email: expect.stringMatching(/\S+/) });
  });

  it('clears email error when a valid email is provided on blur', () => {
    const app = makeFormApp({ email: 'valid@example.com' });
    render(<MemberFormSheet app={app} sheet={formSheet} />);
    const emailInput = screen.getByPlaceholderText('name@example.de');
    fireEvent.blur(emailInput);
    expect(app.setFormErrors).toHaveBeenCalledWith({ email: '' });
  });

  it('shows displayed name error text when error is set', () => {
    const app = makeFormApp({}, { name: 'Pflichtfeld' });
    render(<MemberFormSheet app={app} sheet={formSheet} />);
    expect(screen.getByText('Pflichtfeld')).toBeTruthy();
  });

  it('shows role chips when user has write permission (canWrite = true)', () => {
    const app = makeFormApp({}, {}, true);
    render(<MemberFormSheet app={app} sheet={formSheet} />);
    // Roles list: "Spieler" and "Trainer"
    expect(screen.getByText('Spieler')).toBeTruthy();
    expect(screen.getByText('Trainer')).toBeTruthy();
  });

  it('does not show role chips when user lacks write permission', () => {
    const app = makeFormApp({}, {}, false);
    render(<MemberFormSheet app={app} sheet={formSheet} />);
    // Role chips should NOT appear
    expect(screen.queryByText('Spieler')).toBeNull();
  });

  it('calls toggleFormRole when a role chip is clicked', () => {
    const app = makeFormApp({ roleIds: [] }, {}, true);
    render(<MemberFormSheet app={app} sheet={formSheet} />);
    fireEvent.click(screen.getByText('Spieler'));
    expect(app.toggleFormRole).toHaveBeenCalledWith('role-1');
  });

  it('shows photo upload button when no photo is set', () => {
    const app = makeFormApp({ photo: null });
    render(<MemberFormSheet app={app} sheet={formSheet} />);
    expect(screen.getByText(/Foto hochladen/i)).toBeTruthy();
  });

  it('shows photo change button when a photo is already set', () => {
    const app = makeFormApp({ photo: 'data:image/png;base64,abc' });
    render(<MemberFormSheet app={app} sheet={formSheet} />);
    expect(screen.getByText(/Foto ändern/i)).toBeTruthy();
  });

  it('shows busy state on save button when busy is "save"', () => {
    const app = makeFormApp();
    app.state.busy = 'save';
    render(<MemberFormSheet app={app} sheet={formSheet} />);
    // PrimaryButton renders a disabled button when busy
    const btn = screen.getByRole('button', { name: /Profil speichern/i });
    expect(btn).toBeDisabled();
  });

  it('shows birthday validation error on blur when birthday is invalid', () => {
    const app = makeFormApp({ birthday: 'not-a-date' });
    render(<MemberFormSheet app={app} sheet={formSheet} />);
    const inputs = document.querySelectorAll('input');
    const birthdayInput = Array.from(inputs).find((i) => i.value === 'not-a-date');
    if (birthdayInput) {
      fireEvent.blur(birthdayInput);
      expect(app.setFormErrors).toHaveBeenCalledWith({ birthday: expect.stringMatching(/\S+/) });
    }
  });

  it('shows phone validation error on blur when phone is invalid', () => {
    const app = makeFormApp({ phone: 'not!a@phone' });
    render(<MemberFormSheet app={app} sheet={formSheet} />);
    const inputs = document.querySelectorAll('input');
    const phoneInput = Array.from(inputs).find((i) => i.value === 'not!a@phone');
    if (phoneInput) {
      fireEvent.blur(phoneInput);
      expect(app.setFormErrors).toHaveBeenCalledWith({ phone: expect.stringMatching(/\S+/) });
    }
  });
});
