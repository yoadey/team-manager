import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, act } from '@testing-library/react';
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
    setState: vi.fn(),
    onFile: vi.fn(),
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

  // Regression test: sheet.member is looked up from the already-loaded
  // local member list (not an async fetch), so it can genuinely be
  // undefined -- e.g. a stale bookmarked or browser-back/forward URL for a
  // member who has since been removed. This used to force-unwrap into a
  // render-time crash instead of a graceful empty state.
  it('renders a not-found state instead of crashing when member is undefined', () => {
    const app = makeApp();
    expect(() => render(<MemberDetailSheet app={app} sheet={makeSheet({ member: undefined })} />)).not.toThrow();
    expect(screen.getByText('Dieses Mitglied wurde nicht gefunden. Es könnte entfernt worden sein.')).toBeTruthy();
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

  // self: true by default since most of these tests exercise generic form
  // behavior, not the self-vs-admin-editing-someone-else distinction; the
  // photo-visibility tests below cover the self: false case explicitly.
  function makeFormSheet(formOverrides: Record<string, unknown> = {}, self = true): SheetState {
    return {
      type: 'memberForm',
      self,
      formInitial: { name: '', email: '', phone: '', birthday: '', address: '', photo: null, roleIds: [], ...formOverrides },
    } as SheetState;
  }
  const formSheet = makeFormSheet();

  function makeFormApp(canWrite = false): AppContextValue {
    return makeApp({ busy: null }, { can: vi.fn().mockReturnValue(canWrite) as unknown as AppContextValue['can'] });
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

  it('caps the birthday input at 1900-01-01 matching the backend limit', () => {
    const app = makeFormApp();
    render(<MemberFormSheet app={app} sheet={formSheet} />);
    const input = document.querySelector('input[type="date"]') as HTMLInputElement;
    expect(input.min).toBe('1900-01-01');
  });

  it('renders the address input field', () => {
    const app = makeFormApp();
    render(<MemberFormSheet app={app} sheet={formSheet} />);
    expect(screen.getByPlaceholderText('Straße, PLZ Ort')).toBeTruthy();
  });

  // Regression test: name/phone/address had no client-side maxLength, unlike
  // every other create/edit form field, matching the backend's
  // validate.MaxLen bounds (255 / 32 / 500).
  it('caps the name input at 255 characters matching the backend limit', () => {
    const app = makeFormApp();
    render(<MemberFormSheet app={app} sheet={formSheet} />);
    const input = screen.getByPlaceholderText('Vor- und Nachname') as HTMLInputElement;
    expect(input.maxLength).toBe(255);
  });

  it('caps the phone input at 32 characters matching the backend limit', () => {
    const app = makeFormApp();
    render(<MemberFormSheet app={app} sheet={formSheet} />);
    const input = screen.getByPlaceholderText('+49 …') as HTMLInputElement;
    expect(input.maxLength).toBe(32);
  });

  it('caps the address input at 500 characters matching the backend limit', () => {
    const app = makeFormApp();
    render(<MemberFormSheet app={app} sheet={formSheet} />);
    const input = screen.getByPlaceholderText('Straße, PLZ Ort') as HTMLInputElement;
    expect(input.maxLength).toBe(500);
  });

  it('caps the email input at 254 characters matching the backend limit', () => {
    const app = makeFormApp();
    render(<MemberFormSheet app={app} sheet={formSheet} />);
    const input = screen.getByPlaceholderText('name@example.de') as HTMLInputElement;
    expect(input.maxLength).toBe(254);
  });

  it('renders the save button', () => {
    const app = makeFormApp();
    render(<MemberFormSheet app={app} sheet={formSheet} />);
    expect(screen.getByRole('button', { name: /Profil speichern/i })).toBeTruthy();
  });

  it('calls saveMember when save button is clicked', async () => {
    const app = makeFormApp();
    render(<MemberFormSheet app={app} sheet={makeFormSheet({ name: 'Alice' })} />);
    fireEvent.click(screen.getByRole('button', { name: /Profil speichern/i }));
    await vi.waitFor(() => {
      expect(app.saveMember).toHaveBeenCalled();
    });
  });

  it('disables the save button when the name is empty', () => {
    const app = makeFormApp();
    render(<MemberFormSheet app={app} sheet={makeFormSheet({ name: '' })} />);
    fireEvent.click(screen.getByRole('button', { name: /Profil speichern/i }));
    expect(app.saveMember).not.toHaveBeenCalled();
  });

  it('shows name validation error on blur when name is empty', async () => {
    const app = makeFormApp();
    render(<MemberFormSheet app={app} sheet={makeFormSheet({ name: '' })} />);
    const nameInput = screen.getByPlaceholderText(/Vor- und Nachname/i);
    fireEvent.blur(nameInput);
    await vi.waitFor(() => {
      expect(screen.getByText(/Pflichtfeld|erforderlich|fehlt/i)).toBeTruthy();
    });
  });

  it('shows email validation error on blur when email is invalid', async () => {
    const app = makeFormApp();
    render(<MemberFormSheet app={app} sheet={makeFormSheet({ name: 'Alice', email: 'not-an-email' })} />);
    const emailInput = screen.getByPlaceholderText('name@example.de');
    fireEvent.blur(emailInput);
    await vi.waitFor(() => {
      expect(emailInput.getAttribute('aria-invalid')).toBe('true');
    });
  });

  it('shows role chips when user has write permission (canWrite = true)', () => {
    const app = makeFormApp(true);
    render(<MemberFormSheet app={app} sheet={formSheet} />);
    // Roles list: "Spieler" and "Trainer"
    expect(screen.getByText('Spieler')).toBeTruthy();
    expect(screen.getByText('Trainer')).toBeTruthy();
  });

  it('does not show role chips when user lacks write permission', () => {
    const app = makeFormApp(false);
    render(<MemberFormSheet app={app} sheet={formSheet} />);
    // Role chips should NOT appear
    expect(screen.queryByText('Spieler')).toBeNull();
  });

  it('toggles a role chip selection when clicked', () => {
    const app = makeFormApp(true);
    render(<MemberFormSheet app={app} sheet={makeFormSheet({ roleIds: [] })} />);
    const chip = screen.getByText('Spieler').closest('[role="checkbox"]')!;
    expect(chip).toHaveAttribute('aria-checked', 'false');
    fireEvent.click(chip);
    expect(chip).toHaveAttribute('aria-checked', 'true');
  });

  it('shows photo upload button when no photo is set', () => {
    const app = makeFormApp();
    render(<MemberFormSheet app={app} sheet={makeFormSheet({ photo: null })} />);
    expect(screen.getByText(/Foto hochladen/i)).toBeTruthy();
  });

  it('shows photo change button when a photo is already set', () => {
    const app = makeFormApp();
    render(<MemberFormSheet app={app} sheet={makeFormSheet({ photo: 'data:image/png;base64,abc' })} />);
    expect(screen.getByText(/Foto ändern/i)).toBeTruthy();
  });

  it("hides the photo control when an admin edits someone else's profile", () => {
    // No backend endpoint exists to set another member's photo, so the
    // control must not be shown at all when self is false.
    const app = makeFormApp(true);
    render(<MemberFormSheet app={app} sheet={makeFormSheet({ photo: null }, false)} />);
    expect(screen.queryByText(/Foto hochladen/i)).toBeNull();
    expect(screen.queryByText(/Foto ändern/i)).toBeNull();
  });

  it('applies the picked photo to the preview via the form (no global state write)', () => {
    const app = makeFormApp();
    render(<MemberFormSheet app={app} sheet={makeFormSheet({ photo: null })} />);
    const fileInput = document.querySelector('input[type="file"]')!;
    fireEvent.change(fileInput, { target: { files: [new File(['x'], 'photo.png', { type: 'image/png' })] } });

    expect(app.onFile).toHaveBeenCalledTimes(1);
    const onFileCb = (app.onFile as ReturnType<typeof vi.fn>).mock.calls[0][1] as (d: string) => void;
    act(() => onFileCb('data:image/png;base64,photodata'));

    // The photo lives purely in this form's own RHF state now -- no
    // app.setState() write, so there's no shared buffer for another sheet to race on.
    expect(app.setState).not.toHaveBeenCalled();
    expect(screen.getByText(/Foto ändern/i)).toBeTruthy();
  });

  it('shows busy state on save button while saving', () => {
    const app = makeApp({ busy: null, savingMember: true });
    render(<MemberFormSheet app={app} sheet={makeFormSheet({ name: 'Alice' })} />);
    // PrimaryButton renders a disabled button when busy
    const btn = screen.getByRole('button', { name: /Profil speichern/i });
    expect(btn).toBeDisabled();
  });

  it('shows birthday validation error on blur when birthday is invalid', async () => {
    const app = makeFormApp();
    render(<MemberFormSheet app={app} sheet={makeFormSheet({ name: 'Alice', birthday: '2099-01-01' })} />);
    const birthdayInput = document.querySelector('input[type="date"]') as HTMLInputElement;
    fireEvent.blur(birthdayInput);
    await vi.waitFor(() => {
      expect(birthdayInput.getAttribute('aria-invalid')).toBe('true');
    });
  });

  it('shows phone validation error on blur when phone is invalid', async () => {
    const app = makeFormApp();
    render(<MemberFormSheet app={app} sheet={makeFormSheet({ name: 'Alice', phone: 'not!a@phone' })} />);
    const inputs = document.querySelectorAll('input');
    const phoneInput = Array.from(inputs).find((i) => i.value === 'not!a@phone')!;
    fireEvent.blur(phoneInput);
    await vi.waitFor(() => {
      expect(phoneInput.getAttribute('aria-invalid')).toBe('true');
    });
  });
});
