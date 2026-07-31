import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { NotificationsPanel } from './NotificationsPanel';
import type { AppContextValue } from '@/context/AppContext';

vi.mock('@/features/notifications', async (importOriginal) => {
  const mod = await importOriginal<typeof import('@/features/notifications')>();
  return { ...mod, usePushActions: vi.fn(), usePushPreferencesActions: vi.fn() };
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

import { usePushActions, usePushPreferencesActions } from '@/features/notifications';
const mockUsePushActions = vi.mocked(usePushActions);
const mockUsePushPreferencesActions = vi.mocked(usePushPreferencesActions);

function makeApp(overrides: Record<string, unknown> = {}) {
  return {
    state: { primaryColor: '#1565C0', activeTeamId: 'team-1', ...overrides },
    api: {},
    toastMsg: vi.fn(),
    activeTeam: vi.fn().mockReturnValue({ name: 'FC Testverein' }),
  } as unknown as AppContextValue;
}

// Default: push unsupported (jsdom has no serviceWorker/PushManager), matching
// production behavior when the browser can't do Web Push -- the push section
// (and, transitively, the per-team category panel) doesn't render at all.
// Individual tests override this to exercise the supported/subscribed paths.
beforeEach(() => {
  mockUsePushActions.mockReturnValue({
    support: 'unsupported',
    subscribed: null,
    busy: false,
    enablePush: vi.fn(),
    disablePush: vi.fn(),
  });
  mockUsePushPreferencesActions.mockReturnValue({
    prefs: { attendance: true, events: true, news: true, polls: true, absence: true },
    isLoading: false,
    busy: false,
    setCategory: vi.fn(),
  });
});

describe('NotificationsPanel', () => {
  it('shows neither the push toggle nor the per-team category panel when push is unsupported', () => {
    render(<NotificationsPanel app={makeApp()} />);
    expect(screen.queryByText('Web-Push-Benachrichtigungen')).toBeNull();
    expect(screen.queryByText('Push-Benachrichtigungen anpassen')).toBeNull();
  });

  it('shows the push toggle but not the category panel before the browser is subscribed', () => {
    mockUsePushActions.mockReturnValue({
      support: 'supported',
      subscribed: false,
      busy: false,
      enablePush: vi.fn(),
      disablePush: vi.fn(),
    });
    render(<NotificationsPanel app={makeApp()} />);
    expect(screen.getAllByText('Web-Push-Benachrichtigungen').length).toBeGreaterThan(0);
    expect(screen.queryByText('Push-Benachrichtigungen anpassen')).toBeNull();
  });

  it('shows the category panel with all five toggles once subscribed', () => {
    mockUsePushActions.mockReturnValue({
      support: 'supported',
      subscribed: true,
      busy: false,
      enablePush: vi.fn(),
      disablePush: vi.fn(),
    });
    render(<NotificationsPanel app={makeApp()} />);
    expect(screen.getByText('Push-Benachrichtigungen anpassen')).toBeTruthy();
    expect(screen.getByText('Rückmeldungen (Zu-/Absagen)')).toBeTruthy();
    expect(screen.getByText('Termine (neu, geändert, abgesagt)')).toBeTruthy();
    expect(screen.getByText('Neuigkeiten')).toBeTruthy();
    expect(screen.getByText('Umfragen')).toBeTruthy();
    expect(screen.getByText('Abwesenheiten')).toBeTruthy();
  });

  it('clicking a toggle calls setCategory with the flipped value', () => {
    mockUsePushActions.mockReturnValue({
      support: 'supported',
      subscribed: true,
      busy: false,
      enablePush: vi.fn(),
      disablePush: vi.fn(),
    });
    const setCategory = vi.fn();
    mockUsePushPreferencesActions.mockReturnValue({
      prefs: { attendance: true, events: true, news: true, polls: true, absence: true },
      isLoading: false,
      busy: false,
      setCategory,
    });
    render(<NotificationsPanel app={makeApp()} />);
    fireEvent.click(screen.getByText('Neuigkeiten').closest('button')!);
    expect(setCategory).toHaveBeenCalledWith('news', false);
  });

  it('reflects a disabled category via aria-checked=false', () => {
    mockUsePushActions.mockReturnValue({
      support: 'supported',
      subscribed: true,
      busy: false,
      enablePush: vi.fn(),
      disablePush: vi.fn(),
    });
    mockUsePushPreferencesActions.mockReturnValue({
      prefs: { attendance: true, events: true, news: false, polls: true, absence: true },
      isLoading: false,
      busy: false,
      setCategory: vi.fn(),
    });
    render(<NotificationsPanel app={makeApp()} />);
    expect(screen.getByText('Neuigkeiten').closest('button')).toHaveAttribute('aria-checked', 'false');
    expect(screen.getByText('Umfragen').closest('button')).toHaveAttribute('aria-checked', 'true');
  });

  it('clicking the push toggle enables push when not subscribed', () => {
    const enablePush = vi.fn();
    mockUsePushActions.mockReturnValue({
      support: 'supported',
      subscribed: false,
      busy: false,
      enablePush,
      disablePush: vi.fn(),
    });
    render(<NotificationsPanel app={makeApp()} />);
    fireEvent.click(screen.getByRole('button', { pressed: false }));
    expect(enablePush).toHaveBeenCalledTimes(1);
  });
});
