import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { AppearancePanel } from './AppearancePanel';
import { LocaleProvider } from '@/i18n/LocaleProvider';
import type { AppContextValue } from '@/context/AppContext';

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

function makeApp(overrides: Record<string, unknown> = {}) {
  return {
    state: { primaryColor: '#1565C0', colorScheme: 'system' as const, ...overrides },
    setColorScheme: vi.fn(),
  } as unknown as AppContextValue;
}

describe('AppearancePanel', () => {
  it('renders color scheme buttons (system/light/dark)', () => {
    render(<AppearancePanel app={makeApp()} />, { wrapper: LocaleProvider });
    const btns = document.querySelectorAll('button');
    expect(btns.length).toBeGreaterThan(0);
  });

  it('clicking light scheme calls setColorScheme', () => {
    const app = makeApp();
    render(<AppearancePanel app={app} />, { wrapper: LocaleProvider });
    fireEvent.click(screen.getByText('Hell'));
    expect(app.setColorScheme).toHaveBeenCalledWith('light');
  });

  it('clicking dark scheme calls setColorScheme', () => {
    const app = makeApp();
    render(<AppearancePanel app={app} />, { wrapper: LocaleProvider });
    fireEvent.click(screen.getByText('Dunkel'));
    expect(app.setColorScheme).toHaveBeenCalledWith('dark');
  });

  it('exposes the selected scheme via aria-pressed for screen-reader users', () => {
    render(<AppearancePanel app={makeApp()} />, { wrapper: LocaleProvider });
    // makeApp defaults state.colorScheme to 'system'.
    expect(screen.getByText('Automatisch').closest('button')).toHaveAttribute('aria-pressed', 'true');
    expect(screen.getByText('Hell').closest('button')).toHaveAttribute('aria-pressed', 'false');
    expect(screen.getByText('Dunkel').closest('button')).toHaveAttribute('aria-pressed', 'false');
  });

  it('renders a language switcher with all supported languages', () => {
    render(<AppearancePanel app={makeApp()} />, { wrapper: LocaleProvider });
    expect(screen.getByText('Sprache')).toBeTruthy();
    expect(screen.getByText('Deutsch')).toBeTruthy();
    expect(screen.getByText('English')).toBeTruthy();
  });
});
