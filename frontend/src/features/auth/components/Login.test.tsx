import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { Login } from './Login';

vi.mock('@/context/AppContext', () => ({
  useApp: vi.fn(),
  useAppActions: vi.fn().mockReturnValue({}),
}));

import { useApp } from '@/context/AppContext';
const mockUseApp = vi.mocked(useApp);

const googleProvider = {
  id: 'google',
  name: 'Google',
  sub: 'Weiter mit Google',
  bg: '#fff',
  fg: '#000',
  border: true,
  glyph: 'G',
};

const appleProvider = {
  id: 'apple',
  name: 'Apple',
  sub: 'Weiter mit Apple',
  bg: '#000',
  fg: '#fff',
  border: false,
  glyph: 'A',
};

function makeApp(
  overrides: { providers?: (typeof googleProvider)[]; busy?: string | null; error?: string | null } = {},
) {
  const doLogin = vi.fn();
  const app = {
    state: {
      providers: overrides.providers ?? [googleProvider],
      busy: overrides.busy ?? null,
      error: overrides.error ?? null,
    },
    doLogin,
  };
  mockUseApp.mockReturnValue(app as unknown as ReturnType<typeof useApp>);
  return app;
}

describe('Login', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders the login page', () => {
    makeApp();
    render(<Login />);
    expect(screen.getByText('Teamverwaltung')).toBeTruthy();
  });

  it('renders a button for each provider', () => {
    makeApp({ providers: [googleProvider, appleProvider] });
    render(<Login />);
    expect(screen.getByText('Google')).toBeTruthy();
    expect(screen.getByText('Apple')).toBeTruthy();
    expect(screen.getByText('Weiter mit Google')).toBeTruthy();
    expect(screen.getByText('Weiter mit Apple')).toBeTruthy();
  });

  it('clicking a provider button calls doLogin with the provider id', () => {
    const app = makeApp({ providers: [googleProvider] });
    render(<Login />);
    const googleBtn = screen.getByText('Google').closest('button');
    expect(googleBtn).toBeTruthy();
    fireEvent.click(googleBtn!);
    expect(app.doLogin).toHaveBeenCalledWith('google');
  });

  it('shows a spinner when busy matches the provider id', () => {
    makeApp({ providers: [googleProvider], busy: 'login:google' });
    render(<Login />);
    // Spinner renders with role="status"
    expect(screen.getByRole('status')).toBeTruthy();
  });

  it('does not show a spinner when busy does not match provider id', () => {
    makeApp({ providers: [googleProvider], busy: null });
    render(<Login />);
    expect(screen.queryByRole('status')).toBeNull();
  });

  it('provider button is disabled when that provider is busy', () => {
    makeApp({ providers: [googleProvider], busy: 'login:google' });
    render(<Login />);
    const googleBtn = screen.getByText('Google').closest('button');
    expect(googleBtn).toBeDisabled();
  });

  it('provider button is enabled when a different provider is busy', () => {
    makeApp({ providers: [googleProvider, appleProvider], busy: 'login:apple' });
    render(<Login />);
    const googleBtn = screen.getByText('Google').closest('button');
    expect(googleBtn).not.toBeDisabled();
  });

  it('shows no error banner when state.error is null', () => {
    makeApp();
    render(<Login />);
    expect(screen.queryByRole('alert')).toBeNull();
  });

  it('shows an error banner with the message when state.error is set', () => {
    makeApp({ error: 'Anmeldung fehlgeschlagen.' });
    render(<Login />);
    expect(screen.getByRole('alert').textContent).toContain('Anmeldung fehlgeschlagen.');
  });

  it('renders hint text (auth.loginHint key)', () => {
    makeApp();
    render(<Login />);
    // The hint text is inside a div alongside a <br/> and loginHintNote, so we
    // use getAllByText with a substring matcher to locate it.
    expect(screen.getByText((content) => content.includes('Anmeldung über deinen Identity-Provider.'))).toBeTruthy();
  });
});
