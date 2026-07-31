import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { SettingsPage } from './SettingsPage';

vi.mock('@/context/AppContext', () => ({
  useApp: vi.fn(),
}));

vi.mock('@/layouts/useCompact', () => ({
  useCompact: vi.fn(),
}));

vi.mock('./settingsCategories', () => ({
  SETTINGS_CATEGORIES: [
    {
      key: 'profile',
      labelKey: 'settings.category.profile',
      icon: 'person',
      Component: () => <div data-testid="panel-profile">Profile panel</div>,
    },
    {
      key: 'appearance',
      labelKey: 'settings.category.appearance',
      icon: 'palette',
      Component: () => <div data-testid="panel-appearance">Appearance panel</div>,
    },
    {
      key: 'notifications',
      labelKey: 'settings.category.notifications',
      icon: 'notifications',
      Component: () => <div data-testid="panel-notifications">Notifications panel</div>,
    },
    {
      key: 'privacy',
      labelKey: 'settings.category.privacy',
      icon: 'privacy_tip',
      Component: () => <div data-testid="panel-privacy">Privacy panel</div>,
    },
    {
      key: 'legal',
      labelKey: 'settings.category.legal',
      icon: 'gavel',
      Component: () => <div data-testid="panel-legal">Legal panel</div>,
    },
  ],
}));

import { useApp } from '@/context/AppContext';
import { useCompact } from '@/layouts/useCompact';
const mockUseApp = vi.mocked(useApp);
const mockUseCompact = vi.mocked(useCompact);

function makeApp() {
  const app = { state: { primaryColor: '#1565C0' }, logout: vi.fn() };
  mockUseApp.mockReturnValue(app as unknown as ReturnType<typeof useApp>);
  return app;
}

describe('SettingsPage — desktop', () => {
  it('renders all 5 categories plus Logout', () => {
    mockUseCompact.mockReturnValue(false);
    makeApp();
    render(<SettingsPage />);
    expect(screen.getByText('Profil')).toBeTruthy();
    expect(screen.getByText('Darstellung & Sprache')).toBeTruthy();
    expect(screen.getByText('Benachrichtigungen')).toBeTruthy();
    expect(screen.getByText('Datenschutz')).toBeTruthy();
    expect(screen.getByText('Rechtliches')).toBeTruthy();
    expect(screen.getByText('Abmelden')).toBeTruthy();
  });

  it('shows the first category panel by default', () => {
    mockUseCompact.mockReturnValue(false);
    makeApp();
    render(<SettingsPage />);
    expect(screen.getByTestId('panel-profile')).toBeTruthy();
  });

  it('clicking a category switches the content pane', () => {
    mockUseCompact.mockReturnValue(false);
    makeApp();
    render(<SettingsPage />);
    fireEvent.click(screen.getByText('Benachrichtigungen'));
    expect(screen.getByTestId('panel-notifications')).toBeTruthy();
    expect(screen.queryByTestId('panel-profile')).toBeNull();
  });

  it('clicking Abmelden calls logout', () => {
    mockUseCompact.mockReturnValue(false);
    const app = makeApp();
    render(<SettingsPage />);
    fireEvent.click(screen.getByText('Abmelden'));
    expect(app.logout).toHaveBeenCalledTimes(1);
  });
});

describe('SettingsPage — mobile', () => {
  it('shows only the category list initially, no panel', () => {
    mockUseCompact.mockReturnValue(true);
    makeApp();
    render(<SettingsPage />);
    expect(screen.getByText('Profil')).toBeTruthy();
    expect(screen.queryByTestId('panel-profile')).toBeNull();
  });

  it('tapping a category shows only that category panel', () => {
    mockUseCompact.mockReturnValue(true);
    makeApp();
    render(<SettingsPage />);
    fireEvent.click(screen.getByText('Datenschutz'));
    expect(screen.getByTestId('panel-privacy')).toBeTruthy();
    expect(screen.queryByText('Profil')).toBeNull();
  });

  it('back button returns to the category list', () => {
    mockUseCompact.mockReturnValue(true);
    makeApp();
    render(<SettingsPage />);
    fireEvent.click(screen.getByText('Datenschutz'));
    expect(screen.getByTestId('panel-privacy')).toBeTruthy();
    fireEvent.click(screen.getByText('Einstellungen'));
    expect(screen.queryByTestId('panel-privacy')).toBeNull();
    expect(screen.getByText('Profil')).toBeTruthy();
  });
});
